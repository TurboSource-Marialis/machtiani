import os
import subprocess
import time
import yaml

class Teardown:
    def __init__(self, git_directory):
        """Initialize the Teardown class with the path to the git project directory."""
        if not os.path.isdir(git_directory):
            raise ValueError(f"The specified directory '{git_directory}' is not a valid directory.")
        self.git_directory = git_directory

    def run_git_delete(self):
        """Run 'machtiani git-delete --force' in the specified git directory."""
        command = "machtiani git-delete --force"
        stdout, stderr = run_machtiani_command(command, self.git_directory)
        return clean_output(stdout), clean_output(stderr)

    def force_push(self, from_branch, to_branch):
        """Force push changes from the specified branch to a target branch on the remote repository without authentication."""
        return force_push(from_branch, to_branch, self.git_directory)

    def checkout_branch(self, branch_name):
        """Checkout the specified branch from the remote repository."""
        return checkout_branch(branch_name, self.git_directory)

    def get_branches(self):
        stdout, stderr = run_machtiani_command('git branch', self.git_directory)
        return stdout

    def delete_ignore_file(self):
        """Delete the .machtiani.ignore file if it exists."""
        ignore_file_path = os.path.join(self.git_directory, '.machtiani.ignore')
        if os.path.exists(ignore_file_path):
            os.remove(ignore_file_path)
            print(f"Deleted .machtiani.ignore file.")
        else:
            print(f".machtiani.ignore file does not exist.")

class Setup:
    def __init__(self, git_directory):
        """Initialize the Setup class with the path to the git project directory."""
        if not os.path.isdir(git_directory):
            raise ValueError(f"The specified directory '{git_directory}' is not a valid directory.")
        self.git_directory = git_directory
        self.git_operations = GitOperations(git_directory)  # Initialize GitOperations

    def run_git_store(self):
        """Run 'machtiani git-store --branch-name "master" --force' in the specified git directory."""
        command = 'machtiani git-store --branch-name "master" --force'
        stdout, stderr = run_machtiani_command(command, self.git_directory)
        return clean_output(stdout), clean_output(stderr)

    def fetch_latest_branches(self):
        """Fetch the latest branches from the remote repository."""
        return get_latest_branches(self.git_directory)

    def force_push(self, from_branch, to_branch):
        """Force push changes from the specified branch to a target branch on the remote repository using authentication."""
        return self.git_operations.run_git_push(from_branch, to_branch)

    def checkout_branch(self, branch_name):
        """Checkout the specified branch from the remote repository."""
        return checkout_branch(branch_name, self.git_directory)

    def get_branches(self):
        stdout, stderr = run_machtiani_command('git branch', self.git_directory)
        return stdout

    def create_ignore_file(self):
        """Create a .machtiani.ignore file in the git directory with the specified content."""
        ignore_file_path = os.path.join(self.git_directory, '.machtiani.ignore')
        with open(ignore_file_path, 'w') as f:
            f.write('poetry.lock\n')
        print(f"Created .machtiani.ignore file with content: 'poetry.lock'")

class GitOperations:
    def __init__(self, git_directory):
        self.git_directory = git_directory
        self.api_key = self.load_config_values()

    def load_config_values(self):
        """Load the configuration values from the specified YAML config file."""
        config_file_path = os.path.join(self.git_directory, '.machtiani-config.yml')
        with open(config_file_path, 'r') as file:
            config = yaml.safe_load(file)
        api_key = config['environment']['CODE_HOST_API_KEY']
        return api_key

    def get_remote_url(self):
        """Get the remote URL from the git repository in the specified directory."""
        command = "git remote get-url origin"
        result = run_machtiani_command(command, self.git_directory)
        if result[1]:
            raise Exception("Error fetching remote URL: " + " ".join(result[1]))
        return result[0][0]

    def construct_auth_url(self, remote_url):
        """Construct the authentication URL for the Git operation."""
        if remote_url.startswith("https://"):
            # Ensure we only add the api_key once
            if "@" in remote_url:
                raise ValueError("Remote URL already contains authentication information.")
            return remote_url.replace("https://", f"https://{self.api_key}@")
        elif remote_url.startswith("git@"):
            # SSH URLs do not need username:password format
            return remote_url  # SSH does not need username:password format
        else:
            raise ValueError("Unsupported remote URL format")

    def run_git_push(self, from_branch, to_branch):
        """Force push changes from the specified branch to a target branch on the remote repository using authentication."""
        remote_url = self.get_remote_url()
        auth_url = self.construct_auth_url(remote_url)

        # Store the original remote URL
        original_remote_url = remote_url

        try:
            # Set the remote URL with the auth URL
            command_set_remote = f"git remote set-url origin {auth_url}"
            run_machtiani_command(command_set_remote, self.git_directory)

            command_push = f"git push origin {from_branch}:{to_branch} --force"
            stdout, stderr = run_machtiani_command(command_push, self.git_directory)
            return stdout, stderr

        except Exception as e:
            print(f"An error occurred during push: {e}")
            raise  # Reraise the exception for handling outside if necessary

        finally:
            # Revert the remote URL back to the original
            command_revert_remote = f"git remote set-url origin {original_remote_url}"
            run_machtiani_command(command_revert_remote, self.git_directory)

def clean_output(stdout_lines):
    """Utility function to clean the output from the command."""
    def is_progress_indicator(line):
        return line.strip() in {'|', '/', '-', '\\'}

    cleaned = [line.strip() for line in stdout_lines if not is_progress_indicator(line)]
    return [line.strip() for line in cleaned if line]

def run_machtiani_command(command, directory):
    """Run a shell command in the specified directory and return the output."""
    if not os.path.isdir(directory):
        raise FileNotFoundError(f"The directory {directory} does not exist.")

    process = subprocess.Popen(
        command,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        universal_newlines=True,
        shell=True,
        cwd=directory
    )

    stdout, stderr = [], []
    while True:
        stdout_line = process.stdout.readline()
        stderr_line = process.stderr.readline()

        if not stdout_line and not stderr_line and process.poll() is not None:
            break

        if stdout_line:
            stdout.append(stdout_line.rstrip('\n'))
        if stderr_line:
            stderr.append(stderr_line.strip())

    process.stdout.close()
    process.stderr.close()

    return stdout, stderr

def get_latest_branches(git_dir):
    """Fetch the latest branches from the remote repository."""
    try:
        command = ['git', '-C', git_dir, 'fetch', '--prune']
        subprocess.run(command, check=True)

        command = ['git', '-C', git_dir, 'branch', '-r']
        branches = subprocess.check_output(command, text=True).strip().split('\n')
        return [branch.strip().replace('origin/', '') for branch in branches if branch]
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while fetching branches: {e}")
        return []

def force_push(from_branch, to_branch, git_dir):
    """Force push changes from the specified branch to a target branch on the remote repository without authentication."""
    try:
        command = ['git', '-C', git_dir, 'push', 'origin', f"{from_branch}:{to_branch}", '--force']
        subprocess.run(command, check=True)
        print(f"Successfully force pushed from {from_branch} to {to_branch} on origin.")
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while pushing to the remote repository: {e}")

def checkout_branch(branch_name, git_directory):
    """Checkout the specified branch from the remote repository."""
    command = f'git checkout -b {branch_name} origin/{branch_name}'
    try:
        stdout, stderr = run_machtiani_command(command, git_directory)
        return stdout, stderr
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while checking out branch '{branch_name}': {e}")
        return [], [str(e)]

def wait_for_status(command, expected_output, directory, max_wait_time=30, interval=1):
    """Wait for a command to return the expected output by polling."""
    elapsed_time = 0
    while elapsed_time < max_wait_time:
        stdout_status, stderr_status = run_machtiani_command(command, directory)
        stdout_status_normalized = clean_output(stdout_status)

        if stdout_status_normalized == expected_output:
            return True

        time.sleep(interval)
        elapsed_time += interval

    return False
