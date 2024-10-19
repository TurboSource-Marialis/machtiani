import unittest
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown,
    Setup  # Import the Setup class
)

class TestMachtianiCommand(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.MaxDiff = None
        # Set the directory for the test
        cls.directory = "data/git-projects/chastler"
        # Initialize the Setup class with the git project directory
        cls.setup = Setup(cls.directory)

        # Fetch the latest branches
        fetch_res = cls.setup.fetch_latest_branches()
        print("Fetched branches:", fetch_res)
        cls.setup.force_push("master-backup", "master")

        branches = cls.setup.get_branches()
        if " feature" not in branches:  # Checking for local branch existence
            stdout, stderr = cls.setup.checkout_branch("feature")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)  # Will contain any errors if the checkout fails

    def test_run_machtiani_command(self):
        # The test logic remains unchanged
        command = 'machtiani git-store --branch-name "master" --force'

        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated embedding tokens: 25",
            "Estimated inference tokens: 10086",
            "VCSType.git repository added successfully"
            "",
            "---",
            "Your repo is getting added to machtiani is in progress!",
            "Please check back by running `machtiani status` to see if it completed."
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
