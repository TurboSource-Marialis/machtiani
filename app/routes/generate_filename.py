from pydantic import HttpUrl, BaseModel
from typing import Optional
from fastapi import APIRouter, Query, HTTPException
from app.services.generate_filename_service import generate_filename

# Define a response model
class FilenameResponse(BaseModel):
    filename: str

router = APIRouter()

@router.get("/generate-filename", response_model=FilenameResponse)
async def generate_filename_route(
    context: str = Query(..., description="Context to create filename"),
    llm_model: str = Query(..., description="Optional LLM model"),
    llm_model_api_key: str = Query(..., description="API key for OpenAI model"),
    llm_model_base_url: HttpUrl = Query(..., description="LLM base url"),
    llm_model_base_url_other: Optional[str] = Query(None, description="Optional other LLM base url"),
    llm_model_api_key_other: Optional[str] = Query(None, description="Optional other LLM api key"),
) -> FilenameResponse:
    filename = await generate_filename(
        context,
        llm_model,
        llm_model_api_key,
        llm_model_base_url,
        llm_model_base_url_other,
        llm_model_api_key_other
    )

    return FilenameResponse(filename=filename)
