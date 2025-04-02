import os
import logging
from fastapi import FastAPI
from .routes.generate_filename import router as generate_filename
from .routes.generate_response import router as generate_response
from .routes.get_install_info import router as get_install_info

logging.getLogger().setLevel(logging.CRITICAL)
logger = logging.getLogger(__name__)

app = FastAPI()

logger.critical("Application is starting up...")

# Include the routers
app.include_router(generate_filename)
app.include_router(generate_response)
app.include_router(get_install_info)
