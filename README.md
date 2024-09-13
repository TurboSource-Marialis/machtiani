# machtiani

**Machtiani** is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories. It allows users to interact with their codebases by asking questions and retrieving relevant information from files in the project, utilizing OpenAI's language models for processing and generating responses. The aim is to support other models, including open-source and self-hosted options.

It's very usable, but rough around the edges at the moment.

## Purpose

The main goal of Machtiani is to enable developers and teams to quickly access information from their code repositories using natural language queries. By leveraging the capabilities of 'off-the-shelf' LLMs, users can generate insights, get code suggestions, and enhance their understanding of the codebase without manually sifting through numerous files.

## Limitations

To fully utilize Machtiani for effective document retrieval, it is essential to have concise, informative, and atomic git commit messages. If your commit messages do not meet this criterion, we recommend using the CLI tool `aicommit`, which is designed to assist in generating appropriate commit messages.

While I personally find Machtiani to be my go-to tool—over ChatGPT or any other alternatives—I primarily use it with codebases similar in size to this project.

Keep in mind that using OpenAI's API incurs costs, and there is currently no cost estimator available. However, for a few hundred commits, users should find that the indexing of commit messages with OpenAI embeddings remains manageable.

## Upcoming Features to Look Forward To

- [ ] Add as a submodule `aicommit` to help generate better commit messages, and even rewrite commits to Machtiani standards.
- [ ] Cost management to how much it will cost to index.
- [ ] Improve handling of file path changes in version control systems.
- [ ] Auto-save results in `.machtiani/chat/` with context-aware naming.
- [ ] Enhance the user interface and experience.

## Quick Launch

1. Clone this project.

   ```bash
   git clone --recurse-submodules <repo-url>.git machtiani
   ```

2. Launch the application in `machtiani/machtiani/`.

   ```bash
   docker-compose up --build --remove-orphans
   ```

3. Build the Machtiani CLI in `machtiani/machtiani/`.
   
   ```bash
    go build -o machtiani ./cmd/machtiani
   ```

4. Copy the CLI to a path that works for you in `machtiani/machtiani/`

   ```bash
   cp machtiani ~/.local/bin/
   ```

5. Start the local web server in a new terminal in `machtiani/machtiani-commit-file-retrieval/`

   ```bash
   poetry shell
   poetry run python web/server.py
   ```

## Add a repo

1. Add a repo:
   - ![Adding a Repository](images/add_repo.png)

2. Fetch the latest changes:
   - ![Fetching Repo Info](images/fetch_repo_info.png)

2. Load the updated repository:
   - ![Loading Repo](images/load_repo.png)

## Updating the repo and index

After your project has new commits on GitHub, follow these steps to get updated repository information and load it:

1. Fetch the latest changes:
   - ![Fetching Repo Info](images/fetch_repo_info.png)

2. Load the updated repository:
   - ![Loading Repo](images/load_repo.png)

## Go CLI Usage

### Overview

The `machtiani` CLI allows you to interact with the project through command-line parameters. You can provide a markdown file or a prompt directly via the command line, along with various options such as the project name, model type, match strength, and mode of operation.

### Command Structure

```bash
machtiani [flags] [prompt]
```

### Flags
- `-markdown string` (optional): Specify the path to a markdown file. If provided, the content of this file will be used as the prompt.
- `-project string` (optional): Name of the project. If not set, it will be fetched from git.
- `-model string` (optional): Model to use. Options include `gpt-4o` or `gpt-4o-mini`. Default is `gpt-4o-mini`.
- `-match-strength string` (optional): Match strength options are `high`, `mid`, or `low`. Default is `mid`.
- `-mode string` (optional): Search mode, which can be `chat`, `commit`, or `super`. Default is `commit`.

### Example Usage

1. **Using a markdown file:**
   ```bash
   machtiani -markdown path/to/your/file.md
   ```
   - ![Basic Usage Example](images/basic_usage.png)

2. **Providing a direct prompt:**
   ```bash
   machtiani "Add a new endpoint to get stats."
   ```
   - ![Direct Prompt Example](images/direct_prompt.png)

3. **Specifying additional parameters:**
   ```bash
   machtiani -project "your_project_name" -model "gpt-4o" -match-strength "high" -mode "commit" "Add a new endpoint to get stats."
   ```
   - ![Advanced Usage Example](images/advanced_usage.png)

4. **Chat mode**
   ```bash
   machtiani --markdown old_result.md --mode chat
   ```

### Ignoring Files with `.machtiani.ignore`

To exclude specific files from being processed by the application, you can create a `.machtiani.ignore` file in the root of your project directory. The files listed in this file will be ignored during the retrieval process.

#### Example `.machtiani.ignore` file:
```
poetry.lock
go.sum
go.mod
```

### Environment Variables

Ensure to set the OpenAI API key in your environment:
```bash
export OPENAI_API_KEY="your_openai_api_key"
```

### Output

The CLI will print the response received from the OpenAI API and save the output to a temporary markdown file, which will be displayed in the terminal.

## API Usage

After launch, you can access Machtiani's only endpoint [generate-response](http://localhost:5071/docs#/default/generate_response_generate_response_post) for interacting with the application programmatically.

## Conclusion

This web tool simplifies managing Git repositories through a user-friendly interface, utilizing a FastAPI backend for various tasks like loading projects, adding repositories, fetching project information, and checking out branches.

## Todo

- [x] Retrieve file content and add to prompt.
- [x] Fetch on UI is temperamental; if the wrong URL and token are given, it will mess up. Maybe all that should be done strictly on the commit-file-retrieval server side, the URL and token just pass the project name.
- [x] Separate command for sending edited markdown (don't wrap # User) (completed with commit 5a69231d4b48b6cd8c1b1e3b54a1b57c3d295a74).
- [x] Return list of files used in response with `--mode commit`.
- [x] machtiani.ignore.
- [x] Improve style and organization of web UI. Add links to fetch, add, and load on home page.
- [x] Break up main.go into a well-organized Go project file structure.
- [x] commit-file-retrieval can't handle gpt-4o (i.e., `Unprocessable Entity`).
- [x] commit-file-retrieval doesn't say there are no files to retrieve if it's found, but doesn't exist in the file.
- [x] CLI user should be warned if there are no retrieved files, with a suggestion to lower match-strength.
- [ ] Add as submodule [aicommit](https://chatgpt.com/share/7f3871ea-b125-41fc-8fdc-2d817e70030d).
- [x] Calculate and cap token usage.
- [ ] In commit-file-retrieval, get the most recent path (in case of name change) from git of a file and only use that.
- [ ] Script to rewrite a project's git commit history.
- [ ] Auto-save results in `.machtiani/chat/`. Should name the same if passing filename as --markdown.
- [ ] Markdown generated chats should automatically save and have an auto-generate context-aware name.
- [ ] Content argument for mode flag should be `chat`
- [ ] Hide excessive stdout behind specific logging mode in commit-file-retrieval
