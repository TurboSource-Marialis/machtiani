import httpx
import json
import re
import os
import logging
from typing import List, Optional
from pydantic import SecretStr
from fastapi import HTTPException
from app.utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    send_prompt_to_openai,
    FileContentResponse,
    count_tokens,
    add_sys_path,
)
from utils.enums import (
    SearchMode,
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
    api_key: str,
    codehost_api_key: Optional[SecretStr],
    codehost_url: str,
    ignore_files: List[str],
):
    if model not in TOKEN_LIMITS:
        return {"error": "Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'."}

    if match_strength not in ["high", "mid", "low"]:
        return {"error": "Invalid match strength selected. Choose either 'high', 'mid', or 'low'."}

    base_url = "http://commit-file-retrieval:5070"
    infer_file_url = f"{base_url}/infer-file/"
    get_file_summary_url = f"{base_url}/get-file-summary/?project_name={project}"
    test_pull_access_url = f"{base_url}/test-pull-access/"

    try:
        async with httpx.AsyncClient(timeout=httpx.Timeout(1200, read=1200.0)) as client:
            params = {
                'project_name': project,
                'codehost_api_key': codehost_api_key,
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
                    "api_key": api_key,
                }
                response = await client.post(infer_file_url, json=params)
                response.raise_for_status()
                list_file_search_response = [FileSearchResponse(**item) for item in response.json()]

                list_file_path_entry = await aggregate_file_paths(list_file_search_response)
                list_file_path_entry = await remove_duplicate_file_paths(list_file_path_entry)
                list_file_path_entry = [entry for entry in list_file_path_entry if entry.path not in ignore_files]

                file_paths_payload = [entry.dict() for entry in list_file_path_entry]
                logger.info(f"Payload for retrieve-file-contents: {file_paths_payload}")

                if not file_paths_payload:
                    return {"machtiani": "no files found"}

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
                    return {"error": "No relevant file summaries found."}

                summary_prompt = (
                    f"Here are the file summaries:\n\n{json.dumps(file_summaries, indent=2)}\n\n"
                    f"Based on these summaries, return only the paths directly relevant to answer the following prompt:\n\n"
                    f"{prompt}\n\n"
                    f"Encapsulate the relevant paths between `---` markers.\n"
                    f"Example format:\n---\n/path/to/relevant_file1\n/path/to/relevant_file2\n---"
                )

                openai_response = await send_prompt_to_openai(summary_prompt, api_key, model)

                match = re.search(r"---\s*(.*?)\s*---", openai_response, re.DOTALL)

                if match:
                    relevant_file_paths_str = match.group(1).strip()
                    relevant_file_paths = [line.lstrip("/").strip() for line in relevant_file_paths_str.splitlines() if line.strip()]
                    logger.info(f"Relevant file paths returned from OpenAI: {relevant_file_paths}")
                else:
                    logger.error(f"Failed to extract relevant paths from OpenAI response: {openai_response}")
                    return {"error": "Invalid response format from OpenAI API."}

                if not relevant_file_paths:
                    return {"error": "No relevant file paths found from OpenAI response."}

                relevant_file_paths_payload = [
                    entry for entry in file_paths_payload if entry["path"] in relevant_file_paths
                ]

                if not relevant_file_paths_payload:
                    logger.error("No relevant entries found in the original payload after filtering.")
                    return {"error": "No relevant entries found in the original payload after filtering."}

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

                combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
                for path, content in file_content_response.contents.items():
                    combined_prompt += f"\n--- {path} ---\n{content}\n"

            token_count = await count_tokens(combined_prompt)
            max_tokens = TOKEN_LIMITS[model]
            logger.info(f"model: {model}, token count: {token_count}, max limit: {max_tokens}")

            if token_count > max_tokens:
                error_message = (
                    f"Token limit exceeded for the selected model. "
                    f"Limit: {max_tokens}, Count: {token_count}. "
                    f"Please reduce the length of your prompt or the number of retrieved contents."
                )
                logger.error(error_message)
                return {"error": error_message}

            openai_response = await send_prompt_to_openai(combined_prompt, api_key, model)

            return {"openai_response": openai_response, "retrieved_file_paths": retrieved_file_paths}

    except httpx.RequestError as exc:
        logger.error(f"Request error: {exc}")
        return {"error": f"Error connecting to commit-file-retrieval service: {exc}"}
    except httpx.HTTPStatusError as exc:
        logger.error(f"HTTP status error: {exc.response.json()}")
        return {"error": f"Error response from commit-file-retrieval service: {exc.response.json()}"}
    except Exception as e:
        logger.exception("Unexpected error occurred")
        return {"error": f"An unexpected error occurred: {str(e)}"}
