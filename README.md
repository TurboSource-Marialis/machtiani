# machtiani

**Machtiani** is a command-line interface and local service to chat with git repositories, even if it has thousands of files and commits.

- Git strategy is o(n) time, so this will scale to enormous projects.
- Currently, it's fast enough to work on projects that are as large as a few thousand files and commits.
- We aim to optimize syncing and retrieval time by an order of magnitude, reaching for enormous projects.
- It doesn't have an abundance of features (e.g. web search, multiple models), yet!

## How it Works

Let's use machtiani! Doing the following in the `machtiani-commit-file-retrieval` submodule of machtiani.

```
$ mct \
  "Explain how machtiani gets the correct context from files through the git history, \
   and how is this different than common indexing strategies"
```

## Limitations

- Works only in the terminal - we want also a local web app.
- Works only with OpenAI - we want this to work for any model with plugins.
- Works only with Github - we want to expand to all codehosts, even local git servers.
- Input token costs are off by 10-20% - we want it to be exact, not relying on generalized formula.
- The largest project we tried was 4000 commits and 4000 files in git - we want these numbers to grow, rapidly.

These are straightforward to overcome (e.g. support all git servers), doable with some work (e.g. optimize speed), or will get there with some support (e.g. web search).

## Todo

- [ ] Get it to work on any git-line codehost.
- [ ] Increase store and sync speed by an order of magnitude (so even bigger, and bigger projects).
- [ ] Increase response time by an order of magnitue.
- [ ] Web retrieval.


## Quick Launch

1. Clone this project.

   ```bash
   git clone --recurse-submodules <repo-url>.git machtiani
   ```

2. Create a `~/.machtiani-config.yml`.

   ```yaml
   environment:
     MODEL_API_KEY: "your_openapi_api_key"
   ```

   If you want to work with private repos you have access to, add `CODE_HOST_API_KEY`.

   ```yaml
   environment:
     MODEL_API_KEY: "your_openapi_api_key"
     CODE_HOST_API_KEY: "your_github_key"
   ```

   You can override the global config per project by placing a `.machtiani-config.yml` into your git project's root directory.

3. Build the cli and put in path.

   ```bash
   cd machtiani

   go build \
     -ldflags "$(go run generate_ldflags_local.go)" \

   -o mct-cli \
     ./cmd/machtiani

   cp mct-cli /$HOME/.local/bin/mct
   ```


4. Launch the application.

   ```bash
   docker-compose up --build --remove-orphans
   ```

5. Put a project on machtiani.

  ```bash


  mct sync
  ```

  Replace master with main, if that is the default branch.

6. Chat with the project


  ```bash

   mct "Ask whatever you want here"
   ```

7. Sync any new commits you pushed to your remote `origin` on Github.

  ```bash


  mct sync
  ```

  Replace master with main, if that is the default branch.

  If you have local changes in you git that aren't pushed to Github, machtiani won't find changes to sync.

## Go CLI Usage

### Overview

The `machtiani` CLI allows you to interact with the project through command-line parameters. You can provide a markdown file or a prompt directly via the command line, along with various options such as the project name, model type, match strength, and mode of operation.

### Command Structure

```bash
machtiani [flags] [prompt]
```

### Flags

- `-file string` (optional): Specify the path to a markdown file. If provided, the content of this file will be used as the prompt.
- `-project string` (optional): Name of the project. If not set, it will be fetched from Git.
- `-model string` (optional): Model to use. Options include `gpt-4o` or `gpt-4o-mini`. Default is `gpt-4o-mini`.
- `-match-strength string` (optional): Match strength options are `high`, `mid`, or `low`. Default is `mid`.
- `-mode string` (optional): Search mode, which can be `pure-chat`, `commit`, or `super`. Default is `commit`.
- `--force` (optional): Skip confirmation prompt and proceed with the operation.

### Example Usage

1. **Providing a direct prompt:**

   ```bash

   mct "Add a new endpoint to get stats."
   ```

2. **Using an existing markdown chat file:**
   ```bash

   mct --file .machtiani/chat/add_state_endpoint.md
   ```

3. **Specifying additional parameters:**

   ```bash

   mct "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high
   ```

4. **Using the `--force` flag to skip confirmation:**

   ```bash

   mct git-store --force
   ```

### Different Modes

In `commit` mode, it searches commits for possible files to help answer the prompt. In `pure-chat` mode, it does not retrieve any files.

#### `git-store`

The `git-store` command allows you to add a repository to the Machtiani system.

**Usage:**
```bash

mct git-store --remote <remote_name> [--force]
```

**Example:**
```bash
machtiani git-store --force
```

#### `git-sync`

The `git-sync` command is used to fetch and checkout a specific branch of the repository.

**Usage:**
```bash

mct git-sync --remote <remote_name> [--force]
```

**Example:**
```bash
machtiani git-sync --force
```

### `git-delete`

The `git-delete` command allows you to remove a repository from the Machtiani system.

**Usage:**
```bash

mct git-delete --remote <remote_name> [--force]
```

**Example:**
```bash
machtiani git-delete --remote origin --force
```

### Ignoring Files with `.machtiani.ignore`

You can ignore any binary files by providing the full path, such as images, etc. To exclude specific files from being processed by the application, you can create a `.machtiani.ignore` file in the root of your project directory. The files listed in this file will be ignored during the retrieval process.

#### Example `.machtiani.ignore` file:
```
poetry.lock
go.sum
go.mod
```

### Output

The CLI will stream the response and save the chat in `.machtiani/chat/` in the directory you ran the prompt. It also gives a descriptive name to the chat file for you convenience

## Features


Machtiani will let you know if you should pull latest changes so you can get the most powerful version available.

It checks for any latest system messages every so often from Machtiani's codehost (currently GitHub) using the git protocol. It simply does a shallow clone of Machtiani's repo, gets the latest system message, and throws away the clone. This ensures the tool doesn't 'phone home' to external services beyond the codehost, maintaining user privacy.

## Developer Section

### End-to-End Tests

This project includes several end-to-end tests that validate the functionality of the Machtiani commands, with `test_end_to_end.py` serving as the **defacto test** for the application.

#### Setup

1. Make sure you have `git lfs` installed.

   You can use your systems package manager or whatever is best for you.

   Make sure to run after doing the above.

   ```bash
   git lfs install
   ```

Make sure to run after doing the above.

`git lfs install`

2. **Create and activate a Python virtual environment in the project root:**

   ```bash
   python3 -m venv venv  # Create a virtual environment named 'venv'
   source venv/bin/activate  # Activate the virtual environment (Linux/macOS)
   ```

3. **Install test dependencies using Poetry:**

   Navigate to the project root dir and run:

   ```bash
   poetry install
   ```

4. Install `all-MiniLM-L6-v2` into `end-to-end-tests/data/`

   ```
   cd end-to-end-tests/data
   git clone https://huggingface.co/sentence-transformers/all-MiniLM-L6-v2
   ```
   It's a distilled Sentence-BERT embedding model. And it will work just fine on the cpus of a laptop.

   This is used to help test whether generated git messages by machtiani are in the ballpark of being correct using a cosine similarity threshold.

5. Clone `chastler` and `machtiani` into `end-to-end-tests/data/git-projects`

   ```
   cd end-to-end-tests/data/git-projects
   git clone https://github.com/7db9a/chastler
   git clone --branch end-to-end-test --single-branch https://github.com/7db9a/machtiani-end-to-end-test
   ```

   Unless you have the codheost keys for the repo, this may not work.

#### Running the Defacto Tests

To run the end-to-end test suite from inside `end-to-end-tests` directory:

```bash
python -m unittest test_end_to_end test_end_to_end_extra
```

`test_end_to_end_no_codehost_api_key` is no longer useful as count tokens requires auto-deletion of the repo on dry-run on initialization.

#### Why `test_end_to_end.py` is the Defacto Test

- **Comprehensive Coverage**: This test suite encompasses multiple critical functionalities of the Machtiani CLI, including the `git-store`, prompt handling, and synchronization with remote repositories. By running this test, you verify the overall integration and behavior of the CLI tool in a realistic scenario.

- **Realistic Environment**: The tests are designed to execute in a realistic environment, mimicking actual user interactions with the CLI. This helps in identifying issues that may not be apparent in isolated unit tests.

- **Validation of Core Features**: As it encompasses key functionalities, running this test ensures that the essential features of Machtiani are working as expected.

This command prioritizes the most critical integration tests, ensuring that your core functionalities more cost effectively (only a single round of setup and teardown).

### Other Tests

In addition to `test_end_to_end.py`, there are other tests available, such a below. However, it is recommended to prioritize the defacto tests above for a more focused validation of the core features. There no guarantee that the other tests will be maintained or its documentation kept up-to-date.

1. **Test for `git-store`**

   - **File**: `end-to-end-tests/test_git_store.py`
   - **Description**: Verifies the `git-store` command functionality.

2. **Test for Prompt Command**

   - **File**: `end-to-end-tests/test_prompt_command.py`
   - **Description**: Validates the behavior of the prompt command when querying specific questions.

To run all tests, you can still use:

```bash
python -m unittest discover .
```

