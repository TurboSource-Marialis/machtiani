import unittest
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown,
    Setup  # Import the Setup class
)

class TestMachtianiSyncCommand(unittest.TestCase):

    def setUp(self):
        # Set the directory for the test
        self.directory = "data/git-projects/chastler"
        # Initialize the Setup class with the git project directory
        self.setup = Setup(self.directory)

        # Run git-store to set up the test environment
        self.stdout_setup, self.stderr_setup = self.setup.run_git_store()

        # Optionally, you can log or print the output of the setup
        print("Setup Output:", self.stdout_setup)
        print("Setup Errors:", self.stderr_setup)

    def test_run_machtiani_sync_command(self):
        # The test command for syncing
        command = 'machtiani git-sync --branch-name "master" --force'

        # Run the command and capture the output
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        # Clean the output for comparison
        stdout_normalized = clean_output(stdout_machtiani)

        print(stdout_normalized)

        # Define the expected output for the command (updated)
        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Parsed file paths from machtiani.ignore:",
            "poetry.lock",
            "Estimated input tokens: 0",
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}'
        ]

        # Normalize the expected output
        expected_output = [line.strip() for line in expected_output if line.strip()]

        # Assert that the normalized output matches the expected output
        self.assertEqual(stdout_normalized, expected_output)



    def test_04a_git_sync_invalid_flag_format(self):
        """Test that the git-sync command fails properly with invalid flag format."""
        command = 'machtiani git-sync amplify low --depth 1'

        stdout_raw, stderr_raw = run_machtiani_command(command, self.directory)
        stdout_clean = clean_output(stdout_raw)
        stderr_clean = clean_output(stderr_raw)
        combined = stdout_clean + stderr_clean

        # Check for proper error message
        self.assertTrue(any("Error in command arguments" in line for line in combined))
        self.assertTrue(any("invalid flag format: 'amplify'" in line for line in combined))
        self.assertTrue(any("Did you mean '--amplify'?" in line for line in combined))


    def test_04b_git_sync_invalid_amplify_value(self):
        """Test that the git-sync command validates amplify values."""
        command = 'machtiani git-sync --amplify invalid --depth 1'

        stdout_raw, stderr_raw = run_machtiani_command(command, self.directory)
        stdout_clean = clean_output(stdout_raw)
        stderr_clean = clean_output(stderr_raw)
        combined = stdout_clean + stderr_clean

        self.assertTrue(any("invalid value for --amplify" in line for line in combined))
        self.assertTrue(any("Must be one of: off, low, mid, high" in line for line in combined))


    def test_04c_git_sync_valid_flags(self):
        """Test that the git-sync command works correctly with valid flags."""
        command = 'machtiani git-sync --amplify low --depth 1 --force'

        stdout_raw, _ = run_machtiani_command(command, self.directory)
        stdout_clean = clean_output(stdout_raw)

        # Check that the command executed successfully
        self.assertTrue(any("Successfully synced the repository" in line for line in stdout_clean))

    def tearDown(self):
        """Clean up the test environment by running the git delete command."""
        try:
            self.teardown = Teardown(self.directory)  # Initialize the Teardown class
            stdout, stderr = self.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
