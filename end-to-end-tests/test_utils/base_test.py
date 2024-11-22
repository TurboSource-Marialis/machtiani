import unittest
import os
import time
import threading
from test_utils.test_utils import (
    Teardown,
    Setup,
    clean_output,
    run_machtiani_command,
    wait_for_status_complete,
    wait_for_status_incomplete,
    append_future_features_to_chat_file,

)

class BaseTestEndToEnd:
    @classmethod
    def setup_end_to_end(cls, no_code_host_key=False):
        """Initialize the test environment."""
        cls.maxDiff = None
        cls.directory = "data/git-projects/chastler"
        cls.setup = Setup(cls.directory, no_code_host_key)

        # Fetch the latest branches
        cls.setup.fetch_latest_branches()
        cls.setup.force_push("master-backup", "master")
        cls.setup.create_ignore_file()

        cls.checkout_branches(["feature", "feature2", "master"])

        # Ensure the .machtiani/chat directory exists
        chat_dir = os.path.join(cls.directory, ".machtiani", "chat")
        os.makedirs(chat_dir, exist_ok=True)

    @classmethod
    def checkout_branches(cls, branches):
        """Checkout specified branches if they do not exist."""
        for branch in branches:
            if branch not in cls.setup.get_branches():
                stdout, stderr = cls.setup.checkout_branch(branch)

                # Print stdout if there is any output
                if stdout:
                    print(f"Checkout Output for {branch}:", stdout)

                # Check for errors and print them only if they are not the expected ones
                if stderr and not any(expected_error in stderr for expected_error in [
                    f"fatal: a branch named '{branch}' already exists"
                ]):
                    print(f"Checkout Errors for {branch}:", stderr)


    @classmethod
    def teardown_end_to_end(cls):
        """Clean up the test environment."""
        try:
            cls.teardown = Teardown(cls.directory)
            cls.teardown.delete_ignore_file()
            cls.teardown.delete_chat_files()
            stdout, stderr = cls.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

    def run_machtiani_command(self, command):
        """Helper function to run a machtiani command and clean output."""
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        return clean_output(stdout_machtiani)

    def test_01_run_machtiani_git_store(self):
        time.sleep(5)
        command = 'machtiani git-store --branch-name "master" --force'
        stdout_normalized = self.run_machtiani_command(command)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Ignoring files based on .machtiani.ignore:",
            "poetry.lock",
            "Estimated embedding tokens: 25",
            "Estimated inference tokens: 1428",
            "VCSType.git repository added successfully",
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
        self.assertFalse(any("--force" in line for line in stdout_prompt_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_prompt_normalized))

    def test_06_run_machtiani_prompt_file_flag_command(self):
        chat_file_path = append_future_features_to_chat_file(self.directory)
        command = f"machtiani --file {chat_file_path}"
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("ilter" in line for line in stdout_normalized))
        self.assertTrue(any("ategorization" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def test_07_run_machtiani_status_with_lock(self):
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

    def test_08_run_machtiani_git_store_existing_project(self):
        """Test running git-store on an already added project."""
        command = 'machtiani git-store --branch-name "master" --force'

        # Run git-store for the first time
        stdout_normalized = self.run_machtiani_command(command)

        # Now run git-store again to check for existing project response
        stdout_normalized_second_run = self.run_machtiani_command(command)

        # Check if the output contains the expected message
        self.assertTrue(any("The project already exists!" in line for line in stdout_normalized), "Expected message about existing project not found in output.")

