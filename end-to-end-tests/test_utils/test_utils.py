import os
import subprocess

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

        # Clean and return the output
        cleaned_output = clean_output(stdout)
        cleaned_error = clean_output(stderr)

        return cleaned_output, cleaned_error

    def force_push(self, from_branch, to_branch):
        """Force push changes from the specified branch to a target branch on the remote repository without authentication."""
        return force_push(from_branch, to_branch, self.git_directory)

    def checkout_branch(self, branch_name):
        """Checkout the specified branch from the remote repository."""
        return checkout_branch(branch_name, self.git_directory)

    def get_branches(self):
        stdout, stderr = run_machtiani_command('git branch', self.git_directory)
        return stdout  # Directly return stdout since it's already a list

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

    def run_git_store(self):
        """Run 'machtiani git-store --branch-name "master" --force' in the specified git directory."""
        command = 'machtiani git-store --branch-name "master" --force'
        stdout, stderr = run_machtiani_command(command, self.git_directory)

        # Clean and return the output
        cleaned_output = clean_output(stdout)
        cleaned_error = clean_output(stderr)

        return cleaned_output, cleaned_error

    def fetch_latest_branches(self):
        """Run the fetch_latest_branches.py script to fetch the latest branches."""
        #script_path = 'test_utils/fetch_latest_branches.py'
        #command = f'python3 {script_path} --git-dir {self.git_directory}'
        res = get_latest_branches(self.git_directory)

        return res

    def force_push(self, from_branch, to_branch):
        """Force push changes from the specified branch to a target branch on the remote repository without authentication."""
        return force_push(from_branch, to_branch, self.git_directory)

    def checkout_branch(self, branch_name):
        """Checkout the specified branch from the remote repository."""
        return checkout_branch(branch_name, self.git_directory)

    def get_branches(self):
        stdout, stderr = run_machtiani_command('git branch', self.git_directory)
        return stdout  # Directly return stdout since it's already a list

    def create_ignore_file(self):
        """Create a .machtiani.ignore file in the git directory with the specified content."""
        ignore_file_path = os.path.join(self.git_directory, '.machtiani.ignore')
        with open(ignore_file_path, 'w') as f:
            f.write('poetry.lock\n')
        print(f"Created .machtiani.ignore file with content: 'poetry.lock'")


def clean_output(stdout_lines):
    """Utility function to clean the output from the command."""
    def is_progress_indicator(line):
        return line.strip() in {'|', '/', '-', '\\'}

    cleaned = [line.strip() for line in stdout_lines if not is_progress_indicator(line)]
    return [line.strip() for line in cleaned if line]

def run_machtiani_command(command, directory):
    # Ensure the specified directory exists
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
        # Change to the specified Git directory
        command = ['git', '-C', git_dir, 'fetch', '--prune']
        print(f"Running command: {' '.join(command)}")  # Debugging output
        subprocess.run(command, check=True)

        # Get the branches
        command = ['git', '-C', git_dir, 'branch', '-r']
        print(f"Running command: {' '.join(command)}")  # Debugging output
        branches = subprocess.check_output(command, text=True).strip().split('\n')
        return [branch.strip().replace('origin/', '') for branch in branches if branch]
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while fetching branches: {e}")
        return []

def force_push(from_branch, to_branch, git_dir):
    """Force push changes from the specified branch to a target branch on the remote repository without authentication."""
    try:
        # Use 'origin' as the default remote
        command = ['git', '-C', git_dir, 'push', 'origin', f"{from_branch}:{to_branch}", '--force']
        subprocess.run(command, check=True)
        print(f"Successfully force pushed from {from_branch} to {to_branch} on origin.")
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while pushing to the remote repository: {e}")

def checkout_branch(branch_name, git_directory):
    """Checkout the specified branch from the remote repository."""
    print(f"Attempting to checkout branch '{branch_name}' in directory: {git_directory}")
    command = f'git checkout -b {branch_name} origin/{branch_name}'
    try:
        stdout, stderr = run_machtiani_command(command, git_directory)
        return stdout, stderr
    except subprocess.CalledProcessError as e:
        print(f"An error occurred while checking out branch '{branch_name}': {e}")
        return [], [str(e)]
