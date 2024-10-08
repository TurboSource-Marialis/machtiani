import httpx
import yaml
from fastapi import FastAPI, Body, Query, HTTPException
from pydantic import SecretStr
from typing import Optional, List, Dict, Union
from contextlib import asynccontextmanager
from .utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    send_prompt_to_openai,
    FileContentResponse,
    count_tokens,
    add_sys_path
)
import sys
import os
import re
import json
import logging

app = FastAPI()

# Define token limits for different models
TOKEN_LIMITS = {
    "gpt-4o": 128000,
    "gpt-4o-mini": 128000,
}

# Use the logger instead of print
logger = logging.getLogger("uvicorn")
logger.info("Application is starting up...")

path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval/lib')
logger.info("Adding to sys.path: %s", path_to_add)

try:
    with add_sys_path(path_to_add):
        from utils.enums import (
            SearchMode,
            MatchStrength,
            EmbeddingModel,
            FilePathEntry,
            FileSearchResponse,
            VCSType,
            AddRepositoryRequest,
            FetchAndCheckoutBranchRequest,
            FileContentResponse
        )
    logger.info("Imports successful.")
except ModuleNotFoundError as e:
    logger.error(f"ModuleNotFoundError: {e}")
    logger.error("Failed to import the module. Please check the paths and directory structure.")


@app.get("/generate-filename", response_model=str)
async def generate_filename(
    context: str = Query(..., description="Context to create filename"),
    api_key: str = Query(..., description="API key for OpenAI model"),
) -> str:
    filename_prompt = (
        f"Generate a unique filename for the following context: '{context}'.\n"
        "Respond ONLY with the filename in snake_case, wrapped in <filename> and </filename> tags.\n"
        "Do not include any other text or explanations.\n"
        "Example:\n"
        "<filename>example_filename</filename>"
    )

    response = await send_prompt_to_openai(filename_prompt, api_key, model="gpt-4o-mini")
    logger.info(f"OpenAI response: {response}")

    match = re.search(r"<filename>\s*(.*?)\s*</filename>", response, re.DOTALL | re.IGNORECASE)
    if not match:
        match = re.search(r"<\s*(.*?)\s*>", response)
    if match:
        filename = match.group(1).strip()
        return filename
    else:
        logger.error("Failed to extract filename from response.")
        raise HTTPException(status_code=400, detail="Invalid response format from OpenAI API.")

@app.post("/generate-response")
async def generate_response(
    prompt: str = Body(..., description="The prompt to search for"),
    project: str = Body(..., description="The project to search"),
    mode: str = Body(..., description="Search mode: chat, commit, or super"),
    model: str = Body(..., description="The embedding model used"),
    match_strength: str = Body(..., description="The strength of the match"),
    api_key: str = Body(..., description="API key for OpenAI model"),
    codehost_api_key: SecretStr = Body(..., description="Code host API key for authentication"),
):
    # Existing logic remains unchanged
    if model not in TOKEN_LIMITS:
        return {"error": "Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'."}

    if match_strength not in ["high", "mid", "low"]:
        return {"error": "Invalid match strength selected. Choose either 'high', 'mid', or 'low'."}

    # Define base URL for the commit-file-retrieval service
    base_url = "http://commit-file-retrieval:5070"

    infer_file_url = f"{base_url}/infer-file/"
    retrieve_file_contents_url = f"{base_url}/retrieve-file-contents/?project_name={project}"
    get_file_summary_url = f"{base_url}/get-file-summary/?project_name={project}"
    test_pull_access_url = f"{base_url}/test-pull-access/"

    ignore_files = []
    try:
        with open(".machtiani.ignore", "r") as f:
            ignore_files = [line.strip() for line in f if line.strip()]
    except FileNotFoundError:
        logger.warning("No .machtiani.ignore file found, proceeding without ignoring any files.")

    try:
        async with httpx.AsyncClient(timeout=httpx.Timeout(1200, read=1200.0)) as client:
            # Check for pull access before proceeding
            params = {
                'project_name': project,
                'codehost_api_key': codehost_api_key.get_secret_value()
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
                # Step 1: Retrieve file paths from the service
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

                # Step 2: Get file summaries for each retrieved file path
                file_summaries = {}
                file_paths_to_summarize = [entry["path"] for entry in file_paths_payload]  # Extract file paths

                try:
                    # Make a single request to get summaries for all file paths at once
                    file_summary_response = await client.get(
                        get_file_summary_url,
                        params={"file_paths": file_paths_to_summarize, "project_name": project}
                    )
                    file_summary_response.raise_for_status()

                    summary_data = file_summary_response.json()

                    for summary in summary_data:
                        file_path = summary["file_path"]
                        file_summaries[file_path] = summary["summary"]  # Store the summary in the dictionary

                except httpx.HTTPStatusError as exc:
                    if exc.response.status_code == 404:
                        logger.warning(f"No summary found for one or more file paths.")
                    else:
                        logger.error(f"HTTP status error: {exc.response.json()}")
                        raise

                if not file_summaries:
                    logger.error("No summaries found for any of the files.")
                    return {"error": "No relevant file summaries found."}

                # Step 3: Send the summaries to OpenAI for filtering relevant file paths
                summary_prompt = (
                    f"Here are the file summaries:\n\n{json.dumps(file_summaries, indent=2)}\n\n"
                    f"Based on these summaries, return only the paths directly relevant to answer the following prompt:\n\n"
                    f"{prompt}\n\n"
                    f"Encapsulate the relevant paths between `---` markers.\n"
                    f"Example format:\n---\n/path/to/relevant_file1\n/path/to/relevant_file2\n---"
                )

                # Send the summaries prompt to OpenAI
                openai_response = await send_prompt_to_openai(summary_prompt, api_key, model)

                # Step 4: Parse the OpenAI response for paths encapsulated between `---`
                match = re.search(r"---\s*(.*?)\s*---", openai_response, re.DOTALL)

                if match:
                    relevant_file_paths_str = match.group(1).strip()
                    relevant_file_paths = [line.strip() for line in relevant_file_paths_str.splitlines() if line.strip()]
                    logger.info(f"Relevant file paths returned from OpenAI: {relevant_file_paths}")
                else:
                    logger.error(f"Failed to extract relevant paths from OpenAI response: {openai_response}")
                    return {"error": "Invalid response format from OpenAI API."}

                if not relevant_file_paths:
                    return {"error": "No relevant file paths found from OpenAI response."}

                # Step 5: Filter original file paths payload to keep only relevant entries
                relevant_file_paths_payload = [
                    entry for entry in file_paths_payload if entry["path"] in relevant_file_paths
                ]

                if not relevant_file_paths_payload:
                    logger.error("No relevant entries found in the original payload after filtering.")
                    return {"error": "No relevant entries found in the original payload after filtering."}

                # Step 6: Retrieve file contents using relevant file paths
                content_response = await client.post(retrieve_file_contents_url, json=relevant_file_paths_payload)
                content_response.raise_for_status()

                file_content_response = FileContentResponse(**content_response.json())
                retrieved_file_paths = file_content_response.retrieved_file_paths

                combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
                for path, content in file_content_response.contents.items():
                    combined_prompt += f"\n--- {path} ---\n{content}\n"

            # Count tokens in the combined prompt
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

