import sys
import os
import asyncio
import json
import logging
from langchain_core.prompts import PromptTemplate
from langchain_openai import ChatOpenAI
from langchain.callbacks import AsyncIteratorCallbackHandler
from langchain.schema import HumanMessage
from typing import List

logger = logging.getLogger(__name__)

class add_sys_path:
    def __init__(self, path):
        self.path = path
        self.original_sys_path = sys.path.copy()

    def __enter__(self):
        sys.path.append(self.path)

    def __exit__(self, exc_type, exc_value, traceback):
        sys.path = self.original_sys_path

# Update the path to correctly point to machtiani-commit-file-retrieval/lib
path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval/lib')
logger.info("Adding to sys.path: %s", path_to_add)

# Use the context manager to handle imports
try:
    with add_sys_path(path_to_add):
        from utils.enums import (
            FilePathEntry,
            FileSearchResponse,
            FileContentResponse
        )
    logger.info("Imports successful.")
except ModuleNotFoundError as e:
    logger.error(f"ModuleNotFoundError: {e}")
    logger.error("Failed to import the module. Please check the paths and directory structure.")

async def aggregate_file_paths(responses: List[FileSearchResponse]) -> List[FilePathEntry]:
    file_paths = []
    for response in responses:
        file_paths.extend(response.file_paths)
    return file_paths

async def remove_duplicate_file_paths(file_paths: List[FilePathEntry]) -> List[FilePathEntry]:
    unique_paths = {}
    for entry in file_paths:
        if entry.path not in unique_paths:
            unique_paths[entry.path] = entry
    return list(unique_paths.values())

async def send_prompt_to_openai_streaming(
    prompt_text: str,
    api_key: str,
    model: str = "gpt-4o-mini",
    timeout: int = 3600,
    max_retries: int = 5,
):
    # Define the prompt template
    prompt = PromptTemplate(input_variables=["input_text"], template="{input_text}")

    # Initialize the callback handler for streaming
    callback = AsyncIteratorCallbackHandler()

    # Initialize the ChatOpenAI model with streaming enabled
    openai_llm = ChatOpenAI(
        openai_api_key=api_key,
        model=model,
        request_timeout=timeout,
        max_retries=max_retries,
        streaming=True,
        callbacks=[callback],
    )

    # Format the input text using the prompt template
    input_text = prompt.format(input_text=prompt_text)
    messages = [HumanMessage(content=input_text)]

    # Start the asynchronous generation process
    generation_task = asyncio.create_task(openai_llm.agenerate(messages=[messages]))

    # Iterate over the streaming tokens
    async for token in callback.aiter():
        # Yield each token as a JSON-formatted string
        yield json.dumps({"token": token})

    # Await the completion of the generation task
    await generation_task

async def count_tokens(text: str) -> int:
    return len(text) // 4 + 1


async def check_token_limit(prompt: str, model: str, token_limits: dict) -> bool:
    token_count = await count_tokens(prompt)
    max_tokens = token_limits.get(model)

    logger.info(f"model: {model}, token count: {token_count}, max limit: {max_tokens}")

    if token_count > max_tokens:
        return False  # Return False if the token count exceeds the limit

    return True  # Return True if within limit
