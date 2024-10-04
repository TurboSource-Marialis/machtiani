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

def force_push(repo_url, from_branch, to_branch, username, token, git_dir):
    """Force push changes from the specified branch to a target branch on the remote repository."""
    # Construct the repo_url_with_auth correctly
    if repo_url.startswith("https://"):
        repo_url_with_auth = f"{repo_url.replace('https://', '')}"
    else:
        repo_url_with_auth = repo_url

    repo_url_with_auth = f"https://{username}:{token}@{repo_url_with_auth}"

    try:
        # Change to the specified Git directory before pushing
        command = ['git', '-C', git_dir, 'push', repo_url_with_auth, f"{from_branch}:{to_branch}", '--force']
        print(f"Running command: {' '.join(command)}")  # Debugging output
        subprocess.run(command, check=True)
        print(f"Successfully force pushed from {from_branch} to {to_branch} on {repo_url}.")
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while pushing to the remote repository: {e}")

def main():
    parser = argparse.ArgumentParser(description='Force push changes from a specified branch to a target branch in a remote git repository.')
    parser.add_argument('--from', dest='from_branch', type=str, required=True, help='The name of the source branch to push from.')
    parser.add_argument('--to', dest='to_branch', type=str, required=True, help='The name of the target branch to push to.')
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
    force_push(parsed_repo_url, args.from_branch, args.to_branch, args.username, code_host_api_key, args.git_dir)

if __name__ == "__main__":
    main()

#### Usage Instructions
#
#You can now run the script with the new arguments:
#
#```bash
#poetry run python scripts/force_push.py --from <source-branch> --to <target-branch> --remote origin --git-dir data/git-projects/SWE-agent --username 7db9a
#```
#
#### Example
#
#If you want to force push from the `feature` branch to the `main` branch, you would use:
#
#```bash
#poetry run python scripts/force_push.py --from feature --to main --remote origin --git-dir data/git-projects/SWE-agent --username 7db9a
#```
#
#### Important Notes
#
#- Ensure that both the source branch (`from_branch`) and the target branch (`to_branch`) exist in your local repository before running the script.
#- Be cautious when using `--force`, as this can overwrite changes in the target branch on the remote repository.
#
#This should give you the flexibility you need to push changes between branches effectively. If you have any further questions or need additional modifications, feel free to ask!
