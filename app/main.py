import httpx
from fastapi import FastAPI, Query, HTTPException
from typing import Optional, List
from contextlib import contextmanager
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

@app.post("/generate-answer/")
async def generate_answer(
    prompt: str = Query(..., description="The prompt to search for"),
    project: str = Query(..., description="The project to search"),
    mode: SearchMode = Query(..., description="Search mode: content, commit, or super"),
    model: EmbeddingModel = Query(..., description="The embedding model used"),
    api_key: str = Query(..., description="The openai api key."),
    match_strength: MatchStrength = Query(MatchStrength.HIGH, description="The strength of the match")
):
    # Define the URL of the machtiani-commit-file-retrieval service
    infer_file_url = "http://commit-file-retrieval:5070/infer-file/"

    # Prepare the request payload
    params = {
        "prompt": prompt,
        "project": project,
        "mode": mode.value,  # Convert enum to string
        "model": model.value,  # Convert enum to string
        "match_strength": match_strength.value,  # Convert enum to string
        "api_key": api_key,  # Include the api_key in the request
    }

    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(infer_file_url, params=params)
            response.raise_for_status()  # Raise an exception for HTTP errors
            # Log detailed response information
            logger.info(f"Response status code: {response.status_code}")
            logger.info(f"Response headers: {response.headers}")
            logger.info(f"Response content: {response.text}")

        # Return the response from the machtiani-commit-file-retrieval service
        return response.json()
    except httpx.RequestError as exc:
        raise HTTPException(status_code=500, detail=f"Error connecting to machtiani-commit-file-retrieval: {exc}")
    except httpx.HTTPStatusError as exc:
        raise HTTPException(status_code=exc.response.status_code, detail=f"Error response from machtiani-commit-file-retrieval: {exc.response.json()}")

