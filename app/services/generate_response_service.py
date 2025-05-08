import httpx
import json
import re
import os
import logging
from typing import List, Optional
from pydantic import SecretStr, HttpUrl
from fastapi import HTTPException
import asyncio
from app.utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    separate_file_paths_by_type,
    FileContentResponse,
    FilePathEntry,
    FileSearchResponse,
    SearchMode,
    count_tokens,
    add_sys_path,
    check_token_limit,
    adjusted_file_scores,
    top_n_files,
)
from lib.ai.llm_model import LlmModel

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
    # The function treats answer-only mode the same as default mode
    # The answer-only handling is managed client-side in the Go code

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
            llm_model_base_url_to_use = llm_model_base_url_other if llm_model_base_url_other else llm_model_base_url

            llm_model_api_key_to_use = llm_model_api_key_other if llm_model_api_key_other else llm_model_api_key



            logger.info(f"Using LLM model URL: {llm_model_base_url_to_use}")
            logger.info(f"Using LLM model API key: {llm_model_api_key_to_use}")

            llm_model = LlmModel(api_key=llm_model_api_key_to_use, base_url=str(llm_model_base_url_to_use), model=model)

            if mode == SearchMode.pure_chat:
                combined_prompt = prompt
                retrieved_file_paths = []
            else:

                infer_params = {
                    "prompt": prompt,
                    "project": project,
                    "mode": mode,
                    # model will be used for file localization inference, as infer uses a local hosted embedding model.
                    "model": model,
                    "match_strength": match_strength,
                    "llm_model_api_key": llm_model_api_key_to_use,
                    "llm_model_base_url": str(llm_model_base_url_to_use),
                    "embeddings_model_api_key": llm_model_api_key_to_use, # We will change it to refer to embedding_model_api_key
                    "embeddings_model": "all-MiniLM-L6-v2",
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

                # Adjust number of files based on match_strength
                num_commit_files = 3
                num_file_files = 0
                num_localization_files = 3
                if match_strength == "mid":
                    num_commit_files = 5
                    num_file_files = 0
                    num_localization_files = 5
                elif match_strength == "low":
                    num_commit_files = 10
                    num_file_files = 0
                    num_localization_files = 10

                # Get top n commit paths
                scores = adjusted_file_scores(list_file_search_response)  # Dict[str, float]
                if not scores:                          # no commit hits at all
                    logger.critical(f"adjusted scoring of file paths failed")
                    top_commit_paths = commit_paths[:1] #fall back to old scoring if fails.
                else:
                    top_commit_paths = [
                        FilePathEntry(path=p)               # return proper object, not bare str
                        for p, _ in top_n_files(scores, num_commit_files)
                    ]
                logger.info(f"Top {len(top_commit_paths)} commit paths before dedup: {top_commit_paths}")

                logger.debug(f"file scores for commits:\n\n {scores}")

                # Get top n file paths
                top_file_paths = file_paths[:num_file_files]
                logger.info(f"Top {len(top_file_paths)} file paths before removing duplicates: {top_file_paths}\n")

                # Get top n localization paths
                top_localization_paths = localization_paths[:num_localization_files]
                logger.info(f"Top {len(top_localization_paths)} localization paths before removing duplicates: {top_localization_paths}\n")

                list_file_path_entry = top_commit_paths.copy()
                list_file_path_entry.extend(top_file_paths)
                list_file_path_entry.extend(top_localization_paths)
                logger.info(f"list of file paths before removing duplicates: {list_file_path_entry}")

                if list_file_path_entry:
                    # dedupe & filter
                    list_file_path_entry = await remove_duplicate_file_paths(list_file_path_entry)
                    list_file_path_entry = [
                        entry for entry in list_file_path_entry
                        if entry.path not in ignore_files
                    ]
                    # prefer localization, else fall back to everything
                    payload_entries = top_localization_paths or list_file_path_entry
                    file_paths_payload = [entry.dict() for entry in payload_entries]
                else:
                    file_paths_payload = []


                logger.info(f"Payload for retrieve-file-contents: {file_paths_payload}")

                if not file_paths_payload:
                    yield {"machtiani": "no files found"}
                    return

                content_response = await client.post(
                    f"{base_url}/retrieve-file-contents/",
                    json={
                        "project_name": project,
                        "file_paths": file_paths_payload,
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

            if mode == SearchMode.default:
                # Notify the client that we're about to call file-edit and new-files in parallel
                yield {
                    "event": "file_edit_start",
                    "message": "Waiting on file-edit/new-files requests…",
                    "file_count": len(retrieved_file_paths),
                    "retrieved_file_paths": retrieved_file_paths,
                }
                file_edit_url = f"{base_url}/file-edit/"
                updated_contents = {}
                async with httpx.AsyncClient(timeout=600) as edit_client:
                    # Create file-edit tasks for each file
                    file_edit_tasks = []
                    for file_path in retrieved_file_paths:
                        payload = {
                            "project": project,
                            "file_path": file_path,
                            "instructions": final_response_text,
                            "llm_model_api_key": llm_model_api_key_to_use,
                            "llm_model_base_url": str(llm_model_base_url_to_use),
                            "model": model,
                            "ignore_files": ignore_files or []
                        }
                        file_edit_tasks.append(edit_client.post(file_edit_url, json=payload))

                    # Create new-files task (just one)
                    new_files_url = f"{base_url}/new-files/"
                    new_files_payload = {
                        "project": project,
                        "instructions": final_response_text,
                        "llm_model_api_key": llm_model_api_key_to_use,
                        "llm_model_base_url": str(llm_model_base_url_to_use),
                        "model": model,
                        "ignore_files": ignore_files or []
                    }
                    new_files_task = edit_client.post(new_files_url, json=new_files_payload)

                    # Combine all tasks
                    all_tasks = file_edit_tasks + [new_files_task]

                    # Run all concurrently
                    responses = await asyncio.gather(*all_tasks, return_exceptions=True)

                    # The first N are for file edits, the last is for new-files
                    for i, resp in enumerate(responses):
                        try:
                            if isinstance(resp, Exception):
                                raise resp

                            resp.raise_for_status()
                            resp_json = resp.json()

                            # File edit responses
                            if i < len(file_edit_tasks):
                                file_path = retrieved_file_paths[i]
                                errors = resp_json.get("errors", [])
                                if errors:
                                    logger.warning(f"[file-edit] Skipping update for {file_path} due to errors: {errors}")
                                    continue
                                updated_contents[file_path] = {
                                    "updated_content": resp_json.get("updated_content", ""),
                                    "errors": errors,
                                }
                            # New-files response (last task)
                            else:
                                logger.info(f"[new-files] Response status: {resp.status_code}")
                                if resp_json and isinstance(resp_json, dict):
                                    errors = resp_json.get("errors", [])
                                    if errors:
                                        logger.warning(f"[new-files] Errors in response: {errors}")
                                    new_content = resp_json.get("new_content", {})
                                    logger.info(f"[new-files] Received {len(new_content)} new file suggestions")
                                    if new_content and not any(errors):
                                        logger.debug(f"[new-files] New file paths: {list(new_content.keys())}")
                                        yield {"new_files": resp_json}
                                    else:
                                        logger.info("[new-files] No valid new files to suggest or errors present")
                                else:
                                    logger.warning("[new-files] Empty response from new-files endpoint")

                        except Exception as e:
                            if i < len(file_edit_tasks):
                                file_path = retrieved_file_paths[i]
                                logger.error(f"[file-edit] Error editing {file_path}: {e}")
                                updated_contents[file_path] = {
                                    "updated_content": f"[Error updating file: {e}]",
                                    "errors": [str(e)],
                                }
                            else:
                                logger.exception(f"[new-files] Unexpected error calling endpoint")
                                # Just log error; don't yield to client

                    # Yield updated file contents if any
                    if updated_contents:
                        logger.info(f"updated_file_contents: {updated_contents}")
                        yield {"updated_file_contents": updated_contents}

    except httpx.RequestError as exc:
        logger.error(f"Request error: {exc}")
        yield {"error": f"Error connecting to commit-file-retrieval service: {exc}"}
    except httpx.HTTPStatusError as exc:
        logger.error(f"HTTP status error: {exc.response.json()}")
        yield {"error": f"Error response from commit-file-retrieval service: {exc.response.json()}"}
    except Exception as e:
        logger.exception("Unexpected error occurred")
        yield {"error": f"An unexpected error occurred: {str(e)}"}
