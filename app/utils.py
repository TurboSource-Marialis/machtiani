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

