import httpx
from fastapi import FastAPI, Body, Query, HTTPException
from typing import Optional, List, Dict, Union
from contextlib import contextmanager
from .utils import aggregate_file_paths, remove_duplicate_file_paths, send_prompt_to_openai, FileContentResponse, count_tokens
import sys
import os
import re
import logging

app = FastAPI()

# Define token limits for different models
TOKEN_LIMITS = {
    "gpt-4o": 128000,
    "gpt-4o-mini": 128000
}

# Use the logger instead of print
logger = logging.getLogger("uvicorn")
logger.info("Application is starting up...")

@contextmanager
def add_sys_path(path):
    original_sys_path = sys.path.copy()
    sys.path.append(path)
    try:
        yield
    finally:
        sys.path = original_sys_path

# Update the path to correctly point to machtiani-commit-file-retrieval/lib
path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval/lib')
print("Adding to sys.path:", path_to_add)
sys.path.append(path_to_add)

print("Current sys.path:", sys.path)

# Check if the path is correct
if os.path.exists(path_to_add):
    print("Path exists:", path_to_add)
    print("Contents of the path:", os.listdir(path_to_add))
else:
    print("Path does not exist:", path_to_add)

# Attempt to list directories where 'lib' should be
if os.path.exists(os.path.join(path_to_add, 'utils')):
    print("Contents of 'utils' directory:", os.listdir(os.path.join(path_to_add, 'utils')))
else:
    print("No 'utils' directory found at", os.path.join(path_to_add, 'utils'))

# Import statements
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
            FetchAndCheckoutBranchRequest
        )
    print("Imports successful.")
except ModuleNotFoundError as e:
    print(f"ModuleNotFoundError: {e}")
    print("Failed to import the module. Please check the paths and directory structure.")

@app.get("/generate-filename", response_model=str)
async def generate_filename(
    api_key: str = Query(..., description="The OpenAI API key."),
    context: str = Query(..., description="Context to create filename"),
) -> str:
    filename_prompt = (
        f"Generate a unique filename for the following context: '{context}'.\n"
        "Respond ONLY with the filename in snake_case, wrapped in <filename> and </filename> tags.\n"
        "Do not include any other text or explanations.\n"
        "Example:\n"
        "<filename>example_filename</filename>"
    )
    response = send_prompt_to_openai(filename_prompt, api_key, model="gpt-4o-mini")
    logger.info(f"OpenAI response: {response}")

    # Extract the filename using regex
    match = re.search(r"<filename>\s*(.*?)\s*</filename>", response, re.DOTALL | re.IGNORECASE)
    if not match:
        # Try to match any text within angle brackets
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
    api_key: str = Body(..., description="The OpenAI API key."),
    match_strength: str = Body(..., description="The strength of the match"),
    embeddings: Optional[List[float]] = Body(None, description="Embeddings for the prompt")
):
    # Validate the model
    if model not in TOKEN_LIMITS:
        return {"error": "Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'."}

    # Validate the match strength
    if match_strength not in ["high", "mid", "low"]:
        return {"error": "Invalid match strength selected. Choose either 'high', 'mid', or 'low'."}

    infer_file_url = "http://commit-file-retrieval:5070/infer-file/"
    retrieve_file_contents_url = f"http://commit-file-retrieval:5070/retrieve-file-contents/?project_name={project}"

    params = {
        "prompt": prompt,
        "project": project,
        "mode": mode,
        "model": model,
        "match_strength": match_strength,
        "api_key": api_key,
        "embeddings": embeddings  # Pass the embeddings here
    }

    # Initialize ignore_files list
    ignore_files = []
    try:
        with open('.machtiani.ignore', 'r') as f:
            ignore_files = [line.strip() for line in f if line.strip()]
    except FileNotFoundError:
        logger.warning("No .machtiani.ignore file found, proceeding without ignoring any files.")

    try:
        async with httpx.AsyncClient() as client:
            response = await client.post(infer_file_url, json=params)
            response.raise_for_status()
            logger.info(f"Response status code: {response.status_code}")

            list_file_search_response = [FileSearchResponse(**item) for item in response.json()]

            if mode == SearchMode.pure_chat:
                combined_prompt = prompt
                retrieved_file_paths = []
            else:
                list_file_path_entry = aggregate_file_paths(list_file_search_response)
                list_file_path_entry = remove_duplicate_file_paths(list_file_path_entry)
                list_file_path_entry = [entry for entry in list_file_path_entry if entry.path not in ignore_files]

                file_paths_payload = [entry.dict() for entry in list_file_path_entry]
                logger.info(f"Payload for retrieve-file-contents: {file_paths_payload}")

                if not file_paths_payload:
                    return {"machtiani": "no files found"}

                content_response = await client.post(retrieve_file_contents_url, json=file_paths_payload)
                content_response.raise_for_status()

                file_content_response = FileContentResponse(**content_response.json())
                retrieved_file_paths = file_content_response.retrieved_file_paths

                combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
                for path, content in file_content_response.contents.items():
                    combined_prompt += f"\n--- {path} ---\n{content}\n"

            # Count tokens in the combined prompt
            token_count = count_tokens(combined_prompt)
            max_tokens = TOKEN_LIMITS[model]
            logger.info(f"model: {model}, token count: {token_count}, max limit: {max_tokens}")

            # Validate token count against model limits
            if token_count > max_tokens:
                error_message = (
                    f"Token limit exceeded for the selected model. "
                    f"Limit: {max_tokens}, Count: {token_count}. "
                    f"Please reduce the length of your prompt or the number of retrieved contents."
                )
                logger.error(error_message)
                return {"error": error_message}

            # Call the OpenAI API
            openai_response = send_prompt_to_openai(combined_prompt, api_key, model)

            return {"openai_response": openai_response, "retrieved_file_paths": retrieved_file_paths}

    except httpx.RequestError as exc:
        logger.error(f"Request error: {exc}")
        return {"error": f"Error connecting to machtiani-commit-file-retrieval: {exc}"}
    except httpx.HTTPStatusError as exc:
        logger.error(f"HTTP status error: {exc.response.json()}")
        return {"error": f"Error response from machtiani-commit-file-retrieval: {exc.response.json()}"}
    except Exception as e:
        logger.exception("Unexpected error occurred")
        return {"error": f"An unexpected error occurred: {str(e)}"}
