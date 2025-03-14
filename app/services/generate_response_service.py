import httpx
import json
import re
import os
import logging
from typing import List, Optional, Tuple
from pydantic import SecretStr, HttpUrl
from fastapi import HTTPException
from app.utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    separate_file_paths_by_type,
    send_prompt_to_openai_streaming,
    FileContentResponse,
    count_tokens,
    add_sys_path,
    check_token_limit,
)
from lib.utils.enums import (
    SearchMode,
)

from app.models.responses import (
    FileSearchResponse,
)

# Define token limits for different models
TOKEN_LIMITS = {
    "gpt-4o": 128000,
    "gpt-4o-mini": 128000,
}

# Add the path to sys.path for importing custom modules
path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval/lib')
logger = logging.getLogger("uvicorn")
logger.info("Adding to sys.path: %s", path_to_add)

try:
    with add_sys_path(path_to_add):
        logger.info("Imports successful.")
except ModuleNotFoundError as e:
    logger.error(f"ModuleNotFoundError: {e}")
    logger.error("Failed to import the module. Please check the paths and directory structure.")

async def generate_response(
    prompt: str,
    project: str,
    mode: str,
    model: str,
    match_strength: str,
    llm_model_api_key: str,
    codehost_api_key: Optional[SecretStr],
    codehost_url: HttpUrl,
    ignore_files: List[str],
):
    if model not in TOKEN_LIMITS:
        yield {"error": "Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'."}
        return

    if match_strength not in ["high", "mid", "low"]:
        yield {"error": "Invalid match strength selected. Choose either 'high', 'mid', or 'low'."}
        return

    if not await check_token_limit(prompt, model, TOKEN_LIMITS):
        error_message = (
            f"Prompt token limit exceeded for the selected model. "
            f"Limit: {max_tokens}, Count: {token_count}. "
            f"Please reduce the length of your prompt."
        )
        logger.error(error_message)
        yield {"error": error_message}
        return

    base_url = "http://commit-file-retrieval:5070"
    infer_file_url = f"{base_url}/infer-file/"
    get_file_summary_url = f"{base_url}/get-file-summary/?project_name={project}"
    test_pull_access_url = f"{base_url}/test-pull-access/"

    try:
        async with httpx.AsyncClient(timeout=httpx.Timeout(1200, read=1200.0)) as client:
            params = {
                'project_name': project,
                'codehost_api_key': codehost_api_key.get_secret_value() if codehost_api_key else None,
                'codehost_url': codehost_url
            }
            pull_access_response = await client.post(test_pull_access_url, params=params)
            pull_access_response.raise_for_status()
            pull_access_data = pull_access_response.json()
            if not pull_access_data.get('pull_access', False):
                raise HTTPException(status_code=403, detail="Pull access denied.")

            if mode == SearchMode.pure_chat:
                combined_prompt = prompt
                retrieved_file_paths = []
            else:
                params = {
                    "prompt": prompt,
                    "project": project,
                    "mode": mode,
                    "model": model,
                    "match_strength": match_strength,
                    "llm_model_api_key": llm_model_api_key,
                    "embeddings_model_api_key": llm_model_api_key, # We will change it to refer to embedding_model_api_key
                    "ignore_files": ignore_files,
                }
                response = await client.post(infer_file_url, json=params)
                response.raise_for_status()
                list_file_search_response = [FileSearchResponse(**item) for item in response.json()]

                # Separate file paths by type
                commit_paths, file_paths = separate_file_paths_by_type(list_file_search_response)

                # Get top 5 commit paths
                top_commit_paths = commit_paths[:5]
                logger.info(f"Top 5 commit paths before removing duplicates: {top_commit_paths}")

                # Get top 5 file paths
                top_file_paths = file_paths[:5]
                logger.info(f"Top 5 file paths before removing duplicates: {top_file_paths}")
                list_file_path_entry = top_commit_paths.copy()
                list_file_path_entry.extend(top_file_paths)
                logger.info(f"list of file paths before removing duplicates: {list_file_path_entry}")
                if list_file_path_entry:
                    list_file_path_entry = await remove_duplicate_file_paths(list_file_path_entry)
                    logger.info(f"list of file paths after removing duplicates: {list_file_path_entry}")
                    list_file_path_entry = [entry for entry in list_file_path_entry if entry.path not in ignore_files]
                    logger.info(f"list of file paths after transformation: {list_file_path_entry}")
                    file_paths_payload = [entry.dict() for entry in list_file_path_entry]
                else:
                    file_paths_payload = []
                logger.info(f"Payload for retrieve-file-contents: {file_paths_payload}")

                if not file_paths_payload:
                    yield {"machtiani": "no files found"}
                    return


                file_summaries = {}
                file_paths_to_summarize = [entry["path"] for entry in file_paths_payload]

                try:
                    file_summary_response = await client.get(
                        get_file_summary_url,
                        params={"file_paths": file_paths_to_summarize, "project_name": project}
                    )
                    file_summary_response.raise_for_status()

                    summary_data = file_summary_response.json()

                    for summary in summary_data:
                        file_path = summary["file_path"]
                        file_summaries[file_path] = summary["summary"]

                except httpx.HTTPStatusError as exc:
                    if exc.response.status_code == 404:
                        logger.warning(f"No summary found for one or more file paths.")
                    else:
                        logger.error(f"HTTP status error: {exc.response.json()}")
                        raise

                if not file_summaries:
                    logger.error("No summaries found for any of the files.")
                    yield {"error": "No relevant file summaries found."}
                    return

                summary_prompt = (
                    f"Here are the file summaries:\n\n{json.dumps(file_summaries, indent=2)}\n\n"
                    f"Based on these summaries, return only the paths directly relevant to answer the following prompt:\n\n"
                    f"{prompt}\n\n"
                    f"Encapsulate the relevant paths between `---` markers.\n"
                    f"Example format:\n---\n/path/to/relevant_file1\n/path/to/relevant_file2\n---"
                )

                # Collect tokens from streaming response
                response_tokens = []
                async for token_json in send_prompt_to_openai_streaming(summary_prompt, llm_model_api_key, model):
                    token_data = json.loads(token_json)
                    token = token_data.get("token", "")
                    response_tokens.append(token)

                llm_model_response = ''.join(response_tokens)

                match = re.search(r"---\s*(.*?)\s*---", llm_model_response, re.DOTALL)

                if match:
                    relevant_file_paths_str = match.group(1).strip()
                    relevant_file_paths = [line.lstrip("/").strip() for line in relevant_file_paths_str.splitlines() if line.strip()]
                    logger.info(f"Relevant file paths returned from OpenAI: {relevant_file_paths}")
                else:
                    logger.error(f"Failed to extract relevant paths from OpenAI response: {llm_model_response}")
                    yield {"error": "Invalid response format from OpenAI API."}
                    return

                if not relevant_file_paths:
                    yield {"error": "No relevant file paths found from OpenAI response."}
                    return

                relevant_file_paths_payload = [
                    entry for entry in file_paths_payload if entry["path"] in relevant_file_paths
                ]

                if not relevant_file_paths_payload:
                    logger.error("No relevant entries found in the original payload after filtering.")
                    yield {"error": "No relevant entries found in the original payload after filtering."}
                    return

                content_response = await client.post(
                    f"{base_url}/retrieve-file-contents/",
                    json={
                        "project_name": project,
                        "file_paths": relevant_file_paths_payload,
                        "ignore_files": ignore_files
                    }
                )
                content_response.raise_for_status()

                file_content_response = FileContentResponse(**content_response.json())
                retrieved_file_paths = file_content_response.retrieved_file_paths

                # Convert FilePathEntry objects to string paths and filter out duplicates
                top_commit_paths_to_add = [entry.path for entry in top_commit_paths if entry.path not in retrieved_file_paths]

                if top_commit_paths_to_add:
                    logger.info(f"Top commit paths added: {top_commit_paths_to_add}")

                # Prepend the unique commit paths to the retrieved_file_paths list
                retrieved_file_paths = top_commit_paths_to_add + retrieved_file_paths

                combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
                for path, content in file_content_response.contents.items():
                    combined_prompt += f"\n--- {path} ---\n{content}\n"

            if not await check_token_limit(combined_prompt, model, TOKEN_LIMITS):
                error_message = (
                    f"Token limit exceeded for the selected model. "
                    f"Limit: {max_tokens}, Count: {token_count}. "
                    f"Please reduce the length of your prompt or the number of retrieved contents."
                )
                logger.error(error_message)
                yield {"error": error_message}
                return

            # Yield retrieved_file_paths if any
            if retrieved_file_paths:
                yield {"retrieved_file_paths": retrieved_file_paths}

            # Stream tokens from OpenAI response
            async for token_json in send_prompt_to_openai_streaming(combined_prompt, llm_model_api_key, model):
                yield json.loads(token_json)

    except httpx.RequestError as exc:
        logger.error(f"Request error: {exc}")
        yield {"error": f"Error connecting to commit-file-retrieval service: {exc}"}
    except httpx.HTTPStatusError as exc:
        logger.error(f"HTTP status error: {exc.response.json()}")
        yield {"error": f"Error response from commit-file-retrieval service: {exc.response.json()}"}
    except Exception as e:
        logger.exception("Unexpected error occurred")
        yield {"error": f"An unexpected error occurred: {str(e)}"}

