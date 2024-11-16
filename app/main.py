import os
import logging
from fastapi import FastAPI
from .routes.generate_filename import router as generate_filename
from .routes.generate_response import router as generate_response
from .routes.get_head_oid import router as get_head_oid

app = FastAPI()

# Use the logger instead of print
logger = logging.getLogger("uvicorn")
logger.info("Application is starting up...")

# Include the routers
app.include_router(generate_filename)
app.include_router(generate_response)
app.include_router(get_head_oid)
