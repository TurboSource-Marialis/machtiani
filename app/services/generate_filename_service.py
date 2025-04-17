import re
import os
import json
import logging
from pydantic import HttpUrl
from typing import Optional
from fastapi import HTTPException
from app.utils import add_sys_path

# Update the path to correctly point to machtiani-commit-file-retrieval/lib
path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval')
logger = logging.getLogger(__name__)
logger.info("Adding to sys.path: %s", path_to_add)

# Use the context manager to handle imports
try:
    with add_sys_path(path_to_add):
        from lib.ai.llm_model import (
            LlmModel,
        )
    logger.info("Imports successful.")
except ModuleNotFoundError as e:
    logger.error(f"ModuleNotFoundError: {e}")
    logger.error("Failed to import the module. Please check the paths and directory structure.")



async def generate_filename(context: str, llm_model_api_key: str, llm_model_base_url: HttpUrl, llm_model_base_url_other: Optional[str] = None, llm_model_api_key_other: Optional[str] = None) -> str:
    logger.info("Generating filename for context (length: %d chars)", len(context))
    logger.debug("Using LLM base URL: %s", llm_model_base_url)
    if llm_model_base_url_other:
        logger.debug("Using alternate LLM base URL: %s", llm_model_base_url_other)

    filename_prompt = (
        f"Generate a unique filename for the following context: '{context}'.\n"
        "Respond ONLY with the filename in snake_case, wrapped in <filename> and </filename> tags.\n"
        "Do not include any other text or explanations.\n"
        "Example:\n"
        "<filename>example_filename</filename>"
    )

    response_tokens = []

    try:

        # Instantiate LlmModel
        logger.debug("Initializing LLM model with API key (length: %d)", len(llm_model_api_key or ""))
        llm_model = LlmModel(api_key=llm_model_api_key, base_url=str(llm_model_base_url))

        logger.debug("Sending prompt to LLM model")
        async for token_json in llm_model.send_prompt_streaming(filename_prompt):
            # Parse the JSON string to extract the token
            token_data = json.loads(token_json)
            token = token_data.get("token", "")
            response_tokens.append(token)
            logger.debug("Received token: %s", token)


        # Concatenate all tokens to form the complete response string
        response = ''.join(response_tokens)
        logger.debug("Full LLM response: %s", response)

    except Exception as e:
        # Handle potential errors during token retrieval
        raise HTTPException(status_code=500, detail=f"Error processing OpenAI response: {str(e)}")

    # Use regular expressions to extract the filename from the response
    match = re.search(r"<filename>\s*(.*?)\s*</filename>", response, re.DOTALL | re.IGNORECASE)
    if not match:
        match = re.search(r"<\s*(.*?)\s*>", response)

    if match:
        filename = match.group(1).strip()
        logger.debug("Extracted filename: %s", filename)

        # Process the filename to remove all extensions
        base = filename
        while True:
            base, ext = os.path.splitext(base)
            if not ext:
                break
        processed_filename = base
        logger.info("Generated final filename: %s", processed_filename)
        return processed_filename
    else:
        logger.error("Could not extract filename from response: %s", response)
        raise HTTPException(status_code=400, detail="Invalid response format from OpenAI API.")
