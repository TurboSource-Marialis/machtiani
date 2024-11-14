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

class BaseTestEndToEnd(unittest.TestCase):
    @classmethod
    def setUpClass(cls, no_code_host_key=False):
        cls.maxDiff = None
        cls.directory = "data/git-projects/chastler"
        cls.setup = Setup(cls.directory, no_code_host_key)

        # Fetch the latest branches
        cls.setup.fetch_latest_branches()
        cls.setup.force_push("master-backup", "master")
        cls.setup.create_ignore_file()

        branches = cls.setup.get_branches()
        if " feature" not in branches:
            stdout, stderr = cls.setup.checkout_branch("feature")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)
        if " feature2" not in branches:
            stdout, stderr = cls.setup.checkout_branch("feature2")
            print("Checkout Output:", stdout)
            print("Checkout Errors:", stderr)
        stdout, stderr = cls.setup.checkout_branch("master")

    @classmethod
    def tearDownClass(cls):
        try:
            cls.teardown = Teardown(cls.directory)
            cls.teardown.delete_ignore_file()
            stdout, stderr = cls.teardown.run_git_delete()
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

    def run_machtiani_command(self, command):
        """Helper function to run a machtiani command and clean output."""
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)
        return clean_output(stdout_machtiani)
