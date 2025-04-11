import unittest
from test_utils.test_utils import (
    clean_output,
    run_machtiani_command,
    Teardown,
    Setup,
)

class TestMachtianiCommand(unittest.TestCase):

    def setUp(self):
        # Set the directory for the test
        self.directory = "data/git-projects/chastler"
        self.setup = Setup(self.directory)
        # Fetch the latest branches
        fetch_res = self.setup.fetch_latest_branches()
        print("fetch_res")
        print("Fetched branches:", fetch_res)
        self.setup.force_push("master-backup", "master")

        self.setup.run_git_store()

    def test_run_machtiani_command(self):
        command = 'machtiani  "what does the readme say?" --force --mode chat'

        # Perform the command execution
        stdout_machtiani, stderr_machtiani = run_machtiani_command(command, self.directory)

        # Clean the output
        stdout_normalized = clean_output(stdout_machtiani)

        # Check for specific outputs
        # Assert that 'Using remote URL' is in the output
        self.assertTrue(any("Using remote URL" in line for line in stdout_normalized))

        # Assert that there is a line containing 'chastler'
        self.assertTrue(any("chastler" in line for line in stdout_normalized))

        # Assert that there is a line containing 'Response saved to .machtiani/chat/'
        self.assertTrue(any("Response saved to .machtiani/chat/" in line for line in stdout_normalized))

    def tearDown(self):
        """Clean up the test environment by running the git delete command."""
        self.directory = "data/git-projects/chastler"
        self.teardown = Teardown(self.directory)
        self.teardown.force_push("master-backup", "master")
        try:
            stdout, stderr = self.teardown.run_git_delete()
            # Optionally, you can log or print the output of the teardown
            print("Teardown Output:", stdout)
            print("Teardown Errors:", stderr)
        except Exception as e:
            print(f"Error during teardown: {e}")

if __name__ == '__main__':
    unittest.main()
