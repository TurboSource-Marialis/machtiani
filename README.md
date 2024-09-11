# machtiani

Code chat, against retrieved files from commits.

## Quick launch

Clone this project.

Add, fetch, and load a repo to be indexed - see the [commit-file-retrieval readme](machtiani-commit-file-retrieval/README.md).

Launch.

```bash
docker-compose up --build
```

Build machtiani cli

```
go build -o machtiani
```

Copy to path (or path that works for you).

```
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

Each line in the `.machtiani.ignore` file should contain the name of a file or directory that you want to ignore. Ensure that there are no leading or trailing spaces in the file names.

### Environment Variables

Ensure to set the OpenAI API key in your environment:
```bash
export OPENAI_API_KEY="your_openai_api_key"
```

### Output

The CLI will print the response received from the OpenAI API and save the output to a temporary markdown file, which will be displayed in the terminal.

## API Usage

After launch, try machtiani's only endpoint [generate-response](http://localhost:5071/docs#/default/generate_response_generate_response_post).

## Todo

- [x] Retrieve file content and add to prompt.
- [x] Fetch on UI is temperamental; if the wrong URL and token are given, it will mess up. Maybe all that should be done strictly on the commit-file-retrieval server side, the URL and token, just pass the project name.
- [x] Separate command for sending edited markdown (don't wrap # User) (completed with commit 5a69231d4b48b6cd8c1b1e3b54a1b57c3d295a74).
- [x] Return list of files used in response with `--mode commit`
- [x] machtiani.ignore
- [x] Improve style and organization of web UI. Add links to fetch, add, and load on home page.
- [x] Break up main.go into a well-organized go proj file structure.
- [ ] Add as submodule [aicommit](https://chatgpt.com/share/7f3871ea-b125-41fc-8fdc-2d817e70030d).
- [ ] Calculate and cap token usage.
- [ ] In commit-file-retrieval, get the most recent path (in case of name change) from git of a file and only use that.
- [ ] Script to rewrite a projects git commit history.
- [ ] Auto-save results in `.machtiani/chat/`. Should name the same if passing filename as --markdown.
- [ ] Auto-generate an appropriate file name for generated markdown.

