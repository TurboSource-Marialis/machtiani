import httpx
from fastapi import FastAPI, Query, HTTPException
from typing import Optional, List
from contextlib import contextmanager
from .utils import aggregate_file_paths
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

@app.post("/generate-response", response_model=List[FilePathEntry])
async def generate_response(
    prompt: str = Query(..., description="The prompt to search for"),
    project: str = Query(..., description="The project to search"),
    mode: SearchMode = Query(..., description="Search mode: content, commit, or super"),
    model: EmbeddingModel = Query(..., description="The embedding model used"),
    api_key: str = Query(..., description="The OpenAI API key."),
    match_strength: MatchStrength = Query(MatchStrength.HIGH, description="The strength of the match")
) -> List[FilePathEntry]:
    infer_file_url = "http://commit-file-retrieval:5070/infer-file/"

    params = {
        "prompt": prompt,
        "project": project,
        "mode": mode.value,
        "model": model.value,
        "match_strength": match_strength.value,
        "api_key": api_key,
    }

    try:
        async with httpx.AsyncClient() as client:
            response = await client.get(infer_file_url, params=params)
            response.raise_for_status()
            logger.info(f"Response status code: {response.status_code}")
            logger.info(f"Response headers: {response.headers}")
            logger.info(f"Response content: {response.text}")

            # Parse the JSON response into a list of FileSearchResponse objects
            list_file_search_response = [FileSearchResponse(**item) for item in response.json()]

        # Aggregate file paths
        list_file_path_entry = aggregate_file_paths(list_file_search_response)

        return list_file_path_entry

    except httpx.RequestError as exc:
        raise HTTPException(status_code=500, detail=f"Error connecting to machtiani-commit-file-retrieval: {exc}")
    except httpx.HTTPStatusError as exc:
        raise HTTPException(status_code=exc.response.status_code, detail=f"Error response from machtiani-commit-file-retrieval: {exc.response.json()}")

