from fastapi import APIRouter, Body
from fastapi.responses import StreamingResponse
import json
from pydantic import SecretStr, HttpUrl
from typing import List, Optional
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
    codehost_api_key: Optional[SecretStr] = Body(..., description="Code host API key for authentication"),
    codehost_url: HttpUrl = Body(..., description="Code host URL for the repository"),
    ignore_files: List[str] = Body(..., description="List of file paths to ignore"),
):
    async def event_stream():
        async for response in generate_response(
            prompt,
            project,
            mode,
            model,
            match_strength,
            api_key,
            codehost_api_key,
            codehost_url,
            ignore_files,
        ):
            yield json.dumps(response) + '\n'

    return StreamingResponse(event_stream(), media_type="application/json")
