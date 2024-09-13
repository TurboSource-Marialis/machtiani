# machtiani

**Machtiani** is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories. It allows users to interact with their codebases by asking questions and retrieving relevant information from files in the project, utilizing language models for processing and generating responses. The aim is to support other models other than OpenAI, including open-source and self-hosted options.

It's very usable, but rough around the edges at the moment.

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

## Go to local homepage localhost:5072

![Home Page](images/web-home.png)

## Add a repo

1. Add a repo:
   - ![Adding a Repository](images/add-repo-filled.png)

2. Get the latest changes:
   - ![Getting Repo Info](images/fetch-filled.png)

2. Load the updated repository:
   - ![Loading Repo](images/load-filled.png)

## Updating the repo and index

After your project has new commits on GitHub, follow these steps to get updated repository information and load it:

1. Get the latest changes:
   - ![Getting Repo Info](images/fetch-filled.png)

2. Load the updated repository:
   - ![Loading Repo](images/load-filled.png)

## Example use

If there are new commits on github for your project, hit `Get Repo Info` and then `Load`.

### A new chat (e.g. refactor, fixes, new feature).

In this example, I'm want to add a feature that allows for the chats to be saved in a consistent place, rather than just in a temporary file.

```
 machtiani "Auto-save results in \`.machtiani/chats/\`. Should name the same if passing filename as --markdown. And also give the full path to files you propose to edit"
```

It responds with the below. You can see at the bottom it says the files it used for the answer. And also where it saved it as a temporary file, which actually we're trying to change with this chat.

```
# User

Auto-save results in \`.machtiani/chat/\`. Should name the same if passing filename as --markdown. And also give the full path to files you propose to edit

# Assistant

To implement the features you described regarding auto-saving results in \`.machtiani/chat/\`, naming files based on the provided markdown filename, and ensuring the full path to the files is correctly managed, you'll need to make modifications primarily in the \`handleAPIResponse\` function in \`machtiani/internal/cli/cli.go\` and potentially in the \`CreateTempMarkdownFile\` function in \`machtiani/internal/utils/utils.go\`.

### Proposed Changes

#### 1. Update \`handleAPIResponse\` in \`machtiani/internal/cli/cli.go\`
You will need to modify the \`handleAPIResponse\` function to save the output in the \`.machtiani/chat/\` directory and use a naming convention based on the provided markdown filename if it exists.

Here's how you can do it:

\`\`\`go
import (
    // Add necessary imports
    "path/filepath"
    "os"
)

// ... existing code ...

func handleAPIResponse(prompt string, apiResponse map[string]interface{}, markdownFlag string) {
    // Check for the "machtiani" key first
    if machtianiMsg, ok := apiResponse["machtiani"].(string); ok {
        log.Printf("Machtiani Message: %s", machtianiMsg)
        return // Exit early since we do not have further responses to handle
    }

    openAIResponse, ok := apiResponse["openai_response"].(string)
    if !ok {
        log.Fatalf("Error: openai_response key missing")
    }

    var retrievedFilePaths []string
    if paths, exists := apiResponse["retrieved_file_paths"].([]interface{}); exists {
        for _, path := range paths {
            if filePath, valid := path.(string); valid {
                retrievedFilePaths = append(retrievedFilePaths, filePath)
            }
        }
    } else {
        log.Fatalf("Error: retrieved_file_paths key missing")
    }

    markdownContent := createMarkdownContent(prompt, openAIResponse, retrievedFilePaths, markdownFlag)
    renderMarkdown(markdownContent)

    // Determine the filename for saving
    var chatFilename string
    if markdownFlag != "" {
        chatFilename = filepath.Base(markdownFlag) // Get the base name of the markdown file
    } else {
        chatFilename = "chat_response.md" // Default filename if no markdown file is provided
    }

    chatDir := ".machtiani/chat/"
    // Ensure the directory exists
    if err := os.MkdirAll(chatDir, os.ModePerm); err != nil {
        log.Fatalf("Error creating chat directory: %v", err)
    }

    chatFilePath := filepath.Join(chatDir, chatFilename)
    if err := ioutil.WriteFile(chatFilePath, []byte(markdownContent), 0644); err != nil {
        log.Fatalf("Error saving chat response to file: %v", err)
    }

    fmt.Printf("Response saved to %s\n", chatFilePath)
}

// ... existing code ...

#### 2. Update \`CreateTempMarkdownFile\` in \`machtiani/internal/utils/utils.go\`
You may want to modify this function to ensure that it saves the temporary markdown file in the appropriate location if desired. However, since you primarily want the chat responses saved in a specific folder, this function can remain as is unless you have other needs.

### Resulting File Paths
- For the chat responses, the files will be saved in:
  \`\`\`
  .machtiani/chat/
  \`\`\`
- The filenames will be either the base name of the provided markdown file or \`chat_response.md\` if no markdown file is provided.

### Summary of Changes
- The main change is in the \`handleAPIResponse\` function where we added logic to save the response in the \`.machtiani/chat/\` directory with context-aware naming based on the input markdown file.
- Ensure that the directory is created if it doesn't exist before saving the file.

You can now follow these changes to implement the requested functionality in your project. If you have further modifications or require additional features, feel free to ask!

# Retrieved File Paths

- README.md
- machtiani/go.mod
- machtiani/internal/cli/cli.go
- machtiani/internal/utils/utils.go
- .machtiani.ignore
- .gitignore
- app/main.py

Response saved to /tmp/response2567089509/response.md
```

## Continue a chat

The first shot was pretty good, but I'm not entirely satisfied as I want the handler to call a utility to save the file.

I simply edit /tmp/response2567089509/response.md a `# User` heading and put my prompt there.

```
...(existing chat)...

- machtiani/internal/utils/utils.go
- .machtiani.ignore
- .gitignore
- app/main.py

Response saved to /tmp/response2567089509/response.md

# User

Refactor the above so that the saving functionality is in utils.go
```

And now I run the modified file

```
machtiani --markdown /tmp/response2567089509/response.md --mode pure-chat
```

Below is the response with the chat continued, where it does what is asked. I omit the majority of the earlier chat for your readability, but it's all there in reality.

```
# User

Auto-save results in \`.machtiani/chat/\`. Should name the same if passing filename as --markdown. And also give the full path to files you propose to edit

# Assistant

To implement the features you described regarding auto-saving results in \`.machtiani/chat/\`, naming files based on the provided markdown filename, and ensuring the full path to the files is correctly managed, you'll need to make modifications primarily in the \`handleAPIResponse\` function in \`machtiani/internal/cli/cli.go\` and potentially in the \`CreateTempMarkdownFile\` function in \`machtiani/internal/utils/utils.go\`.

...(omit for readibility)

# User

Refactor the above so that the saving functionality is in utils.go


# Assistant

To refactor the saving functionality into \`utils.go\`, you can create a new function that handles the creation of the markdown file in the desired directory. This will help keep your code organized and maintain separation of concerns. Here’s how you can do that:

### Proposed Changes

#### 1. Create a New Function in \`machtiani/internal/utils/utils.go\`

Add a new function to handle the saving of the markdown file. This function will take the markdown content and the desired filename as parameters.

\`\`\`go
package utils

import (
    "io/ioutil"
    "os"
    "path/filepath"
    "log"
)

// SaveMarkdownToFile saves the given markdown content to a specified file in the .machtiani/chat/ directory.
func SaveMarkdownToFile(content string, filename string) error {
    chatDir := ".machtiani/chat/"
    // Ensure the directory exists
    if err := os.MkdirAll(chatDir, os.ModePerm); err != nil {
        return err
    }

    chatFilePath := filepath.Join(chatDir, filename)
    if err := ioutil.WriteFile(chatFilePath, []byte(content), 0644); err != nil {
        return err
    }

    log.Printf("Response saved to %s\n", chatFilePath)
    return nil
}
\`\`\`

#### 2. Update \`handleAPIResponse\` in \`machtiani/internal/cli/cli.go\`

Now, modify the \`handleAPIResponse\` function to use the new \`SaveMarkdownToFile\` function instead of handling file saving directly.

Here's the updated \`handleAPIResponse\` function:

\`\`\`go
import (
    // Add necessary imports
    "path/filepath"
    "log"
    "machtiani/internal/utils" // Import the utils package
)

// ... existing code ...

func handleAPIResponse(prompt string, apiResponse map[string]interface{}, markdownFlag string) {
    // Check for the "machtiani" key first
    if machtianiMsg, ok := apiResponse["machtiani"].(string); ok {
        log.Printf("Machtiani Message: %s", machtianiMsg)
        return // Exit early since we do not have further responses to handle
    }

    openAIResponse, ok := apiResponse["openai_response"].(string)
    if !ok {
        log.Fatalf("Error: openai_response key missing")
    }

    var retrievedFilePaths []string
    if paths, exists := apiResponse["retrieved_file_paths"].([]interface{}); exists {
        for _, path := range paths {
            if filePath, valid := path.(string); valid {
                retrievedFilePaths = append(retrievedFilePaths, filePath)
            }
        }
    } else {
        log.Fatalf("Error: retrieved_file_paths key missing")
    }

    markdownContent := createMarkdownContent(prompt, openAIResponse, retrievedFilePaths, markdownFlag)
    renderMarkdown(markdownContent)

    // Determine the filename for saving
    var chatFilename string
    if markdownFlag != "" {
        chatFilename = filepath.Base(markdownFlag) // Get the base name of the markdown file
    } else {
        chatFilename = "chat_response.md" // Default filename if no markdown file is provided
    }

    // Save the markdown content using the utility function
    if err := utils.SaveMarkdownToFile(markdownContent, chatFilename); err != nil {
        log.Fatalf("Error saving chat response to file: %v", err)
    }
}

// ... existing code ...

\`\`\`

Response saved to tmp/response3524948614/response.md
```

### Different modes

In the last example, you can don't have to select `pure-chat` to have a conversation with a markdown file

You could have ran the command as

```
machtiani --markdown /tmp/response2567089509/response.md
```

without the `--mode pure-chat` as originally.

```
machtiani --markdown /tmp/response2567089509/response.md --mode pure-chat
```

If you don't select `--mode`, it's the same as `--mode commit`, where it searches commits for possible files to help answer the prompt.

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
- `-mode string` (optional): Search mode, which can be  `pure-chat`, `commit`, or `super`. Default is `commit`.

### Example Usage

1. **Using a markdown file:**
   ```bash
   machtiani -markdown path/to/your/file.md

   ```
   - ![Basic Usage Example](images/editing-markdown-response.png)

2. **Providing a direct prompt:**
   ```bash
   machtiani "Add a new endpoint to get stats."
   ```
   - ![Direct Prompt Example](images/default-result.png)

3. **Specifying additional parameters:**

   ```bash
   machtiani "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high
   ```

4. **Chat mode**

   ```bash
   machtiani --markdown path/to/your/file.md --mode pure-chat
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

The CLI will print the response received and save the output to a temporary markdown file, which will be displayed in the terminal.

## API Usage

After launch, you can access Machtiani's only endpoint [generate-response](http://localhost:5071/docs#/default/generate_response_generate_response_post) for interacting with the application programmatically.

## Conclusion

This web tool simplifies managing Git repositories through a user-friendly interface, utilizing a FastAPI backend for various tasks like loading projects, adding repositories, fetching project information, and checking out branches.

## Todo

- [x] Retrieve file content and add to prompt.
- [x] Get on UI is temperamental; if the wrong URL and token are given, it will mess up. Maybe all that should be done strictly on the commit-file-retrieval server side, the URL and token just pass the project name.
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
- [x] Content argument for mode flag should be `pure-chat`
- [ ] Hide excessive stdout behind specific logging mode in commit-file-retrieval

