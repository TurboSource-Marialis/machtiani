import os
import subprocess

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
