import re
import json
from fastapi import HTTPException
from app.utils import send_prompt_to_openai_streaming

async def generate_filename(context: str, llm_model_api_key: str) -> str:
    filename_prompt = (
        f"Generate a unique filename for the following context: '{context}'.\n"
        "Respond ONLY with the filename in snake_case, wrapped in <filename> and </filename> tags.\n"
        "Do not include any other text or explanations.\n"
        "Example:\n"
        "<filename>example_filename</filename>"
    )

    response_tokens = []

    try:
        # Asynchronously iterate over each token yielded by send_prompt_to_openai_streaming
        async for token_json in send_prompt_to_openai_streaming(filename_prompt, llm_model_api_key, model="gpt-4o-mini"):
            # Parse the JSON string to extract the token
            token_data = json.loads(token_json)
            token = token_data.get("token", "")
            response_tokens.append(token)

        # Concatenate all tokens to form the complete response string
        response = ''.join(response_tokens)

    except Exception as e:
        # Handle potential errors during token retrieval
        raise HTTPException(status_code=500, detail=f"Error processing OpenAI response: {str(e)}")

    # Use regular expressions to extract the filename from the response
    match = re.search(r"<filename>\s*(.*?)\s*</filename>", response, re.DOTALL | re.IGNORECASE)
    if not match:
        match = re.search(r"<\s*(.*?)\s*>", response)
    if match:
        filename = match.group(1).strip()
        return filename
    else:
        # If no valid filename is found, raise an HTTP exception
        raise HTTPException(status_code=400, detail="Invalid response format from OpenAI API.")

