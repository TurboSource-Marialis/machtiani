import sys
import os
import asyncio
import json
import logging
from typing import List, Tuple

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
path_to_add = os.path.abspath('/app/machtiani-commit-file-retrieval')
logger.info("Adding to sys.path: %s", path_to_add)

# Use the context manager to handle imports
try:
    with add_sys_path(path_to_add):
        from lib.utils.enums import (
            FilePathEntry,
            SearchMode,
        )

        from app.models.responses import (
            FileSearchResponse,
            FileContentResponse
        )
        from lib.ai.llm_model import (
            LlmModel,
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

def separate_file_paths_by_type(file_search_responses: List[FileSearchResponse]) -> Tuple[List[FilePathEntry], List[FilePathEntry]]:
    commit_file_paths = []
    file_file_paths = []

    for response in file_search_responses:
        for file_path_entry in response.file_paths:
            if response.path_type == 'commit':
                commit_file_paths.append(FilePathEntry(path=file_path_entry.path))
            elif response.path_type == 'file':
                file_file_paths.append(FilePathEntry(path=file_path_entry.path))

    return commit_file_paths, file_file_paths

async def count_tokens(text: str) -> int:
    return len(text) // 4 + 1


async def check_token_limit(prompt: str, model: str, max_tokens: int) -> bool:
    token_count = await count_tokens(prompt)

    logger.info(f"model: {model}, token count: {token_count}, max limit: {max_tokens}")

    if token_count > max_tokens:
        return False  # Return False if the token count exceeds the limit

    return True  # Return True if within limit
