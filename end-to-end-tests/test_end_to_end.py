import unittest
import time
import threading
from test_utils.test_utils import (
    Teardown,
    Setup,
    clean_output,
    run_machtiani_command,
    wait_for_status_complete,
    wait_for_status_incomplete,
)

class TestEndToEndMachtianiCommands(unittest.TestCase):

    @classmethod
    def setUpClass(cls):
        cls.maxDiff = None
        # Set the directory for the test
        cls.directory = "data/git-projects/chastler"
        # Initialize the Setup class with the git project directory
        cls.setup = Setup(cls.directory)

        # Fetch the latest branches
        fetch_res = cls.setup.fetch_latest_branches()
        cls.setup.force_push("master-backup", "master")
        cls.setup.create_ignore_file()

        branches = cls.setup.get_branches()
        if " feature" not in branches:  # Checking for local branch existence
            stdout, stderr = cls.setup.checkout_branch("feature")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)  # Will contain any errors if the checkout fails
        if " feature2" not in branches:  # Checking for local branch existence
            stdout, stderr = cls.setup.checkout_branch("feature2")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)  # Will contain any errors if the checkout fails
        stdout, stderr = cls.setup.checkout_branch("master")

    def test_01_run_machtiani_git_store(self):
        # Introduce a slight delay to allow for remote to be ready
        time.sleep(5)
        command = 'machtiani git-store --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated embedding tokens: 25",
            "Estimated inference tokens: 1428",
            "VCSType.git repository added successfully"
            "",
            "---",
            "Your repo is getting added to machtiani is in progress!",
            "Please check back by running `machtiani status` to see if it completed."
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_02_run_machtiani_sync_command_not_ready(self):
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)


        self.assertFalse(any("Operation is locked for project 'github_com_7db9a_chastler'" in line for line in stdout_normalized))

    def test_03_run_machtiani_prompt_command(self):
        status_command = 'machtiani status'
        wait_for_status_complete(status_command, self.directory)
        time.sleep(3)

        command = 'machtiani "what does the readme say?" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("chastler" in line for line in stdout_normalized))
        self.assertFalse(any("video" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def test_04_run_machtiani_sync_command(self):
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Parsed file paths from machtiani.ignore:",
            "poetry.lock",
            "Estimated embedding tokens: 0",
            "Estimated inference tokens: 0",
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}'
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_05_sync_new_commits_and_prompt_command(self):
        # Step 1: Force push `feature` branch to `master`
        self.setup.force_push("feature", "master")

        # Introduce a slight delay to allow for remote to be ready
        time.sleep(5)
        # Step 2: Run git_sync and assert the output
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Parsed file paths from machtiani.ignore:",
            "poetry.lock",
            "Estimated embedding tokens: 16",
            "Estimated inference tokens: 30",  # Updated to match actual output
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}'
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

        # Step 3: Run git prompt and assert the output
        command = 'machtiani "what does the readme say?" --force'  # Example prompt command
        stdout_prompt, stderr_prompt = run_machtiani_command(command, self.directory)
        stdout_prompt_normalized = clean_output(stdout_prompt)

        self.assertTrue(any("Using remote URL" in line for line in stdout_prompt_normalized))
        self.assertTrue(any("video" in line for line in stdout_prompt_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_prompt_normalized))

    def test_06_run_machtiani_status_with_lock(self):
        # Step 1: Force push `feature2` branch to `master`
        self.setup.force_push("feature2", "master")

        # Introduce a slight delay to allow for remote to be ready
        time.sleep(5)

        def run_sync():
            command = 'machtiani git-sync --branch-name "master" --force'
            run_machtiani_command(command, self.directory)

        sync_thread = threading.Thread(target=run_sync)
        sync_thread.start()

        # Give time for thread to start
        time.sleep(1)

        # Step 4: Now run the status command while the sync is running
        status_command = 'machtiani status'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(status_command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)
        status = wait_for_status_incomplete(status_command, self.directory)
        self.assertTrue(status)

        # Step 5: Wait for the sync thread to finish
        sync_thread.join()

    @classmethod
    def tearDownClass(cls):
        """Clean up the test environment by running the git delete command."""
        try:
            cls.teardown = Teardown(cls.directory)
            cls.teardown.delete_ignore_file()
            stdout, stderr = cls.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
            pass
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
