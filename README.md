# machtiani

**Machtiani** is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories. It allows users to interact with their codebases by asking questions and retrieving relevant information from files in the project, utilizingl, currently, OpenAI's language models for processing and generating responses. The aim is to support other models, such open-source and self-hosted models.

## Purpose

The main goal of Machtiani is to enable developers and teams to quickly access information from their code repositories using natural language queries. By leveraging the capabilities of 'off-the-shelf' LLMs, users can generate insights, get code suggestions, and enhance their understanding of the codebase without manually sifting through numerous files.

## Quick Launch

1. Clone this project.
2. Add, fetch, and load a repository to be indexed—see the [commit-file-retrieval readme](machtiani-commit-file-retrieval/README.md).
3. Launch the application.

```bash
docker-compose up --build
```

4. Build the Machtiani CLI.

```bash
go build -o machtiani
```

5. Copy the CLI to a path that works for you.

```bash
cp machtiani ~/.local/bin/
```

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
- `-mode string` (optional): Search mode, which can be `content`, `commit`, or `super`. Default is `commit`.

### Example Usage

1. **Using a markdown file:**
   ```bash
   machtiani -markdown path/to/your/file.md
   ```

2. **Providing a direct prompt:**
   ```bash
   machtiani "Add a new endpoint to get stats."
   ```

3. **Specifying additional parameters:**
   ```bash
   machtiani -project "your_project_name" -model "gpt-4o" -match-strength "high" -mode "commit" "Add a new endpoint to get stats."
   ```

4. **Content mode**
   ```bash
   machtiani --markdown old_result.md --mode content
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
- [ ] Handle file path changes in vcs.
     - [x] Get new path, but response acts if original name exists.
     - [ ] Confirm the above with logging.
     - [ ] Need to pass a message somehow of the file name change, so the response can clearly say the file path has changed.
- [ ] Script to rewrite a project's git commit history.
- [ ] Auto-save results in `.machtiani/chat/`. Should name the same if passing filename as --markdown.
- [ ] Markdown generated chats should automatically save and have an auto-generate context-aware name.
- [ ] Content argument for mode flag should be `chat`
- [ ] Hide excessive stdout behind specific logging mode in commit-file-retrieval
- [ ] commit flag should be called --commit-files
---
