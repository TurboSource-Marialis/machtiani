import re
from fastapi import HTTPException
from app.utils import send_prompt_to_openai

async def generate_filename(context: str, api_key: str) -> str:
    filename_prompt = (
        f"Generate a unique filename for the following context: '{context}'.\n"
        "Respond ONLY with the filename in snake_case, wrapped in <filename> and </filename> tags.\n"
        "Do not include any other text or explanations.\n"
        "Example:\n"
        "<filename>example_filename</filename>"
    )

    response = await send_prompt_to_openai(filename_prompt, api_key, model="gpt-4o-mini")

    match = re.search(r"<filename>\s*(.*?)\s*</filename>", response, re.DOTALL | re.IGNORECASE)
    if not match:
        match = re.search(r"<\s*(.*?)\s*>", response)
    if match:
        filename = match.group(1).strip()
        return filename
    else:
        raise HTTPException(status_code=400, detail="Invalid response format from OpenAI API.")
