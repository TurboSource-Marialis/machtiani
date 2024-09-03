from langchain_core.prompts import PromptTemplate
from langchain_openai import ChatOpenAI
from typing import List
from contextlib import contextmanager
import sys
import os
import logging

logger = logging.getLogger(__name__)

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
logger.info("Adding to sys.path: %s", path_to_add)
sys.path.append(path_to_add)

logger.info("Current sys.path: %s", sys.path)

# Check if the path is correct
if os.path.exists(path_to_add):
    logger.info("Path exists: %s", path_to_add)
    logger.info("Contents of the path: %s", os.listdir(path_to_add))
else:
    logger.warning("Path does not exist: %s", path_to_add)

# Attempt to list directories where 'lib' should be
if os.path.exists(os.path.join(path_to_add, 'utils')):
    logger.info("Contents of 'utils' directory: %s", os.listdir(os.path.join(path_to_add, 'utils')))
else:
    logger.warning("No 'utils' directory found at %s", os.path.join(path_to_add, 'utils'))

# Import statements
try:
    with add_sys_path(path_to_add):
        from utils.enums import (
            FilePathEntry,
            FileSearchResponse
        )
    logger.info("Imports successful.")
except ModuleNotFoundError as e:
    logger.error(f"ModuleNotFoundError: {e}")
    logger.error("Failed to import the module. Please check the paths and directory structure.")

def aggregate_file_paths(responses: List[FileSearchResponse]) -> List[FilePathEntry]:
    file_paths = []
    for response in responses:
        file_paths.extend(response.file_paths)
    return file_paths

def remove_duplicate_file_paths(file_paths: List[FilePathEntry]) -> List[FilePathEntry]:
    unique_paths = {}
    for entry in file_paths:
        if entry.path not in unique_paths:
            unique_paths[entry.path] = entry
    return list(unique_paths.values())

def send_prompt_to_openai(prompt_text: str, api_key: str, model: str = "gpt-4o-mini") -> str:
    """
    Sends a prompt to OpenAI and returns the response.

    :param prompt_text: The text prompt to send to OpenAI.
    :param api_key: The API key for authentication with OpenAI.
    :param model: The model to use (default is "gpt-4o-mini").
    :return: The response from OpenAI as a string.
    """
    # Define the prompt template
    prompt = PromptTemplate(input_variables=["input_text"], template="{input_text}")

    # Initialize OpenAI LLM with the provided API key
    openai_llm = ChatOpenAI(api_key=api_key, model=model)

    # Chain the prompt and the LLM
    openai_chain = prompt | openai_llm

    # Execute the chain with the invoke method and return the response
    openai_response = openai_chain.invoke({"input_text": prompt_text})
    return openai_response.content

