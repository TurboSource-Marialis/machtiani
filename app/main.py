import httpx
from fastapi import FastAPI, Query, HTTPException
from typing import Optional, List, Dict
from contextlib import contextmanager
from .utils import aggregate_file_paths, remove_duplicate_file_paths, send_prompt_to_openai
import sys
import os
import logging

app = FastAPI()

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

@app.post("/generate-response", response_model=Dict[str, str])
async def generate_response(
    prompt: str = Query(..., description="The prompt to search for"),
    project: str = Query(..., description="The project to search"),
    mode: SearchMode = Query(..., description="Search mode: content, commit, or super"),
    model: str = Query("gpt-4o-mini", description="The embedding model used"),
    api_key: str = Query(..., description="The OpenAI API key."),
    match_strength: str = Query("mid", description="The strength of the match")
) -> Dict[str, str]:
    # Validate the model
    if model not in ["gpt-4o", "gpt-4o-mini"]:
        raise HTTPException(status_code=400, detail="Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")

    # Validate the match strength
    if match_strength not in ["high", "mid", "low"]:
        raise HTTPException(status_code=400, detail="Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")

    infer_file_url = "http://commit-file-retrieval:5070/infer-file/"
    retrieve_file_contents_url = f"http://commit-file-retrieval:5070/retrieve-file-contents/?project_name={project}"

    params = {
        "prompt": prompt,
        "project": project,
        "mode": mode.value,
        "model": model,
        "match_strength": match_strength,
        "api_key": api_key,
    }

    try:
        async with httpx.AsyncClient() as client:
            # First, call the infer-file endpoint
            response = await client.get(infer_file_url, params=params)
            response.raise_for_status()
            logger.info(f"Response status code: {response.status_code}")
            logger.info(f"Response headers: {response.headers}")
            logger.info(f"Response content: {response.text}")

            # Parse the JSON response into a list of FileSearchResponse objects
            list_file_search_response = [FileSearchResponse(**item) for item in response.json()]

            # Aggregate and deduplicate file paths
            list_file_path_entry = aggregate_file_paths(list_file_search_response)
            list_file_path_entry = remove_duplicate_file_paths(list_file_path_entry)

            # Prepare the list of file paths for the retrieve-file-contents endpoint
            file_paths_payload = [entry.dict() for entry in list_file_path_entry]

            # Log the payload for debugging
            logger.info(f"Payload for retrieve-file-contents: {file_paths_payload}")
            # Check if file_paths_payload is empty and return an appropriate response if so
            if not file_paths_payload:
                return {"machtiani": "no files found"}

            # Call the retrieve-file-contents endpoint with the deduplicated paths
            content_response = await client.post(retrieve_file_contents_url, json=file_paths_payload)
            content_response.raise_for_status()

            # Get the file contents
            file_contents = content_response.json()

            # Append the file contents to the prompt
            combined_prompt = f"{prompt}\n\nHere are the relevant files:\n"
            for path, content in file_contents.items():
                combined_prompt += f"\n--- {path} ---\n{content}\n"

            # Use the utility function to send the combined prompt to OpenAI
            openai_response = send_prompt_to_openai(combined_prompt, api_key)

            # Return the OpenAI response
            return {"openai_response": openai_response}

    except httpx.RequestError as exc:
        raise HTTPException(status_code=500, detail=f"Error connecting to machtiani-commit-file-retrieval: {exc}")
    except httpx.HTTPStatusError as exc:
        raise HTTPException(status_code=exc.response.status_code, detail=f"Error response from machtiani-commit-file-retrieval: {exc.response.json()}")

