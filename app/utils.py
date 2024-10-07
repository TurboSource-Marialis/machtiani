import sys
import os
import logging
from langchain_core.prompts import PromptTemplate
from langchain_openai import ChatOpenAI
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

async def send_prompt_to_openai(prompt_text: str, api_key: str, model: str = "gpt-4o-mini", timeout: int = 3600, max_retries: int = 5) -> str:
    prompt = PromptTemplate(input_variables=["input_text"], template="{input_text}")
    openai_llm = ChatOpenAI(openai_api_key=api_key, model=model, request_timeout=timeout, max_retries=max_retries)
    openai_chain = prompt | openai_llm
    openai_response = openai_chain.invoke({"input_text": prompt_text})
    return openai_response.content

async def count_tokens(text: str) -> int:
    return len(text) // 4 + 1
