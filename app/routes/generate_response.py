from fastapi import APIRouter, Body
from fastapi.responses import StreamingResponse
import json
from pydantic import SecretStr, HttpUrl
from typing import List, Optional
from app.services.generate_response_service import generate_response

router = APIRouter()

# Set up logging
import logging
logger = logging.getLogger(__name__)

@router.post("/generate-response")
async def generate_response_route(
    prompt: str = Body(..., description="The prompt to search for"),
    project: str = Body(..., description="The project to search"),

    mode: str = Body(..., description="Search mode: chat, pure-chat, answer-only, or default"),
    model: str = Body(..., description="The embedding model used"),
    match_strength: str = Body(..., description="The strength of the match"),
    llm_model_api_key: str = Body(..., description="API key for OpenAI model"),
    llm_model_base_url: HttpUrl = Body(..., description="LLM base url"),
    codehost_api_key: Optional[SecretStr] = Body(..., description="Code host API key for authentication"),
    codehost_url: HttpUrl = Body(..., description="Code host URL for the repository"),
    ignore_files: List[str] = Body(..., description="List of file paths to ignore"),
    head_commit_hash: str = Body(..., description="The head of the git repository"),
    llm_model_base_url_other: Optional[str] = Body(None, description="Optional other LLM base url"),
    llm_model_api_key_other: Optional[str] = Body(None, description="Optional other LLM api key"),
):

    # Log the received payload values and their types for debugging.
    logger.debug("Received /generate-response call with:")
    logger.debug(f"  prompt: {prompt} (type: {type(prompt)})")
    logger.debug(f"  project: {project} (type: {type(project)})")
    logger.debug(f"  mode: {mode} (type: {type(mode)})")
    logger.debug(f"  model: {model} (type: {type(model)})")
    logger.debug(f"  match_strength: {match_strength} (type: {type(match_strength)})")
    logger.debug(f"  llm_model_api_key: {llm_model_api_key} (type: {type(llm_model_api_key)})")
    logger.debug(f"  llm_model_base_url: {llm_model_base_url} (type: {type(llm_model_base_url)})")
    logger.debug(f"  codehost_api_key: {codehost_api_key} (type: {type(codehost_api_key)})")
    logger.debug(f"  codehost_url: {codehost_url} (type: {type(codehost_url)})")
    logger.debug(f"  ignore_files: {ignore_files} (type: {type(ignore_files)})")
    logger.debug(f"  llm_model_base_url_other: {llm_model_base_url_other} (type: {type(llm_model_base_url_other)})")
    logger.debug(f"  llm_model_api_key_other: {llm_model_api_key_other} (type: {type(llm_model_api_key_other)})")
    logger.debug(f"  head_commit_hash: {head_commit_hash} (type: {type(head_commit_hash)})")
    async def event_stream():
        async for response in generate_response(
            prompt,
            project,
            mode,
            model,
            match_strength,
            llm_model_api_key,
            llm_model_base_url,
            codehost_api_key,
            codehost_url,
            ignore_files,
            head_commit_hash,
            llm_model_base_url_other,
            llm_model_api_key_other,
        ):
            logger.debug(f"Streaming response chunk: {response}")
            yield json.dumps(response) + '\n'

    return StreamingResponse(event_stream(), media_type="application/json")
