import subprocess
import unittest

class TestMachtiainiGitStore(unittest.TestCase):
    def test_git_store(self):
        # Initialize the Popen command
        process = subprocess.Popen(
            ['machtiani', 'git-store', '--branch-name', 'master', '--force'],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,  # Line-buffered
            universal_newlines=True
        )
        
        # Capture the output in real time
        try:
            while True:
                output = process.stdout.readline()
                if output == '' and process.poll() is not None:
                    break
                if output:
                    print(output.strip())  # Print or store the output as needed
            
            # Capture any errors
            stderr_output = process.stderr.read()
            if stderr_output:
                print("Error output:", stderr_output.strip())

            # Check for process exit code
            exit_code = process.wait()
            self.assertEqual(exit_code, 0, f"Process exited with code {exit_code}")

        except Exception as e:
            self.fail(f"An exception occurred: {e}")

if __name__ == '__main__':
    unittest.main()
