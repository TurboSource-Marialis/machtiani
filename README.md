# machtiani

**Machtiani** is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories. It allows users to interact with their codebases by asking questions and retrieving relevant information from files in the project, utilizing language models for processing and generating responses. The aim is to support other models aside from OpenAI, including open-source and self-hosted options.

It's very usable, but rough around the edges at the moment.

## How it works

Machtiani employs a clever document retrieval algorithm that reduces the total code file search to improve efficiency:

1. Find files related to indexed commit messages that are similar to the user prompt.
2. Find files related to indexed file summaries that are similar to the user prompt.
3. Eliminate unrelated files to the prompt via inference.

This method has successfully yielded accurate and concise answers for an open-source project with over 1400 files checked into version control systems (VCS). However, users may experience long wait times, particularly when executing the `git-store` command. There have been instances where the match strength needed to be increased due to the prolonged processing time for all files.

## Limitations

- The current implementation does not accurately account for input token usage, primarily due to the recent additions in steps 2 and 3 above.
- The application may appear to hang if it needs to process hundreds of files.

To fully utilize Machtiani for effective document retrieval, it is essential to have concise, informative, and atomic git commit messages. If your commit messages do not meet this criterion, we recommend using the CLI tool [aicommit](https://github.com/coder/aicommit), which is designed to assist in generating appropriate commit messages.

Machtiani currently relies on OpenAI's gpt-4o-mini (optionally gpt-4o) API, which incurs costs. There is a cost estimator in the works, but users should be aware that for projects with several hundred commits to be indexed and a large number of retrieved files may incur higher OpenAI usage costs.

It is important to note that Machtiani may not effectively handle repositories with a large number of commits. This could potentially limit access to the full history of a repository.

Additionally, while Machtiani aims to improve the relevance of retrieved files, there may still be instances where unrelated files are returned, requiring further refinement in the dynamic match-strength algorithm.

## Upcoming Features to Look Forward To

- [ ] Optional support for 'libre' hosted version.
- [ ] Support open-source LLMs and other models (self-config).
- [x] Add as a submodule `aicommit` to help generate better commit messages, and even rewrite commits to Machtiani standards.
- [x] Cost management to assess indexing costs.
- [ ] Improve handling of file path changes in version control systems.
- [x] Auto-save results in `.machtiani/chat/` with context-aware naming.
- [ ] Enhance the user interface and experience.

## Quick Launch

1. Clone this project.

   ```bash
   git clone --recurse-submodules <repo-url>.git machtiani
   ```

2. Place a `.machtiani-config.yml` file in the project root directory you're chatting with.

   ```yaml
   environment:
     MODEL_API_KEY: "your_openai_api_key"
     MACHTIANI_URL: "http://localhost:5071"  # or your desired API URL
     MACHTIANI_REPO_MANAGER_URL: "http://localhost:5070"
     API_GATEWAY_HOST_KEY: "x-api-gateway-host"      # Header key for API gateway host (e.g. "x-rapidapi-host")
     API_GATEWAY_HOST_VALUE: "your-api-gateway-value" # Header value for API gateway host (e.g. "rapidapi-api-key")
     CONTENT_TYPE_KEY: "Content-Type"                  # Header key for Content-Type
     CONTENT_TYPE_VALUE: "application/json"             # Header value for Content-Type
   ```

   Or place it in ~/.machtiani-config.yml aka $HOME, but any `.machtiani-config.yml` placed in your git project directory will override the one in $HOME.

   **Warning:** If the `OPENAI_API_KEY` is set, please be aware that costs will be incurred for embedding the prompt using the OpenAI API.

   ***Also, you'll have to add a .env file in `machtiani-commit-file-retrieval/`. See [Running the FastAPI Application](machtiani-commit-file-retrieval/README.md#running-the-fastapi-application).***

3. Launch the application in development (without the API Gateway):

   ```bash
   docker-compose up --build --remove-orphans
   ```

4. If you want to run the application in production (with the API Gateway), use the following command:

   ```bash
   docker-compose -f docker-compose.yml -f docker-compose.prod.yml up --build --remove-orphans
   ```

5. Build the Machtiani CLI in `machtiani/machtiani/`.

   ```bash
   go build -o machtiani ./cmd/machtiani
   ```

6. Copy the CLI to a path that works for you in `machtiani/machtiani/`.

   ```bash
   cp machtiani ~/.local/bin/
   ```

7. Build the `aicommit` binary in `machtiani/aicommit/`.

   ```bash
   cd aicommit
   go mod tidy
   go build -o machtiani-aicommit-binary ./cmd/aicommit
   ```

8. Move the binary to a directory in your PATH.

   ```bash
   mv machtiani-aicommit-binary ~/.local/bin/
   ```

9. Start the local web server in a new terminal in `machtiani/machtiani-commit-file-retrieval/`.

   ```bash
   poetry install
   poetry run python web/server.py
   ```

## Go to local homepage at localhost:5072

![Home Page](images/web-home.png)

## Add a repo

1. Add a repo:
   - ![Adding a Repository](images/add-repo-filled.png)

2. Get the latest changes:

   Click on `Get Repo Info` on the homepage, and it will prefill the values you chose when adding the repo originally.

   - ![Getting Repo Info](images/fetch-filled.png)

3. Load the updated repository:
   - ![Loading Repo](images/load-filled.png)

## Updating the repo and index

After your project has new commits on GitHub, follow these steps to get updated repository information and load it:

1. Get the latest changes:
   - ![Getting Repo Info](images/fetch-filled.png)

2. Load the updated repository:
   - ![Loading Repo](images/load-filled.png)

## Go CLI Usage

### Overview

The `machtiani` CLI allows you to interact with the project through command-line parameters. You can provide a markdown file or a prompt directly via the command line, along with various options such as the project name, model type, match strength, and mode of operation.

### Command Structure

```bash
machtiani [flags] [prompt]
```

### Flags
- `-file string` (optional): Specify the path to a markdown file. If provided, the content of this file will be used as the prompt.
- `-project string` (optional): Name of the project. If not set, it will be fetched from git.
- `-model string` (optional): Model to use. Options include `gpt-4o` or `gpt-4o-mini`. Default is `gpt-4o-mini`.
- `-match-strength string` (optional): Match strength options are `high`, `mid`, or `low`. Default is `mid`.
- `-mode string` (optional): Search mode, which can be `pure-chat`, `commit`, or `super`. Default is `commit`.

### Example Usage

1. **Providing a direct prompt:**

   ```bash
   machtiani "Add a new endpoint to get stats."
   ```

   - Output Example:
   - ![Direct Prompt Example](images/default-result.png)

2. **Using an existing markdown chat file:**
   ```bash
   machtiani --file .machtiani/chat/add_state_endpoint.md
   ```

   ```
   ...(previous conversion above)...

   # User

   <Your new prompt instructions>
   ```

   - Output Example:
   - ![Markdown Chat Example](images/editing-markdown-response.png)

3. **Specifying additional parameters:**

   Just a sample of the options.

   ```bash
   machtiani "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high
   ```

4. **Chat mode**

   ```bash
   machtiani --file .machtiani/chat/add_state_endpoint.md --mode pure-chat
   ```

   *Note: This won't retrieve any files with this flag.*

### Different modes

In the last example, you don't have to select `pure-chat` to have a conversation with a markdown file.

You could run the command as

```bash
machtiani --file .machtiani/chat/add_state_endpoint.md
```

without the `--mode pure-chat`.

If you don't select `--mode`, it's the same as `--mode commit`, where it searches commits for possible files to help answer the prompt.

### Store and sync repos

The `git-store` command allows you to add a repository to the Machtiani system.

**Usage:**
```bash
machtiani git-store --branch <default branch> --remote <remote name>
```

- `--remote`: Optionally pass a remote name; otherwise, it will choose 'origin' from the git project directory you run this in.
- `--branch`: Mandatory branch name of the default branch.

**Example:**
```bash
machtiani git-store --branch master
```

#### `git-sync`

The `git-sync` command is used to fetch and checkout a specific branch of the repository.

**Usage:**
```bash
machtiani git-sync --branch <default branch> --remote <remote name>
```

- `--remote`: Optionally pass a remote name; otherwise, it will choose 'origin' from the git project directory you run this in.
- `--branch`: Mandatory branch name of the default branch.

**Example:**
```bash
machtiani git-sync --branch main
```

### Ignoring Files with `.machtiani.ignore`

You can ignore any binary files by providing the full path, such as images, etc.

To exclude specific files from being processed by the application, you can create a `.machtiani.ignore` file in the root of your project directory. The files listed in this file will be ignored during the retrieval process.

#### Example `.machtiani.ignore` file:
```
poetry.lock
go.sum
go.mod
```

### Output

The CLI will print the response received and save the output to a temporary markdown file, which will be displayed in the terminal.

## API Usage

After launch, you can access Machtiani's only endpoint [generate-response](http://localhost:5071/docs#/default/generate_response_generate_response_post) for interacting with the application programmatically.

## Conclusion

This web tool simplifies managing Git repositories through a user-friendly interface, utilizing a FastAPI backend for various tasks like loading projects, adding repositories, fetching project information, and checking out branches.

## Todo

- [x] Retrieve file content and add to prompt.
- [x] Improve UI; if the wrong URL and token are given, it will mess up. Consider handling these strictly on the commit-file-retrieval server side.
- [x] Separate command for sending edited markdown (don't wrap # User) (completed with commit 5a69231d4b48b6cd8c1b1e3b54a1b57c3d295a74).
- [x] Return list of files used in response with `--mode commit`.
- [x] Implement `.machtiani.ignore`.
- [x] Improve style and organization of web UI. Add links to fetch, add, and load on the homepage.
- [x] Break up main.go into a well-organized Go project file structure.
- [x] Fix commit-file-retrieval issue with gpt-4o (i.e., `Unprocessable Entity`).
- [x] Improve messaging when no files are retrieved but found.
- [x] Warn CLI user if there are no retrieved files, suggesting to lower match-strength.
- [x] Add as a submodule [aicommit](https://github.com/coder/aicommit).
- [x] Calculate and cap token usage.
- [ ] In commit-file-retrieval, get the most recent path (in case of name change) from git of a file and only use that.
- [ ] Script to rewrite a project's git commit history.
- [x] Auto-save results in `.machtiani/chat/`. Should name the same if passing filename as `--file`.
- [x] Markdown generated chats should automatically save and have an auto-generate context-aware name.
- [x] Content argument for mode flag should be `pure-chat`.
- [ ] Hide excessive stdout behind specific logging mode in commit-file-retrieval.
- [x] Modify CLI to generate embeddings for the original prompt created by the user, then pass it as a parameter to the generate-response endpoint.
- [ ] Chunk retrieval response by order of ranking according to token cap.
- [ ] Implement a strategy to filter code files possibly related to the prompt.
- [ ] Introduce a `--mode super` option.
- [ ] A `--search-web` option.
- [ ] Make the application serviceable:
    - [x] Add Machtiani commands for `add repo`, `fetch and update`, and `load`.
    - [x] Ensure `git-sync` code-host URL is sensitive to `/` (must) at end of the URL.
    - [x] Simplify user process for adding any repo, storing git keys, fetching and updating.
    - [x] Retrieve code host keys for git-store and git-sync commands from `machtiani-config.yml`.
    - [x] Pass all OpenAI or other LLM keys via the `machtiani-config.yml`.
    - [x] Optionally pass GitHub key.
    - [x] Allow user choice to proceed based on approximated input token usage.
    - [x] Retrieve code host URL from remote origin, allowing override of remote name in git repo with `--remote <remote>`.
    - [x] Ensure `git-sync` works for individual repos.
    - [x] Add header with token, unhardcoded with config.
    - [x] Refactor to compile with variable setting flags for `osEnv`.
    - [ ] Server should not block the thread (async).
    - [ ] Round robin several instances when behind api-gateway.
    - [ ] In `commit-file-retrieval`, fetching of summaries once list of files found could be done async and simultaneously.
    - [x] In app/main.py prompt engineer so each just returns a list of files rather than representing the entire json, less irrelevant entries. This should speed things up to get a response from model.
- [x] Retrieve code host key and URLs from `.machtiani.config`.
- [x] Ensure unique repo names are passed for data save.
