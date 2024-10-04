#!/usr/bin/env python3

import os
import subprocess
import argparse
import yaml
from urllib.parse import urlparse

def load_config():
    """Load configuration from machtiani-config.yml."""
    config_path = "data/.machtiani-config.yml"
    try:
        with open(config_path, 'r') as file:
            config = yaml.safe_load(file)
        return config
    except FileNotFoundError:
        print(f"Configuration file '{config_path}' not found.")
        return None
    except yaml.YAMLError as e:
        print(f"Error parsing YAML configuration: {e}")
        return None

def get_remote_url(remote_name='origin', git_dir=None):
    """Fetch the remote URL for the given remote name."""
    try:
        if git_dir:
            command = ['git', '-C', git_dir, 'remote', 'get-url', remote_name]
        else:
            command = ['git', 'remote', 'get-url', remote_name]

        print(f"Running command: {' '.join(command)}")  # Debugging output
        remote_url = subprocess.check_output(command, text=True).strip()
        return remote_url
    except subprocess.CalledProcessError as e:
        print(f"Error getting remote URL for {remote_name}: {e}")
        return None

def parse_repo_url(repo_url):
    """Return the repository URL as is for authentication."""
    return repo_url  # Simply return the repo_url without modification

def force_push(repo_url, branch_name, username, token, git_dir):
    """Force push changes to the specified branch of the remote repository."""
    # Construct the repo_url_with_auth correctly
    if repo_url.startswith("https://"):
        repo_url_with_auth = f"{repo_url.replace('https://', '')}"
    else:
        repo_url_with_auth = repo_url

    repo_url_with_auth = f"https://{username}:{token}@{repo_url_with_auth}"

    try:
        # Change to the specified Git directory before pushing
        command = ['git', '-C', git_dir, 'push', repo_url_with_auth, branch_name, '--force']
        print(f"Running command: {' '.join(command)}")  # Debugging output
        subprocess.run(command, check=True)
        print(f"Successfully force pushed to {repo_url} on branch {branch_name}.")
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while pushing to the remote repository: {e}")

def main():
    parser = argparse.ArgumentParser(description='Force push changes to a remote git repository.')
    parser.add_argument('branch_name', type=str, help='The name of the branch to push changes to.')
    parser.add_argument('--remote', type=str, default='origin', help='The name of the remote repository (default: origin).')
    parser.add_argument('--git-dir', type=str, required=True, help='Path to the Git repository directory.')
    parser.add_argument('--username', type=str, required=True, help='Your Git username.')

    args = parser.parse_args()

    # Load the configuration to get the access token
    config = load_config()
    if config is None or 'environment' not in config:
        print("Failed to load configuration or 'environment' section is missing.")
        return

    code_host_api_key = config['environment'].get('CODE_HOST_API_KEY')
    if not code_host_api_key:
        print("CODE_HOST_API_KEY not found in the configuration.")
        return

    # Get the remote URL from the Git configuration
    remote_url = get_remote_url(args.remote, args.git_dir)
    if remote_url is None:
        print(f"Could not find the remote URL for '{args.remote}' in the specified directory.")
        return

    # Parse the repository URL
    parsed_repo_url = parse_repo_url(remote_url)

    # Execute the force push operation
    force_push(parsed_repo_url, args.branch_name, args.username, code_host_api_key, args.git_dir)

if __name__ == "__main__":
    main()

#### Usage Instructions
#
#You can now run the script with the username passed as an argument:
#
#```bash
#poetry run python scripts/force_push.py main --remote origin --git-dir data/git-projects/SWE-agent --username your_username
#```
#
#### Important Notes
#
#- **Username**: Ensure to replace `your_username` with your actual Git username when running the script.
#- **Configuration**: The `machtiani-config.yml` file must still contain the `CODE_HOST_API_KEY` for authentication with the remote repository.
#- **Debugging**: The command being executed to fetch the remote URL is printed to the console for easier debugging if issues arise.
#
#With these changes, the script should now work correctly while allowing you to specify the username dynamically. If you encounter any further issues, please let me know!
