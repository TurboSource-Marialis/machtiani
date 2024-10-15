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

def clean_output(stdout_lines):
    """Utility function to clean the output from the command."""
    # Define function to identify progress indicator lines
    def is_progress_indicator(line):
        # Progress indicators are lines that consist solely of '|', '/', '-', or '\'
        return line.strip() in {'|', '/', '-', '\\'}

    # Remove progress indicator lines
    cleaned = [line.strip() for line in stdout_lines if not is_progress_indicator(line)]
    # Remove empty lines and normalize the lines by stripping leading/trailing whitespace
    return [line.strip() for line in cleaned if line]

def run_machtiani_command(command, directory):
    # Ensure the specified directory exists
    if not os.path.isdir(directory):
        raise FileNotFoundError(f"The directory {directory} does not exist.")

    # Start the process in the specified directory
    process = subprocess.Popen(
        command,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        universal_newlines=True,
        shell=True,
        cwd=directory  # Run command in the specified directory
    )

    # Collect stdout and stderr in real-time
    stdout, stderr = [], []
    while True:
        stdout_line = process.stdout.readline()
        stderr_line = process.stderr.readline()

        # Break if both are empty and the process is done
        if not stdout_line and not stderr_line and process.poll() is not None:
            break

        if stdout_line:
            stdout.append(stdout_line.rstrip('\n'))  # Keep the line, removing the trailing newline
        if stderr_line:
            stderr.append(stderr_line.strip())

    process.stdout.close()
    process.stderr.close()

    return stdout, stderr
