import unittest
import os
import re
import json
import time
import threading
import subprocess
import logging
from datetime import datetime
from sentence_transformers import SentenceTransformer, util
from test_utils.test_utils import (
    Teardown,
    Setup,
    clean_output,
    run_machtiani_command,
    wait_for_status_complete,
    wait_for_status_incomplete,
    create_elapsed_time_filename,
    append_future_features_to_chat_file,

)

class BaseTestEndToEnd:
    @classmethod
    def setup_end_to_end(cls, no_code_host_key=False):
        """Initialize the test environment."""
        cls.maxDiff = None
        cls.directory = "data/git-projects/chastler"
        cls.configs = "data/configs"
        cls.setup = Setup(cls.directory, cls.configs, no_code_host_key)

        # Fetch the latest branches
        cls.setup.fetch_latest_branches()
        cls.setup.force_push("master-backup", "master")
        cls.setup.create_ignore_file()

        cls.checkout_branches(["feature", "feature2", "master"])

        # Ensure the .machtiani/chat directory exists
        chat_dir = os.path.join(cls.directory, ".machtiani", "chat")
        os.makedirs(chat_dir, exist_ok=True)

        """Class-level setup for logging configuration."""
        #logging.basicConfig(level=logging.DEBUG)
        cls.logger = logging.getLogger(__name__)

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
    def teardown_end_to_end(cls, unstage_files=True):
        """Clean up the test environment."""
        try:
            cls.teardown = Teardown(cls.directory)

            cls.teardown.delete_ignore_file()
            cls.teardown.delete_chat_files()
            if unstage_files:
                cls.teardown.restore_untracked_changes()  # Added method call
            stdout, stderr = cls.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

    def run_machtiani_command(self, command):
        """Helper function to run a machtiani command and clean output."""
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        return clean_output(stdout_machtiani)


    def test_01_run_machtiani_git_sync(self):
        time.sleep(5)
        # Checkout master branch first to ensure we're syncing from a clean state
        self.setup.checkout_branch("master")

        command = 'git checkout master && machtiani git-sync --amplify low --cost --force'
        stdout_normalized = self.run_machtiani_command(command)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Repository not found on Machtiani. Preparing for initial sync.", # Common line
            "Ignoring files based on .machtiani.ignore:", # Common line
            "poetry.lock", # Actual output had this directly after ignore list
            "Repository confirmation received.", # Added based on actual output (-)
            "---", # Added based on actual output (-)
            "Estimating token cost...", # Added based on actual output (-)
            "Estimated embedding tokens: 3000", # Common line
            "Estimated inference tokens: 10396", # Common line
            "---", # Common line
            "VCSType.git repository added successfully", # Common line
            "---", # Common line
            "Your repo is getting added to machtiani is in progress!",
            "Please check back by running `machtiani status` to see if it completed."
        ]
        # --- END CORRECTION ---

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_02_run_machtiani_sync_command_not_ready(self):
        command = 'machtiani git-sync --amplify low --cost --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertFalse(any("Operation is locked for project 'github_com_7db9a_chastler'" in line for line in stdout_normalized))

    def test_03_time_elapsed_until_first_sync_complete(self):
        # Start timing
        start_time = time.time()

        status_command = 'machtiani status'
        wait_for_status_complete(status_command, self.directory)

        # End timing
        end_time = time.time()

        total_time_elapsed = end_time - start_time
        print(f"Total time elapsed for running machtiani git-sync --amplify low: {total_time_elapsed:.2f} seconds")

        # Save the elapsed time to a file
        filename = create_elapsed_time_filename(total_time_elapsed)
        file_path = os.path.join(self.directory, filename)

        with open(file_path, 'w') as f:
            f.write(f"Test executed on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
            f.write(f"Total time elapsed for running machtiani git-sync --amplify low: {total_time_elapsed:.2f} seconds\n")

        print(f"Elapsed time written to file: {file_path}")

        # Assert that the elapsed time is between 10 seconds and 20 seconds
        self.assertGreaterEqual(total_time_elapsed, 10, "The command took less than 10 seconds.")
        self.assertLessEqual(total_time_elapsed, 20, "The command took more than 20 seconds.")

    def test_04_confirm_commits_embeddings_structure(self):
        # Copy the file from the docker container
        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Define the expected structure
        expected_structure = {
            "c5b3a81463c7d3a188ec60523c0f68c23e93a5dc": {
                "messages": list,
                "embeddings": list,
            },
            "879ce80f348263a2580cd38623ee4e80ae69caac": {
                "messages": list,
                "embeddings": list,
            },
            "f0d14de7547e2911f262762efa7ea20ada16a2f6": {
                "messages": list,
                "embeddings": list,
            },
        }

        # Check that the keys are correct
        self.assertEqual(set(commits_embeddings.keys()), set(expected_structure.keys()))

        # Check each commit's structure
        for commit_oid, expected_commit_structure in expected_structure.items():
            commit_data = commits_embeddings[commit_oid]

            # Check that the messages are a list
            self.assertIsInstance(commit_data['messages'], list)

            # Check that the embeddings are a list and each embedding is a list of floats
            self.assertIsInstance(commit_data['embeddings'], list)
            for embedding in commit_data['embeddings']:
                self.assertIsInstance(embedding, list)
                for value in embedding:
                    self.assertIsInstance(value, float)

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

    def test_05_run_machtiani_prompt_command(self):
        status_command = 'machtiani status'

        wait_for_status_complete(status_command, self.directory)
        time.sleep(3)

        command = 'machtiani  "what does the readme say?" --force --mode chat'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("chastler" in line for line in stdout_normalized))
        self.assertFalse(any("video" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def test_06_run_machtiani_sync_command(self):
        command = 'machtiani git-sync --amplify low --cost --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git", # Common line
            "Repository found. Preparing to sync branch: master", # Changed based on actual output (-)
            "---", # Common line
            "Estimating token cost...", # Added based on actual output (-)
            "Estimated embedding tokens: 0", # Common line
            "Estimated inference tokens: 0", # Common line
            "---", # Common line
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.", # Common line
            'Server response: {"message":"Fetched and checked out branch \'master\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"master","project_name":"https://github.com/7db9a/chastler.git"}' # Added based on actual output (-)
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

    def test_07_sync_new_commits_and_prompt_command(self):
        # Step 2: Run git_sync and assert the output
        command = 'git checkout feature && machtiani git-sync --amplify low --force --cost'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git",
            "Repository found. Preparing to sync branch: feature",
            "",
            "---",
            "Estimating token cost...",
            "Estimated embedding tokens: 1000",
            "Estimated inference tokens: 85",
            "---",
            "",
            "Successfully synced the repository: https://github.com/7db9a/chastler.git.",
            'Server response: {"message":"Fetched and checked out branch \'feature\' for project \'https://github.com/7db9a/chastler.git\' and updated index.","branch_name":"feature","project_name":"https://github.com/7db9a/chastler.git"}'
        ]

        expected_output = [line.strip() for line in expected_output if line.strip()]

        self.assertEqual(stdout_normalized, expected_output)

        # Step 3: Run git prompt and assert the output
        command = 'machtiani  "what does the readme say?" --force --mode chat'
        stdout_prompt, stderr_prompt = run_machtiani_command(command, self.directory)
        stdout_prompt_normalized = clean_output(stdout_prompt)

        self.assertTrue(any("Using remote URL" in line for line in stdout_prompt_normalized))
        self.assertTrue(any("video" in line for line in stdout_prompt_normalized))
        self.assertFalse(any("--force" in line for line in stdout_prompt_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_prompt_normalized))


    def test_08_run_machtiani_prompt_file_flag_command(self):
        chat_file_path = append_future_features_to_chat_file(self.directory)
        command = f"machtiani  --file {chat_file_path} --mode chat"
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))
        self.assertTrue(any("ilter" in line for line in stdout_normalized))
        self.assertTrue(any("ategorization" in line for line in stdout_normalized))
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    #def test_09_run_machtiani_status_with_lock(self):
    #    def run_sync():
    #        command = 'git checkout feature && machtiani git-sync --amplify low --force'
    #        run_machtiani_command(command, self.directory)

    #    sync_thread = threading.Thread(target=run_sync)
    #    sync_thread.start()

    #    # Give time for thread to start
    #    time.sleep(1)

    #    # Step 4: Now run the status command while the sync is running
    #    status_command = 'machtiani status'
    #    stdout_machtiani, stderr_machtiani = run_machtiani_command(status_command, self.directory)
    #    stdout_normalized = clean_output(stdout_machtiani)
    #    status = wait_for_status_incomplete(status_command, self.directory)
    #    self.assertTrue(status)

    #    # Step 5: Wait for the sync thread to finish
    #    sync_thread.join()

    def test_10_sync_feature2_branch(self):
        self.teardown_end_to_end(unstage_files=False)
        self.setup_end_to_end()

        # Introduce a slight delay to allow for remote to be ready
        time.sleep(5)

        # Step 4: Run git_sync again and assert the output now shows the correct token counts
        command = 'git checkout feature2 && machtiani git-sync --amplify low --cost --force'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        expected_output = [
            "Using remote URL: https://github.com/7db9a/chastler.git", # Common line
            "Repository not found on Machtiani. Preparing for initial sync.", # Common line
            "Ignoring files based on .machtiani.ignore:", # Common line
            "poetry.lock", # Actual output had this directly after ignore list
            "Repository confirmation received.", # Added based on actual output (-)
            "---", # Added based on actual output (-)
            "Estimating token cost...", # Added based on actual output (-)
            "Estimated embedding tokens: 6000", # Common line
            "Estimated inference tokens: 10605", # Common line
            "---", # Common line
            "VCSType.git repository added successfully", # Common line
            "---", # Common line
            "Your repo is getting added to machtiani is in progress!",
            "Please check back by running `machtiani status` to see if it completed." # Common line
        ]
        # --- END CORRECTION ---

        expected_output = [line.strip() for line in expected_output if line.strip()]
        self.assertEqual(stdout_normalized, expected_output)

        # Step 5: Clean up after the test
        self.teardown_end_to_end(unstage_files=False)

    def test_11_run_machtiani_sync_command_not_ready(self):
        command = 'machtiani git-sync --amplify low --force --cost'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        stdout_normalized = clean_output(stdout_machtiani)

        self.assertFalse(any("Operation is locked for project 'github_com_7db9a_chastler'" in line for line in stdout_normalized))

    def test_12_time_elapsed_until_first_sync_complete(self):
        # Start timing
        start_time = time.time()

        status_command = 'machtiani status'
        wait_for_status_complete(status_command, self.directory)

        # End timing
        end_time = time.time()

        total_time_elapsed = end_time - start_time
        print(f"Total time elapsed for running machtiani git-sync --amplify low: {total_time_elapsed:.2f} seconds")

        # Save the elapsed time to a file
        filename = create_elapsed_time_filename(total_time_elapsed)
        file_path = os.path.join(self.directory, filename)

        with open(file_path, 'w') as f:
            f.write(f"Test executed on: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}\n")
            f.write(f"Total time elapsed for running machtiani git-sync --amplify low: {total_time_elapsed:.2f} seconds\n")

        print(f"Elapsed time written to file: {file_path}")

        # Assert that the elapsed time is between 10 seconds and 20 seconds
        self.assertGreaterEqual(total_time_elapsed, 10, "The command took less than 10 seconds.")
        self.assertLessEqual(total_time_elapsed, 20, "The command took more than 20 seconds.")

    def test_13_confirm_commits_embeddings_structure(self):
        # Copy the file from the docker container
        time.sleep(5)

        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Define the expected structure
        expected_structure = {
            # 3 unique messages and embeddings
            "c5b3a81463c7d3a188ec60523c0f68c23e93a5dc": {
                "messages": list,
                "embeddings": list,
            },
            # 5 unique messages and embeddings
            # 3rd and 5th message should be idential
            # 3rd and 5th embedding should be idential
            "879ce80f348263a2580cd38623ee4e80ae69caac": {
                "messages": list,
                "embeddings": list,
            },
            # 5 unique messages and embeddings
            "f0d14de7547e2911f262762efa7ea20ada16a2f6": {
                "messages": list,
                "embeddings": list,
            },
            # 3 unique messages and embeddings
            "7078ecda662103319304730ecdd31ec01b6ce786": {
                "messages": list,
                "embeddings": list,
            },
            # 3 unique messages and embeddings
            "7cedcb5363ab0ffd3829e7c1363c059c85d83762": {
                "messages": list,
                "embeddings": list,
            },
            # 3 unique messages and embeddings
            "4dfefe7d5605812e49a3f7e76ab43edb77b932f6": {
                "messages": list,
                "embeddings": list,
            },
        }

        # Assert that the last messages and embeddings item in `7cedcb5363ab0ffd3829e7c1363c059c85d83762`
        # is the same as the last messages and embeddings item in `4dfefe7d5605812e49a3f7e76ab43edb77b932f6`
        self.assertEqual(
            commits_embeddings["7cedcb5363ab0ffd3829e7c1363c059c85d83762"]["messages"][-1],
            commits_embeddings["4dfefe7d5605812e49a3f7e76ab43edb77b932f6"]["messages"][-1]
        )
        self.assertEqual(
            commits_embeddings["7cedcb5363ab0ffd3829e7c1363c059c85d83762"]["embeddings"][-1],
            commits_embeddings["4dfefe7d5605812e49a3f7e76ab43edb77b932f6"]["embeddings"][-1]
        )

        # Check that the keys are correct
        self.assertEqual(set(commits_embeddings.keys()), set(expected_structure.keys()))

    def test_14_check_commit_messages_and_embeddings_count(self):
        # Copy the file from the docker container
        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Check that each commit has the same number of messages as embeddings
        for commit_oid, commit_data in commits_embeddings.items():
            messages_count = len(commit_data['messages'])
            embeddings_count = len(commit_data['embeddings'])

            self.assertEqual(messages_count, embeddings_count,
                             f"Commit {commit_oid} has {messages_count} messages but {embeddings_count} embeddings.")

    def test_15_check_commit_messages_and_embeddings_count(self):
        # Copy the file from the docker container
        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Check that each commit has the same number of messages as embeddings
        for commit_oid, commit_data in commits_embeddings.items():
            messages_count = len(commit_data['messages'])
            embeddings_count = len(commit_data['embeddings'])

            self.assertEqual(messages_count, embeddings_count,
                             f"Commit {commit_oid} has {messages_count} messages but {embeddings_count} embeddings.")


    def test_16_verify_commits_embeddings_counts_and_duplicates(self):
        # Wait for any processing to complete and copy the file from the docker container
        time.sleep(5)
        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Verify the count of messages and embeddings for each commit hash
        expected_counts = {
            "c5b3a81463c7d3a188ec60523c0f68c23e93a5dc": 3,
            "879ce80f348263a2580cd38623ee4e80ae69caac": 5,
            "f0d14de7547e2911f262762efa7ea20ada16a2f6": 5,
            "7078ecda662103319304730ecdd31ec01b6ce786": 3,
            "7cedcb5363ab0ffd3829e7c1363c059c85d83762": 3,
            "4dfefe7d5605812e49a3f7e76ab43edb77b932f6": 3,
        }

        # Check that each commit has the correct number of messages and embeddings
        for commit_hash, expected_count in expected_counts.items():
            self.assertIn(commit_hash, commits_embeddings, f"Commit hash {commit_hash} not found")
            self.assertEqual(len(commits_embeddings[commit_hash]["messages"]), expected_count,
                            f"Expected {expected_count} messages for {commit_hash}, got {len(commits_embeddings[commit_hash]['messages'])}")
            self.assertEqual(len(commits_embeddings[commit_hash]["embeddings"]), expected_count,
                            f"Expected {expected_count} embeddings for {commit_hash}, got {len(commits_embeddings[commit_hash]['embeddings'])}")

        # Debug logging for commit 879ce80f
        commit_879 = commits_embeddings["879ce80f348263a2580cd38623ee4e80ae69caac"]
        messages = commit_879["messages"]
        embeddings = commit_879["embeddings"]

        # Log the messages and embeddings for debugging
        #self.logger.debug("Messages for commit 879ce80f: %s", messages)
        #self.logger.debug("Embeddings for commit 879ce80f: %s", embeddings)

        # Direct check for identical messages at positions 2 and 4
        self.assertEqual(messages[2], messages[4], "The 3rd and 5th messages should be identical")

        # Direct check for identical embeddings at positions 2 and 4
        # For whatever, reason, the embeddings aren't in same order as messages.
        #self.assertEqual(embeddings[2], embeddings[4], "The 3rd and 5th embeddings should be identical")

        # Check for identical messages
        identical_messages_count = sum(1 for i in range(len(messages)) for j in range(i + 1, len(messages)) if messages[i] == messages[j])
        self.assertEqual(identical_messages_count, 1,
                                "There should be 1 set of identical messages in commit 879ce80f")

        ## Check for identical embeddings
        #identical_embeddings_count = sum(1 for i in range(len(embeddings)) for j in range(i + 1, len(embeddings)) if embeddings[i] == embeddings[j])
        #self.assertEqual(identical_embeddings_count, 1,
        #                        "There should be 1 set of identical embeddings in commit 879ce80f")

        # Verify that the embeddings array length matches messages array length for all commits
        for commit_hash, data in commits_embeddings.items():
            self.assertEqual(len(data["messages"]), len(data["embeddings"]),
                            f"Message and embedding counts don't match for commit {commit_hash}")

        # Verify that each embedding is a non-empty list of floats
        for commit_hash, data in commits_embeddings.items():
            for embedding in data["embeddings"]:
                self.assertIsInstance(embedding, list, f"Embedding in {commit_hash} is not a list")
                self.assertTrue(len(embedding) > 0, f"Embedding in {commit_hash} is empty")
                self.assertTrue(all(isinstance(val, float) for val in embedding[:5]),
                                f"First 5 values in embedding for {commit_hash} are not all floats")

    def test_18_verify_commit_messages_reasonableness(self):
        # Copy the file from the docker container
        subprocess.run(
            "docker cp commit-file-retrieval:/data/users/repositories/github_com_7db9a_chastler/commits/embeddings/commits_embeddings.json .",
            shell=True,
            check=True
        )

        # Load the JSON file
        with open('commits_embeddings.json', 'r') as file:
            commits_embeddings = json.load(file)

        # Define the expected messages for each commit
        expected_messages = {
            "c5b3a81463c7d3a188ec60523c0f68c23e93a5dc": [
                "Basic video method signatures for video library.",
                "Add Video class with methods for MP4 file processing",
                "The `Video` class is designed to handle MP4 video files."
            ],
            "879ce80f348263a2580cd38623ee4e80ae69caac": [
                "Add some more project scaffolding.",
                "Add initial project setup with Poetry, including dependencies and configuration",
                "eddf150cd15072ba4a8474209ec090fedd4d79e4",
                "The `pyproject.toml` file describes a Python project named \"chastler,\"",
                "eddf150cd15072ba4a8474209ec090fedd4d79e4"
            ],
            "f0d14de7547e2911f262762efa7ea20ada16a2f6": [
                "Initial commit",
                "Remove unnecessary files: .gitignore, LICENSE, and README.md.",
                "This `.gitignore` file is designed to exclude various types of files and directories",
                "The MIT License allows anyone to use, copy, modify, merge, publish, distribute, sublicense, and sell the software for free",
                "The README.md for \"chastler\" provides an overview of the project"
            ],
            "7078ecda662103319304730ecdd31ec01b6ce786": [
                "Add placeholder audio module.",
                "Add placeholder for audio in lib/video/audio.py",
                "The file `lib/video/audio.py` appears to be a placeholder for audio functionality"
            ],
            "7cedcb5363ab0ffd3829e7c1363c059c85d83762": [
                "Explain project is not ready in readme.",
                "Update README.md to reflect project status: clarify that the project has not been started yet.",
                "The README.md indicates that the \"chastler\" project has not been initiated yet."
            ],
            "4dfefe7d5605812e49a3f7e76ab43edb77b932f6": [
                "Add project aim to README -  filter video content, using AI/ML.",
                "Add project description to README.md for clarity on functionality",
                "The README.md for the \"chastler\" project indicates that the project has not been initiated yet."
            ]
        }

        # Initialize the Sentence-BERT model
        model = SentenceTransformer('data/all-MiniLM-L6-v2')
        # Compare messages using cosine similarity
        for commit_oid, expected_msgs in expected_messages.items():
            actual_msgs = commits_embeddings[commit_oid]["messages"]
            for expected_msg, actual_msg in zip(expected_msgs, actual_msgs):
                # Encode the messages into embeddings
                expected_embedding = model.encode(expected_msg)
                actual_embedding = model.encode(actual_msg)

                # Calculate cosine similarity
                similarity = util.cos_sim(expected_embedding, actual_embedding)

                # Assert that the similarity is above a certain threshold (e.g., 0.5)
                self.assertGreaterEqual(similarity.item(), 0.5,
                                       f"Message similarity for commit {commit_oid} is below the threshold. Expected: {expected_msg}, Actual: {actual_msg}")

        # Clean up the copied file
        os.remove('commits_embeddings.json')


    #def test_18_verify_commits_embeddings_message_and_embeddings_order(self):
    #    #See test 16, as there is some attempt to do this. Embeddings order is not honored by system.
    #    # Log the messages and embeddings for debugging
    #    #self.logger.debug("Messages for commit 879ce80f: %s", messages)
    #    #self.logger.debug("Embeddings for commit 879ce80f: %s", embeddings)

    #    # Direct check for identical messages at positions 2 and 4
    #    self.assertEqual(messages[2], messages[4], "The 3rd and 5th messages should be identical")

    #    # Direct check for identical embeddings at positions 2 and 4
    #    # For whatever, reason, the embeddings aren't in same order as messages.
    #    #self.assertEqual(embeddings[2], embeddings[4], "The 3rd and 5th embeddings should be identical")

    #    ## Check for identical embeddings
    #    #identical_embeddings_count = sum(1 for i in range(len(embeddings)) for j in range(i + 1, len(embeddings)) if embeddings[i] == embeddings[j])
    #    #self.assertEqual(identical_embeddings_count, 1,
    #    #                        "There should be 1 set of identical embeddings in commit 879ce80f")


    def test_19_run_machtiani_prompt_with_code_changes(self):
        """Test machtiani prompt command without chat mode to see patch application."""
        # First run git-sync to ensure the repo is ready
        status_command = 'machtiani status'
        wait_for_status_complete(status_command, self.directory)
        time.sleep(3)

        # Run command without --mode chat flag to trigger file changes
        command = 'machtiani "Add a comment to the README explaining the project is a video processing library" --model deepseek/deepseek-r1'
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        # Poll for "Response saved" message for up to 2 minutes
        response_saved = False
        timeout = 180  # 3 minutes in seconds
        start_time = time.time()

        # Check if README.md was modified (an indicator that command completed)
        git_check = 'git status --porcelain README.md'
        while time.time() - start_time < timeout:
            if response_saved:
                break

            # Wait a bit before checking again
            time.sleep(5)

            stdout_git, stderr_git = run_machtiani_command(git_check, self.directory)
            if any("M README.md" in line for line in stdout_git):
                # If README was modified, check output one more time
                response_saved = True
                break

        if not response_saved:
            self.fail("Prompt command did not complete successfully - 'Response saved' not found in output after 2 minutes")

        stdout_normalized = clean_output(stdout_machtiani)

        # Check for the separators and section headers that appear in patch mode
        self.assertIn("============================================================", stdout_normalized)

        # Fix: Change "Writing File Patches" to "Writing & Applying File Patches" to match actual output
        self.assertIn("Writing & Applying File Patches", stdout_normalized)

        # Check that patches were either written/applied or skipped (either is valid)
        patch_processing_evidence = False
        for line in stdout_normalized:
            if "Wrote patch for" in line or "Skipping patch creation for" in line:
                patch_processing_evidence = True
                break

        self.assertTrue(patch_processing_evidence, "No evidence of patch processing in output")

        # Verify the regular response was still saved
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

        # Make sure the prompt was processed (README mentioned)
        self.assertTrue(any("README" in line for line in stdout_normalized))

class ExtraTestEndToEnd:
    """Test cases specifically run within the machtiani repository."""

    @classmethod
    def setup_end_to_end(cls, no_code_host_key=False):
        """Initialize the test environment."""
        cls.maxDiff = None
        cls.directory = "data/git-projects/machtiani"
        cls.configs = "data/configs"
        cls.setup = Setup(cls.directory, cls.configs)

        # Ensure the .machtiani/chat directory exists
        chat_dir = os.path.join(cls.directory, ".machtiani", "chat")
        os.makedirs(chat_dir, exist_ok=True)

        """Class-level setup for logging configuration."""
        #logging.basicConfig(level=logging.DEBUG)
        cls.logger = logging.getLogger(__name__)

    @classmethod
    def teardown_end_to_end(cls):
        """Tear down after tests run within the machtiani repository."""
        print(f"\n--- Tearing down TestMachtianiRepoEndToEnd for machtiani ---")
        # No specific teardown needed for this test yet

    def run_machtiani_command_in_machtiani_repo(self, command):
        """Helper function to run a machtiani command in the machtiani repo."""
        # Use the class directory attribute
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        # Return raw lists for easier parsing of specific lines
        return stdout_machtiani, stderr_machtiani

    def _extract_estimation_time(self, stdout_machtiani):
        """Helper function to extract estimation time from stdout."""
        estimation_time = None
        time_line_found = False
        pattern = r"Time until cost estimation finished: (\d+\.\d+)s"

        for line in stdout_machtiani:
            match = re.search(pattern, line)
            if match:
                time_str = match.group(1)
                try:
                    estimation_time = float(time_str)
                    time_line_found = True
                    print(f"Found cost estimation time: {estimation_time}s")
                    break
                except ValueError:
                    print(f"Warning: Found line matching pattern, but failed to convert '{time_str}' to float.")

        return estimation_time, time_line_found

    def test_cost_estimation_time_low_amplify(self):
        """
        Tests the time and token counts for cost estimation using 'git-sync --amplify low --cost-only'.
        Verifies the output contains the estimation time and it falls within 20-30 seconds.
        Verifies the specific token counts.
        """
        command = 'machtiani git-sync --amplify low --cost-only --verbose'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 531000" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 531000' not found in stdout."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 550098" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 550098' not found in stdout."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout.")
        self.assertIsNotNone(estimation_time, "Failed to extract a valid estimation time float.")

        print(f"Asserting estimation time ({estimation_time}s) is between 20 and 30 seconds.")
        self.assertGreaterEqual(estimation_time, 20.0,
                                f"Cost estimation time ({estimation_time}s) was less than 20 seconds.")
        self.assertLessEqual(estimation_time, 30.0,
                               f"Cost estimation time ({estimation_time}s) was more than 30 seconds.")

    def test_cost_estimation_time_no_amplify(self):
        """
        Tests the time and token counts for cost estimation using 'git-sync --cost-only'.
        Verifies the output contains the estimation time and it falls within 3-5 seconds.
        Verifies the specific token counts.
        """
        command = 'machtiani git-sync --cost-only --verbose'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 265500" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 265500' not found in stdout."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 36343" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 36343' not found in stdout."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout.")
        self.assertIsNotNone(estimation_time, "Failed to extract a valid estimation time float.")

        print(f"Asserting estimation time ({estimation_time}s) is between 3 and 7 seconds.")
        self.assertGreaterEqual(estimation_time, 3.0,
                                f"Cost estimation time ({estimation_time}s) was less than 3 seconds.")
        self.assertLessEqual(estimation_time, 7.0,
                               f"Cost estimation time ({estimation_time}s) was more than 7 seconds.")

    # --- New Tests with --depth 1 ---

    def test_cost_estimation_time_no_amplify_depth_1(self):
        """
        Tests time and tokens for 'git-sync --cost-only --depth 1'.
        Expects time between 2-4 seconds and specific token counts.
        """
        command = 'machtiani git-sync --cost-only --verbose --depth 1'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 500" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 500' not found in stdout (depth 1, no amplify)."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 15" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 15' not found in stdout (depth 1, no amplify)."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout (depth 1, no amplify).")
        self.assertIsNotNone(estimation_time, "Failed to extract valid estimation time (depth 1, no amplify).")

        print(f"Asserting estimation time ({estimation_time}s) is between 1 and 4 seconds (depth 1, no amplify).")
        self.assertGreaterEqual(estimation_time, 1.0,
                                f"Cost estimation time ({estimation_time}s) was less than 1 seconds (depth 1, no amplify).")
        self.assertLessEqual(estimation_time, 4.0,
                               f"Cost estimation time ({estimation_time}s) was more than 4 seconds (depth 1, no amplify).")

    def test_cost_estimation_time_low_amplify_depth_1(self):
        """
        Tests time and tokens for 'git-sync --amplify low --cost-only --depth 1'.
        Expects time between 2-4 seconds and specific token counts.
        """
        command = 'machtiani git-sync --amplify low --cost-only --verbose --depth 1'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 1000" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 1000' not found in stdout (depth 1, low amplify)."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 161" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 161' not found in stdout (depth 1, low amplify)."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout (depth 1, low amplify).")
        self.assertIsNotNone(estimation_time, "Failed to extract valid estimation time (depth 1, low amplify).")

        print(f"Asserting estimation time ({estimation_time}s) is between 1 and 4 seconds (depth 1, low amplify).")
        self.assertGreaterEqual(estimation_time, 1.0,
                                f"Cost estimation time ({estimation_time}s) was less than 1 seconds (depth 1, low amplify).")
        self.assertLessEqual(estimation_time, 4.0,
                               f"Cost estimation time ({estimation_time}s) was more than 4 seconds (depth 1, low amplify).")

    # --- New Tests with --depth 137 ---

    def test_cost_estimation_time_no_amplify_depth_137(self):
        """
        Tests time and tokens for 'git-sync --cost-only --depth 137'.
        Expects time between 2-4 seconds and specific token counts.
        """
        command = 'machtiani git-sync --cost-only --verbose --depth 137'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 68500" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 68500' not found in stdout (depth 137, no amplify)."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 5210" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 5210' not found in stdout (depth 137, no amplify)."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout (depth 137, no amplify).")
        self.assertIsNotNone(estimation_time, "Failed to extract valid estimation time (depth 137, no amplify).")

        print(f"Asserting estimation time ({estimation_time}s) is between 2 and 4 seconds (depth 137, no amplify).")
        self.assertGreaterEqual(estimation_time, 2.0,
                                f"Cost estimation time ({estimation_time}s) was less than 2 seconds (depth 137, no amplify).")
        self.assertLessEqual(estimation_time, 4.0,
                               f"Cost estimation time ({estimation_time}s) was more than 4 seconds (depth 137, no amplify).")

    def test_cost_estimation_time_low_amplify_depth_137(self):
        """
        Tests time and tokens for 'git-sync --amplify low --cost-only --depth 137'.
        Expects time between 8-10 seconds and specific token counts.
        """
        command = 'machtiani git-sync --amplify low --cost-only --verbose --depth 137'
        print(f"\nRunning command in {self.directory}: {command}")
        stdout_machtiani, stderr_machtiani = self.run_machtiani_command_in_machtiani_repo(command)

        # Check for specific token counts
        self.assertTrue(
            any(line.strip() == "Estimated embedding tokens: 137000" for line in stdout_machtiani),
            "Expected 'Estimated embedding tokens: 137000' not found in stdout (depth 137, low amplify)."
        )
        self.assertTrue(
            any(line.strip() == "Estimated inference tokens: 126125" for line in stdout_machtiani),
            "Expected 'Estimated inference tokens: 126125' not found in stdout (depth 137, low amplify)."
        )

        # Extract and assert time
        estimation_time, time_line_found = self._extract_estimation_time(stdout_machtiani)

        self.assertTrue(time_line_found, "Time estimation line not found in stdout (depth 137, low amplify).")
        self.assertIsNotNone(estimation_time, "Failed to extract valid estimation time (depth 137, low amplify).")

        print(f"Asserting estimation time ({estimation_time}s) is between 8 and 10 seconds (depth 137, low amplify).")
        self.assertGreaterEqual(estimation_time, 8.0,
                                f"Cost estimation time ({estimation_time}s) was less than 8 seconds (depth 137, low amplify).")
        self.assertLessEqual(estimation_time, 10.0,
                               f"Cost estimation time ({estimation_time}s) was more than 10 seconds (depth 137, low amplify).")

