import unittest
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

        ## Run git-store to set up the test environment
        #cls.stdout_setup, cls.stderr_setup = cls.setup.run_git_store()

        ## Optionally, log or print the output of the setup
        #print("Setup Output:", cls.stdout_setup)
        #print("Setup Errors:", cls.stderr_setup)

    def test_run_machtiani_git_store(self):
        command = 'machtiani git-store --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated input tokens: 25",
            "VCSType.git repository added successfully"
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_run_machtiani_prompt_command(self):
        command = 'machtiani "what does the readme say?" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("Estimated input tokens: 7" in line for line in stdout_normalized))
        self.assertTrue(any("chastler" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def test_run_machtiani_sync_command(self):
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Parsed file paths from machtiani.ignore:",
            "poetry.lock",
            "Estimated input tokens: 0",
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}'
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
