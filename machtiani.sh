#!/bin/bash

# Initialize variables
markdown_flag=false
markdown_file=""

# Parse the --markdown flag and file path
while [[ "$1" == --* ]]; do
    case "$1" in
        --markdown)
            markdown_flag=true
            markdown_file="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

# Check if required arguments are provided
if [ "$#" -lt 2 ]; then
    echo "Usage: $0 [--markdown <markdown_file>] <project> <match_strength>"
    exit 1
fi

# Assign arguments to variables
if [ "$markdown_flag" = true ]; then
    prompt=$(cat "$markdown_file")
else
    prompt="$1"
    shift
fi

project="$1"
match_strength="$2"

# Check if OPENAI_API_KEY is set
if [ -z "$OPENAI_API_KEY" ]; then
    echo "Error: OPENAI_API_KEY environment variable is not set."
    exit 1
fi

# URL encode the prompt
encoded_prompt=$(printf '%s' "$prompt" | jq -sRr @uri)

# Make the API call
response=$(curl -s -X 'POST' \
  "http://localhost:5071/generate-response?prompt=$encoded_prompt&project=$project&mode=commit&model=gpt-4o-mini&api_key=$OPENAI_API_KEY&match_strength=$match_strength" \
  -H 'accept: application/json' \
  -d '')

# Extract and process the OpenAI response
openai_response=$(echo "$response" | jq -r '.openai_response' | sed 's/\\n/\n/g' | sed 's/\\"/"/g')

# Create a temporary directory
temp_dir=$(mktemp -d)

# Save the Markdown content to a file in the temporary directory
temp_file="$temp_dir/response.md"
if [ "$markdown_flag" = true ]; then
    echo -e "$prompt\n\n# Assitant\n\n$openai_response" > "$temp_file"
else
    echo -e "# User\n\n$prompt\n\n# Assitant\n\n$openai_response" > "$temp_file"
fi

# Open the Markdown file
glow "$temp_file"

# Print out the path to the file
echo "Response saved to $temp_file and opened in your default Markdown viewer."

