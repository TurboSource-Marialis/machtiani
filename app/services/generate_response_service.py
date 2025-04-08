import httpx
import json
import re
import os
import logging
from typing import List, Optional
from pydantic import SecretStr, HttpUrl
from fastapi import HTTPException
from app.utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    separate_file_paths_by_type,
    FileContentResponse,
    FileSearchResponse,
    SearchMode,
    count_tokens,
    add_sys_path,
    check_token_limit,
)
from lib.ai.llm_model import LlmModel

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Define token limits for different models
MAX_TOKENS = 128000

async def generate_response(
    prompt: str,
    project: str,
    mode: str,
    model: str,
    match_strength: str,
    llm_model_api_key: str,
    llm_model_base_url: HttpUrl,
    codehost_api_key: Optional[SecretStr],
    codehost_url: HttpUrl,
    ignore_files: List[str],
    head_commit_hash: str,
    llm_model_base_url_other: Optional[str] = None,
    llm_model_api_key_other: Optional[str] = None,
):

    logger.debug("Begin generate_response service")
    logger.debug(f"Input parameters: prompt ({len(prompt)} chars), project: {project}, mode: {mode}, model: {model}, match_strength: {match_strength}")

    if match_strength not in ["high", "mid", "low"]:
        yield {"error": "Invalid match strength selected. Choose either 'high', 'mid', or 'low'."}
        return

    if not await check_token_limit(prompt, model, MAX_TOKENS):
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
            logger.debug("Calling pull access check with params: %s", params)
            pull_access_response = await client.post(test_pull_access_url, params=params)
            pull_access_response.raise_for_status()
            pull_access_data = pull_access_response.json()
            logger.debug("Pull access response: %s", pull_access_data)
            if not pull_access_data.get('pull_access', False):
                raise HTTPException(status_code=403, detail="Pull access denied.")

            # Safely determine which API key to use
            llm_model_base_url_to_use = llm_model_base_url_other if llm_model_base_url_other is not None else llm_model_base_url


            # Only use llm_model_api_key_other if it has a valid string value
            if llm_model_api_key_other and isinstance(llm_model_api_key_other, str) and llm_model_api_key_other.strip():
                llm_model_api_key_to_use = llm_model_api_key_other
                if model == 'reason':
                    model = "deepseek-reasoner"
            else:
                llm_model_api_key_to_use = llm_model_api_key
                if model == 'reason':
                    model = "o3-mini"

            # Initialize LlmModel with the selected API key
            logger.debug(f"Using LLM model URL: {llm_model_base_url_to_use} and API key: {llm_model_api_key_to_use}")
            llm_model = LlmModel(api_key=llm_model_api_key_to_use, base_url=str(llm_model_base_url_to_use), model=model)

            if mode == SearchMode.pure_chat:
                combined_prompt = prompt
                retrieved_file_paths = []
            else:
                infer_params = {
                    "prompt": prompt,
                    "project": project,
                    "mode": mode,
                    "model": "gpt-4o-mini", # model not actuall needed as infer uses a method of GitCommitManager where its not needed. It only needs to do embeddings.
                    "match_strength": match_strength,
                    "llm_model_api_key": llm_model_api_key,
                    "llm_model_base_url": str(llm_model_base_url),
                    "embeddings_model_api_key": llm_model_api_key, # We will change it to refer to embedding_model_api_key
                    "ignore_files": ignore_files,
                    "head": head_commit_hash,
                }

                logger.debug("Calling infer-file with params: %s", infer_params)
                response = await client.post(infer_file_url, json=infer_params)
                response.raise_for_status()
                list_file_search_response = [FileSearchResponse(**item) for item in response.json()]
                logger.debug("Response from infer-file: %s", list_file_search_response)

                # Separate file paths by type
                commit_paths, file_paths, localization_paths = separate_file_paths_by_type(list_file_search_response)

                # Get top 5 commit paths
                top_commit_paths = commit_paths[:5]
                logger.info(f"Top 5 commit paths before removing duplicates: {top_commit_paths}\n")

                # Get top 5 file paths
                top_file_paths = file_paths[:5]
                logger.info(f"Top 5 file paths before removing duplicates: {top_file_paths}\n")

                # Get top 5 localization paths
                top_localization_paths = localization_paths[:5]
                logger.info(f"Top 5 localization paths before removing duplicates: {top_localization_paths}\n")

                list_file_path_entry = top_commit_paths.copy()
                list_file_path_entry.extend(top_file_paths)
                list_file_path_entry.extend(top_localization_paths)
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
                async for token_json in llm_model.send_prompt_streaming(summary_prompt):
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


                seen = set()
                deduped_paths = []
                for path in retrieved_file_paths:
                    if path not in seen:
                        deduped_paths.append(path)
                        seen.add(path)
                retrieved_file_paths = deduped_paths

                combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
                for path, content in file_content_response.contents.items():
                    combined_prompt += f"\n--- {path} ---\n{content}\n"

            if not await check_token_limit(combined_prompt, model, MAX_TOKENS):
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

            # Accumulate tokens from OpenAI response
            response_tokens = []
            async for token_json in llm_model.send_prompt_streaming(combined_prompt):
                token_data = json.loads(token_json)
                token = token_data.get("token", "")
                response_tokens.append(token)
                yield token_data  # Stream tokens as before

            final_response_text = ''.join(response_tokens)

            # Call file-edit for each retrieved file path, log response
            file_edit_url = f"{base_url}/file-edit/"
            async with httpx.AsyncClient(timeout=600) as edit_client:
                for file_path in retrieved_file_paths:
                    payload = {
                        "project": project,
                        "file_path": file_path,
                        "instructions": final_response_text,
                        "llm_model_api_key": llm_model_api_key_other,
                        "llm_model_base_url": str(llm_model_base_url_other),
                        "model": model,
                        "ignore_files": ignore_files or []
                    }
                    try:
                        resp = await edit_client.post(file_edit_url, json=payload)
                        resp.raise_for_status()
                        resp_json = resp.json()
                        logger.info(f"[file-edit] {file_path} response: {resp_json}")
                    except Exception as e:
                        logger.error(f"[file-edit] Error editing {file_path}: {e}")

    except httpx.RequestError as exc:
        logger.error(f"Request error: {exc}")
        yield {"error": f"Error connecting to commit-file-retrieval service: {exc}"}
    except httpx.HTTPStatusError as exc:
        logger.error(f"HTTP status error: {exc.response.json()}")
        yield {"error": f"Error response from commit-file-retrieval service: {exc.response.json()}"}
    except Exception as e:
        logger.exception("Unexpected error occurred")
        yield {"error": f"An unexpected error occurred: {str(e)}"}
