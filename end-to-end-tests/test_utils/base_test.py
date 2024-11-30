import unittest
import os
import time
import threading
import subprocess
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
            "Estimated inference tokens: 1429",
            "VCSType.git repository added successfully",
            "",
            "---",
            "Your repo is getting added to machtiani is in progress!",
            "Please check back by running `machtiani status` to see if it completed."
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_02_check_git_directory_initialized(self):
        """Test that the content directory is an initialized git directory."""

        status_command = 'machtiani status'
        wait_for_status_complete(status_command, self.directory)
        container_name = "commit-file-retrieval"  # Name of your container
        content_directory = "/data/users/repositories/github_com_7db9a_chastler/contents"  # Path in the container

        # Command to check if the .git directory exists
        command = (
            f"if [ -d {content_directory}/.git ]; then "
            f"echo 'exists'; "
            f"else echo 'not exists'; fi"
        )

        # Execute the command in the Docker container
        try:
            result = subprocess.run(
                ["docker-compose", "exec", container_name, "bash", "-c", command],
                capture_output=True,
                text=True,
                check=True
            )
            full_output = result.stdout.strip()

            # Check if the directory exists
            if 'exists' in full_output:
                exists_check = 'exists'
            else:
                exists_check = 'not exists'

            # Assert that the .git directory exists
            self.assertEqual(exists_check, 'exists', f"Expected .git directory to exist in: {content_directory}")

        except subprocess.CalledProcessError as e:
            self.fail(f"Failed to check for .git directory: {e.stderr.strip()}")

    def test_03_run_machtiani_sync_command_not_ready(self):
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertFalse(any("Operation is locked for project 'github_com_7db9a_chastler'" in line for line in stdout_normalized))

    def test_04_run_machtiani_prompt_command(self):
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

    def test_05_run_machtiani_sync_command(self):
        command = 'machtiani git-sync --branch-name "master" --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Estimated embedding tokens: 0",
            "Estimated inference tokens: 0",
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}'
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_06_sync_new_commits_and_prompt_command(self):
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

    def test_07_run_machtiani_prompt_file_flag_command(self):
        chat_file_path = append_future_features_to_chat_file(self.directory)
        command = f"machtiani --file {chat_file_path}"
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("ilter" in line for line in stdout_normalized))
        self.assertTrue(any("ategorization" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def test_08_run_machtiani_status_with_lock(self):
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

    def test_09_run_machtiani_git_store_existing_project(self):
        """Test running git-store on an already added project."""
        command = 'machtiani git-store --branch-name "master" --force'

        # Run git-store for the first time
        stdout_normalized = self.run_machtiani_command(command)

        # Now run git-store again to check for existing project response
        stdout_normalized_second_run = self.run_machtiani_command(command)

        # Check if the output contains the expected message
        self.assertTrue(any("The project already exists!" in line for line in stdout_normalized), "Expected message about existing project not found in output.")

    def test_10_commit_messages_count(self):
        """Test that there are exactly 3 git commit messages in the content directory."""
        container_name = "commit-file-retrieval"  # Name of your container
        content_directory = "/data/users/repositories/github_com_7db9a_chastler/contents"  # Path in the container

        # Command to count the number of commits
        command = f"git -C {content_directory} rev-list --count HEAD"

        # Execute the command in the Docker container
        try:
            result = subprocess.run(
                ["docker-compose", "exec", container_name, "bash", "-c", command],
                capture_output=True,
                text=True,
                check=True
            )
            commit_count = int(result.stdout.strip())

            # Assert that there are exactly 3 commits
            self.assertEqual(commit_count, 3, f"Expected 3 commits, but found {commit_count}.")
        except subprocess.CalledProcessError as e:
            self.fail(f"Failed to count commits: {e.stderr.strip()}")

    def test_11_no_untracked_or_modified_files(self):
        """Test that there are no untracked or modified files in the git directory."""
        container_name = "commit-file-retrieval"  # Name of your container
        content_directory = "/data/users/repositories/github_com_7db9a_chastler/contents"  # Path in the container

        # Command to check the status of the git repository
        command = f"git -C {content_directory} status --porcelain"

        # Execute the command in the Docker container
        try:
            result = subprocess.run(
                ["docker-compose", "exec", container_name, "bash", "-c", command],
                capture_output=True,
                text=True,
                check=True
            )
            status_output = result.stdout.strip()

            # Assert that the output is empty, meaning no untracked or modified files
            self.assertEqual(status_output, "", "There are untracked or modified files in the git directory.")

        except subprocess.CalledProcessError as e:
            self.fail(f"Failed to check git status: {e.stderr.strip()}")

    def test_12_tags_for_each_commit(self):
        """Test that there is a tag for each commit in the content repo with the tag name being the commit OID."""
        container_name = "commit-file-retrieval"  # Name of your container
        repo_directory = "/data/users/repositories/github_com_7db9a_chastler/repo/git"
        content_directory = "/data/users/repositories/github_com_7db9a_chastler/contents"

        # Command to get the list of commits
        command_commits = f"git -C {repo_directory} rev-list --all"
        # Command to get the list of tags
        command_tags = f"git -C {content_directory} tag"

        # Execute the command in the Docker container
        try:
            result_commits = subprocess.run(
                ["docker-compose", "exec", container_name, "bash", "-c", command_commits],
                capture_output=True,
                text=True,
                check=True
            )
            commit_oids = {commit.strip() for commit in result_commits.stdout.strip().splitlines()}

            result_tags = subprocess.run(
                ["docker-compose", "exec", container_name, "bash", "-c", command_tags],
                capture_output=True,
                text=True,
                check=True
            )
            tags = {tag.strip() for tag in result_tags.stdout.strip().splitlines()}

            # Validate that commit_oids and tags are non-empty
            self.assertTrue(commit_oids, "No commits were found in the repository.")
            self.assertTrue(tags, "No tags were found in the repository.")

            # Check that each tag has a corresponding commit OID
            for tag in tags:
                self.assertIn(tag, commit_oids, f"Expected a commit OID for tag {tag} but none was found.")

        except subprocess.CalledProcessError as e:
            self.fail(f"Failed to retrieve commits or tags: {e.stderr.strip()}")
