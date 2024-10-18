import unittest
import time
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown,
    Setup
)

class TestEndToEndMachtianiCommands(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        # Set the directory for the test
        cls.directory = "data/git-projects/chastler"
        # Initialize the Setup class with the git project directory
        cls.setup = Setup(cls.directory)

        # Fetch the latest branches
        fetch_res = cls.setup.fetch_latest_branches()
        print("Fetched branches:", fetch_res)
        cls.setup.force_push("master-backup", "master")

        branches = cls.setup.get_branches()
        if "feature" not in branches:  # Checking for local branch existence
            stdout, stderr = cls.setup.checkout_branch("feature")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)
        cls.setup.run_git_store()

    def test_run_machtiani_status_command(self):
        command = 'machtiani status'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Project is ready for chat!"
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    @classmethod
    def tearDownClass(cls):
        """Clean up the test environment by running the git delete command."""
        try:
            cls.teardown = Teardown(cls.directory)
            stdout, stderr = cls.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
