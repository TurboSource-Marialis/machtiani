import unittest
import subprocess
import os

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

class TestMachtianiCommand(unittest.TestCase):

    def test_run_machtiani_command(self):
        # Set the directory for the test
        directory = "data/git-projects/chastler"
        command = 'machtiani git-store --branch-name "master" --force'

        # Perform the command execution
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, directory)

        # Define function to identify progress indicator lines
        def is_progress_indicator(line):
            # Progress indicators are lines that consist solely of '|', '/', '-', or '\'
            return line.strip() in {'|', '/', '-', '\\'}

        # Remove progress indicator lines
        stdout_cleaned = [line.strip() for line in stdout_machtiani if not is_progress_indicator(line)]

        # Remove empty lines
        stdout_cleaned = [line for line in stdout_cleaned if line]

        # Normalize the lines by stripping leading/trailing whitespace
        stdout_normalized = [line.strip() for line in stdout_cleaned]

        # Expected output lines, also stripped of leading/trailing whitespace
        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated input tokens: 48",
            "VCSType.git repository added successfully"
        ]

        # Remove empty lines from expected output, in case they exist
        expected_output = [line.strip() for line in expected_output if line.strip()]

        # Compare the outputs line by line
        self.assertEqual(stdout_normalized, expected_output)

if __name__ == '__main__':
    unittest.main()

