import unittest
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown  # Import the Teardown class
)

class TestMachtianiCommand(unittest.TestCase):

    def setUp(self):
        # Set the directory for the test
        self.directory = "data/git-projects/chastler"
        # Initialize the Teardown class with the git project directory
        self.teardown = Teardown(self.directory)

    def test_run_machtiani_command(self):
        command = 'machtiani git-store --branch-name "master" --force'

        # Perform the command execution
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        # Clean the output
        stdout_normalized = clean_output(stdout_machtiani)

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

    def tearDown(self):
        """Clean up the test environment by running the git delete command."""
        try:
            stdout, stderr = self.teardown.run_git_delete()
            # Optionally, you can log or print the output of the teardown
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
