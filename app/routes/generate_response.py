from fastapi import APIRouter, Body
from pydantic import SecretStr
from typing import List
from app.services.generate_response_service import generate_response

router = APIRouter()

@router.post("/generate-response")
async def generate_response_route(
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
    return await generate_response(
        prompt,
        project,
        mode,
        model,
        match_strength,
        api_key,
        codehost_api_key,
        codehost_url,
        ignore_files,
    )
