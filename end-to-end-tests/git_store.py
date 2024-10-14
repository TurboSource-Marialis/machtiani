import subprocess
import os

def run_machtiani_command(command, directory):
    # Ensure the specified directory exists
    if not os.path.isdir(directory):
        raise FileNotFoundError(f"The directory {directory} does not exist.")

    # Start the process in the specified directory
    process = subprocess.Popen(
        command, 
        stdout=subprocess.PIPE, 
        stderr=subprocess.PIPE, 
        universal_newlines=True, 
        shell=True, 
        cwd=directory  # Run command in the specified directory
    )

    # Collect stdout and stderr in real-time
    stdout, stderr = [], []
    while True:
        stdout_line = process.stdout.readline()
        stderr_line = process.stderr.readline()

        # Break if both are empty and the process is done
        if not stdout_line and not stderr_line and process.poll() is not None:
            break

        if stdout_line:
            stdout.append(stdout_line.strip())
            print(f"stdout: {stdout_line.strip()}")  # Optional: Print to console
        if stderr_line:
            stderr.append(stderr_line.strip())
            print(f"stderr: {stderr_line.strip()}")  # Optional: Print to console

    process.stdout.close()
    process.stderr.close()

    return stdout, stderr

# Run Machtiani command in the specific directory
machtiani_directory = "data/git-projects/chastler"
command_machtiani = 'machtiani git-store --branch-name "master" --force'  # Replace with the actual Machtiani command
stdout_machtiani, stderr_machtiani = run_machtiani_command(command_machtiani, machtiani_directory)

# Optionally, you can print or further process the captured output
print("\nOutput from Machtiani command:")
print("\n".join(stdout_machtiani))

