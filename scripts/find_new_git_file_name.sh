#!/bin/bash
# Usage: ./find_new_git_file_name.sh <original-file-name>
#
#
# This script builds a mapping of rename events (from Git’s history)
# and then follows the chain starting with the provided original file name.
# It prints the final file name that the file was renamed to.
#
# Note: This script is not used in the project, but is merely for
# demonstration purposes for potential future needs.

if [ "$#" -ne 1 ]; then
  echo "Usage: $0 <original-file-name>"
  exit 1
fi

original_file="$1"

declare -A rename_map

# Retrieve all rename events using --name-status to get R entries
git_log=$(git log --diff-filter=R --name-status --reverse --pretty=format:"")

while IFS=$'\t' read -r status old new _; do
  if [[ "$status" =~ ^R ]]; then
    rename_map["$old"]="$new"
  fi
done <<< "$git_log"

current="$original_file"
chain=("$current")

while true; do
  if [ -n "${rename_map[$current]}" ]; then
    current="${rename_map[$current]}"
    chain+=("$current")
  else
    break
  fi
done

if [ "${#chain[@]}" -eq 1 ]; then
  echo "No renames found for '$original_file'."
else
  final="${chain[-1]}"
  echo "File '$original_file' was renamed to '$final'."
fi
