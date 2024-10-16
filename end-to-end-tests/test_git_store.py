import unittest
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown,
    Setup  # Import the Setup class
)

class TestMachtianiCommand(unittest.TestCase):

    def setUp(self):
        # Set the directory for the test
        self.directory = "data/git-projects/chastler"
        # Initialize the Teardown class with the git project directory
        self.teardown = Teardown(self.directory)
        # Initialize the Setup class with the git project directory
        self.setup = Setup(self.directory)

        # Run git-store to set up the test environment
        self.stdout_setup, self.stderr_setup = self.setup.run_git_store()

        # Optionally, you can log or print the output of the setup
        print("Setup Output:", self.stdout_setup)
        print("Setup Errors:", self.stderr_setup)

    def test_run_machtiani_command(self):
        # The test logic remains unchanged
        command = 'machtiani git-store --branch-name "master" --force'

        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated input tokens: 48",
            "VCSType.git repository added successfully"
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]

        self.assertEqual(stdout_normalized, expected_output)

    def tearDown(self):
        """Clean up the test environment by running the git delete command."""
        try:
            stdout, stderr = self.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
