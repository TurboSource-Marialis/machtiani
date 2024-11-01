import httpx
import json
import re
from fastapi import APIRouter, Body, HTTPException
from pydantic import SecretStr
from app.utils import (
    aggregate_file_paths,
    remove_duplicate_file_paths,
    send_prompt_to_openai,
    FileContentResponse,
    count_tokens,
)
from app.utils.enums import SearchMode, FileSearchResponse

router = APIRouter()

# Define token limits for different models
TOKEN_LIMITS = {
    "gpt-4o": 128000,
    "gpt-4o-mini": 128000,
}

@router.post("/generate-response")
async def generate_response(
    prompt: str = Body(..., description="The prompt to search for"),
    project: str = Body(..., description="The project to search"),
    mode: str = Body(..., description="Search mode: chat, commit, or super"),
    model: str = Body(..., description="The embedding model used"),
    match_strength: str = Body(..., description="The strength of the match"),
    api_key: str = Body(..., description="API key for OpenAI model"),
    codehost_api_key: SecretStr = Body(..., description="Code host API key for authentication"),
    codehost_url: str = Body(..., description="Code host URL for the repository"),
    ignore_files: List[str] = Body(..., description="List of file paths to ignore"),
):
    if model not in TOKEN_LIMITS:
        raise HTTPException(status_code=400, detail="Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")

    if match_strength not in ["high", "mid", "low"]:
        raise HTTPException(status_code=400, detail="Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")

    base_url = "http://commit-file-retrieval:5070"

    infer_file_url = f"{base_url}/infer-file/"
    get_file_summary_url = f"{base_url}/get-file-summary/?project_name={project}"
    test_pull_access_url = f"{base_url}/test-pull-access/"

    try:
        async with httpx.AsyncClient(timeout=httpx.Timeout(1200, read=1200.0)) as client:
            params = {
                'project_name': project,
                'codehost_api_key': codehost_api_key.get_secret_value(),
                'codehost_url': codehost_url
            }
            pull_access_response = await client.post(test_pull_access_url, params=params)
            pull_access_response.raise_for_status()
            pull_access_data = pull_access_response.json()
            if not pull_access_data.get('pull_access', False):
                raise HTTPException(status_code=403, detail="Pull access denied.")

            # Logic for handling different modes and retrieving file contents follows...
            # (The rest of the original generate_response functionality goes here)

    except httpx.RequestError as exc:
        raise HTTPException(status_code=500, detail=f"Error connecting to commit-file-retrieval service: {exc}")
    except httpx.HTTPStatusError as exc:
        raise HTTPException(status_code=exc.response.status_code, detail=f"Error response from commit-file-retrieval service: {exc.response.json()}")
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"An unexpected error occurred: {str(e)}")
