import os

# List of auto-generated directory names to ignore
IGNORE_DIRS = [
    'scripts',
    'images',
    'build',
    'dist',
    'out',
    'target',
    'bin',
    'obj',
    '.next',
    '.nuxt',
    '.cache',
    'coverage',
    'test_output',
    'tests',
    '__pycache__',
    '.pytest_cache',
    '.tox',
    '.gradle',
    '.idea',
    '.vscode',
    '.vs',
    '.npm',
    '.yarn',
    '.venv',
]

# List of auto-generated file types to look for
AUTO_GENERATED_FILES = [
    'poetry.lock',
    'package-lock.json',
    'yarn.lock',
    'Pipfile.lock',
    'Gemfile.lock',
    'npm-debug.log',
    'yarn-error.log',
    '.DS_Store',
    'Thumbs.db',
    # You can add more auto-generated files here if needed
]

# Function to check if a file is a binary file
def is_binary_file(file_path):
    try:
        with open(file_path, 'rb') as f:
            # Read the first few bytes to determine if it's binary
            chunk = f.read(1024)
            return b'\0' in chunk  # Presence of null byte indicates binary
    except Exception:
        return True  # If there is an error, we assume it's binary

# Function to recursively find binary and auto-generated files
def find_files(root_directory, output_file):
    total_files_found = 0  # Initialize counter
    with open(output_file, 'w') as outfile:
        for dirpath, dirnames, filenames in os.walk(root_directory):
            # Ignore specific directories
            dirnames[:] = [d for d in dirnames if d not in IGNORE_DIRS]
            
            for filename in filenames:
                file_path = os.path.join(dirpath, filename)
                # Check if the file is binary or auto-generated
                if is_binary_file(file_path) or filename in AUTO_GENERATED_FILES:
                    # Get the relative path to the file
                    relative_path = os.path.relpath(file_path, root_directory)
                    # Write the relative path to the output file
                    outfile.write(f"{relative_path}\n")
                    total_files_found += 1  # Increment counter

    return total_files_found  # Return the total count

if __name__ == "__main__":
    # Specify the root directory to start searching
    root_dir = input("Enter the root directory to scan: ")
    output_txt_file = "found_files.txt"
    
    total_found = find_files(root_dir, output_txt_file)
    print(f"File paths have been written to {output_txt_file}")
    print(f"Total files found: {total_found}")
