
# machtiani

**Machtiani** is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories. It allows users to interact with their codebases by asking questions and retrieving relevant information from files in the project, utilizing language models for processing and generating responses. The aim is to support models aside from OpenAI, including open-source and self-hosted options.

It's very usable but rough around the edges at the moment.

## How it Works

Machtiani employs a high performance document retrieval algorithm that leverages git data. By avoiding file chunking, aligns the user query without a catastrophic loss in context. Furthermore, it amplifies the git history, thereby increasing the probability of finding matches.

1. Find files related to indexed commit messages that are similar to the user prompt.
2. Find files related to indexed file summaries that are similar to the user prompt.
3. Eliminate unrelated files to the prompt via inference.


This method has successfully yielded accurate and concise answers for open-source projects with over 1400 files checked into version control systems (VCS). However, users may experience long wait times, particularly when executing the `git-store` command. There have been instances where the match strength needed to be increased due to the prolonged processing time for all files.

## Limitations

- The current implementation does not accurately account for input token usage, primarily due to recent additions in steps 2 and 3 above.
- Has not been tested on projects with more than 4000 commits and 4000 files.

To fully utilize Machtiani for effective document retrieval, it is essential to have concise, informative, and atomic Git commit messages. If your commit messages do not meet this criterion, we recommend using the CLI tool [aicommit](https://github.com/coder/aicommit), which is designed to assist in generating appropriate commit messages.

Machtiani currently relies on OpenAI's `text-embedding-3-large` for embedding and uses `gpt-4o-mini` for inference by default. Users can optionally choose `gpt-4o` for inference by using the `--model` flag. Note that this API usage incurs costs. There is a cost estimator in the works, but users should be aware that for projects with several hundred commits to be indexed and a large number of retrieved files, this may incur higher OpenAI usage costs.

It is important to note that Machtiani may not effectively handle repositories with a large number of commits, potentially limiting access to the full history of a repository.

Additionally, while Machtiani aims to improve the relevance of retrieved files, there may still be instances where unrelated files are returned, requiring further refinement in the dynamic match-strength algorithm.

## Upcoming Features to Look Forward To

- [ ] Optional support for 'libre' hosted version.
- [ ] Support open-source LLMs and other models (self-config).
- [x] Add `aicommit` as a submodule to help generate better commit messages and even rewrite commits to Machtiani standards.
- [x] Cost management to assess indexing costs.
- [ ] Improve handling of file path changes in version control systems.
- [x] Auto-save results in `.machtiani/chat/` with context-aware naming.
- [ ] Enhance the user interface and experience.

## Quick Launch

1. Clone this project.

   ```bash
   git clone --recurse-submodules <repo-url>.git machtiani
   ```

2. Create a `.machtiani-config.yml` file in the project root directory.

   ```yaml
   environment:
     MODEL_API_KEY: "your_openapi_api_key"
   ```

   If you want to work with private repos you have access to.

   ```
     CODE_HOST_API_KEY: "your_github_key_with_repo_scopes"
   ```

    Alternatively, you can place it in `~/.machtiani-config.yml`, but any `.machtiani-config.yml` placed in your Git project directory will override the one in `$HOME`.

    ***Also, you'll have to add a `.env` file in `machtiani-commit-file-retrieval/`.***


3. Launch the application in development (without the API Gateway):

   ```bash
   docker-compose up --build --remove-orphans
   ```

4. Build the Machtiani CLI in `machtiani/machtiani/`.

   ```bash
    go build -o generate_ldflags generate_ldflags.go
   ./build.sh
   ```

5. Copy the CLI to a path that works for you.

   ```bash
   cp machtiani ~/.local/bin/
   ```

6. Optional: build the `aicommit` binary in `machtiani/aicommit/`.

   ```bash
   cd aicommit
   go mod tidy
   go build -o machtiani-aicommit-binary ./cmd/aicommit
   ```

7. Move the binary to a directory in your PATH.

   ```bash
   mv machtiani-aicommit-binary ~/.local/bin/
   ```

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
   machtiani "Add a new endpoint to get stats."
   ```

2. **Using an existing markdown chat file:**
   ```bash
   machtiani --file .machtiani/chat/add_state_endpoint.md
   ```

3. **Specifying additional parameters:**

   ```bash
   machtiani "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high
   ```

4. **Using the `--force` flag to skip confirmation:**

   ```bash
   machtiani git-store --branch master --force
   ```

### Different Modes

In `commit` mode, it searches commits for possible files to help answer the prompt. In `pure-chat` mode, it does not retrieve any files.

#### `git-store`

The `git-store` command allows you to add a repository to the Machtiani system.

**Usage:**
```bash
machtiani git-store --branch <default_branch> --remote <remote_name> [--force]
```

**Example:**
```bash
machtiani git-store --branch master --force
```

#### `git-sync`

The `git-sync` command is used to fetch and checkout a specific branch of the repository.

**Usage:**
```bash
machtiani git-sync --branch <default_branch> --remote <remote_name> [--force]
```

**Example:**
```bash
machtiani git-sync --branch main --force
```

### `git-delete`

The `git-delete` command allows you to remove a repository from the Machtiani system.

**Usage:**
```bash
machtiani git-delete --remote <remote_name> [--force]
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

The CLI will print the response received and save the output to a temporary markdown file, which will be displayed in the terminal.

## Developer Section

### End-to-End Tests

This project includes several end-to-end tests that validate the functionality of the Machtiani commands, with `test_end_to_end.py` serving as the **defacto test** for the application.

#### Running the Defacto Tests

To run the end-to-end test suite from inside `end-to-end-tests` directory:

```bash
python -m unittest test_end_to_end.py test_end_to_end_no_codehost_api_key.py
```

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


## Conclusion

Machtiani simplifies code retrieval and interaction with repositories through a command-line interface, utilizing language models for effective responses.


