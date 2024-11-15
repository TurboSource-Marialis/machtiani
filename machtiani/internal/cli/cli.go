package cli

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url" // Import the net/url package
    "os"
    "os/exec"
    "strings"
    "path"
    "context"
    "time"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
    "github.com/charmbracelet/glamour"
    "github.com/sashabaranov/go-openai" // Ensure you import the OpenAI package
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

func float32ToFloat64(input []float32) []float64 {
    output := make([]float64, len(input))
    for i, v := range input {
        output[i] = float64(v)
    }
    return output
}


func generateEmbeddings(apiKey, prompt string) ([]float64, error) {
    client := openai.NewClient(apiKey)
    req := openai.EmbeddingRequest{
        Model: "text-embedding-3-large",
        Input: []string{prompt},
    }
    resp, err := client.CreateEmbeddings(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("failed to create embeddings: %w", err)
    }

    if len(resp.Data) > 0 {
        return float32ToFloat64(resp.Data[0].Embedding), nil
    }
    return nil, fmt.Errorf("no embeddings returned")
}

// Call the /generate-filename endpoint
func generateFilename(context string, apiKey string) (string, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    endpoint := config.Environment.MachtianiURL
    if endpoint == "" {
        return "", fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }


    url := fmt.Sprintf("%s/generate-filename?context=%s&api_key=%s", endpoint, url.QueryEscape(context), url.QueryEscape(apiKey))

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", fmt.Errorf("failed to create request: %v", err)
    }

    // Set API Gateway headers if not blank
    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return "", fmt.Errorf("failed to call generate-filename endpoint: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(resp.Body)
        return "", fmt.Errorf("generate-filename endpoint returned status %d: %s", resp.StatusCode, string(body))
    }

    var filename string
    if err := json.NewDecoder(resp.Body).Decode(&filename); err != nil {
        return "", fmt.Errorf("failed to decode response from generate-filename endpoint: %v", err)
    }

    return filename, nil
}

func Execute() {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
    fileFlag := fs.String("file", "", "Path to the markdown file")
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: pure-chat, commit, or super")
    verboseFlag := fs.Bool("verbose", false, "Enable verbose output.")
    remoteName := fs.String("remote", "origin", "Name of the remote repository")
    branchName := fs.String("branch-name", "", "Branch name")
    forceFlag := fs.Bool("force", false, "Skip confirmation prompt and proceed with the operation.")

    // Use the new remote URL function
    remoteURL, err := utils.GetRemoteURL(remoteName)
    if err != nil {
        log.Fatalf(err.Error())
    }
    fmt.Printf("Using remote URL: %s\n", remoteURL)
    projectName :=  remoteURL

    var apiKey *string = utils.GetAPIKey(config)


    args := os.Args[1:]

    // Check if the user requests help
    if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" {
        printHelp()
        return
    }

    if len(os.Args) >= 2 && os.Args[1] == "status" {
        // Call CheckStatus
        statusResponse, err := api.CheckStatus(remoteURL, apiKey)
        if err != nil {
            log.Fatalf("Error checking status: %v", err)
        }

        // Output the result
        if statusResponse.LockFilePresent {
            fmt.Println("Project is getting processed and not ready for chat.")
            // Convert the float64 seconds to a duration (in nanoseconds)
            duration := time.Duration(statusResponse.LockTimeDuration * float64(time.Second))
            // Format the duration to show hours, minutes, seconds
            fmt.Printf("Lock duration: %02d:%02d:%02d\n", int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
        } else {
            fmt.Println("Project is ready for chat!")
        }
        return // Exit after handling status
    }

    if len(os.Args) >= 2 && os.Args[1] == "git-store" {
        err := utils.ParseFlags(fs, args[1:]) // Parse flags after the command
        if err != nil {
            log.Fatalf("Error parsing flags: %v", err)
        }

        // Call the new function to add the repository
        response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, config.Environment.RepoManagerURL, *forceFlag)
        if err != nil {
            log.Fatalf("Error adding repository: %v", err)
        }

        fmt.Println(response.Message)
        // Print the success message
        fmt.Println("---")
        fmt.Println("Your repo is getting added to machtiani is in progress!")
        fmt.Println("Please check back by running `machtiani status` to see if it completed.")
        return // Exit after handling git-store
    }

    // Check if the command is git-sync
    if len(os.Args) >= 2 && os.Args[1] == "git-sync" {
        err := utils.ParseFlags(fs, args[1:]) // Parse flags after the command
        if err != nil {
            log.Fatalf("Error parsing flags: %v", err)
        }

        if remoteURL == "" || *branchName == "" {
            log.Fatal("Error: all flags --code-url, --branch-name must be provided.")
        }

        // Call the new function to fetch and checkout the branch
        message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, *branchName, apiKey, config.Environment.ModelAPIKey, *forceFlag)
        if err != nil {
            log.Fatalf("Error syncing repository: %v", err)
        }

        // Print the returned message
        fmt.Println(message)
        return
    }

    if len(os.Args) >= 2 && os.Args[1] == "git-delete" {
        err := utils.ParseFlags(fs, args[1:]) // Parse flags after the command
        if err != nil {
            log.Fatalf("Error parsing flags: %v", err)
        }

        if remoteURL == "" {
            log.Fatal("Error: --remote must be provided.")
        }

        ignoreFiles := []string{} // Populate this list as needed
        vcsType := "git" // Set the VCS type as needed
        openaiAPIKey := config.Environment.ModelAPIKey // Adjust as necessary

        // Call the updated DeleteStore function
        response, err := api.DeleteStore(projectName, remoteURL, ignoreFiles, vcsType, apiKey, &openaiAPIKey, config.Environment.RepoManagerURL, *forceFlag)
        if err != nil {
            log.Fatalf("Error deleting store: %v", err)
        }

        fmt.Println(response.Message)
        return // Exit after handling git-delete
    }

    var promptParts []string
    var flagArgs []string

    for i := 0; i < len(args); i++ {
        if strings.HasPrefix(args[i], "-") {
            flagArgs = append(flagArgs, args[i])
            if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
                flagArgs = append(flagArgs, args[i+1])
                i++
            }
        } else {
            promptParts = append(promptParts, args[i])
        }
    }

    if err := fs.Parse(flagArgs); err != nil {
        log.Fatalf("Error parsing flags: %v", err)
    }

    prompt := strings.Join(promptParts, " ")
    if err != nil {
        log.Fatalf("Error getting project name: %v", err)
    }

    utils.ValidateFlags(modelFlag, matchStrengthFlag, modeFlag)

    if *fileFlag != "" {
        content, err := ioutil.ReadFile(*fileFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(content)
    } else if prompt == "" {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    if *verboseFlag {
        printVerboseInfo(*fileFlag, *modelFlag, *matchStrengthFlag, *modeFlag, prompt)
    }


    startTime := time.Now() // Start the timer here
    // Call OpenAI API to generate response
apiResponse, err := api.CallOpenAIAPI(prompt, projectName, *modeFlag, *modelFlag, *matchStrengthFlag, *forceFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    // Check for error in response
    if errorMsg, ok := apiResponse["error"].(string); ok {
        log.Fatalf("Error from API: %s", errorMsg)
    }

    // Determine the filename to save the response
    filename := path.Base(*fileFlag) // Extract filename with extension

    // Strip all extensions from the filename
    for ext := path.Ext(filename); ext != ""; ext = path.Ext(filename) {
        filename = strings.TrimSuffix(filename, ext)
    }

    // Check if the filename is empty or just "."
    if filename == "" || filename == "." {
        // If no markdown file provided, generate a filename
        filename, err = generateFilename(prompt, config.Environment.ModelAPIKey)
        if err != nil {
            log.Fatalf("Error generating filename: %v", err)
        }
    }

    // Handle API response and save it to a markdown file with the generated filename
    handleAPIResponse(prompt, apiResponse, filename, *fileFlag) // Pass filename here
    // End timing after the response is handled
    duration := time.Since(startTime)
    fmt.Printf("Total response handling took %s\n", duration) // Print total duration
}

func handleAPIResponse(prompt string, apiResponse map[string]interface{}, filename string, fileFlag string) {
    // Timing within this function is no longer needed since the timing is handled in Execute

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

    markdownContent := createMarkdownContent(prompt, openAIResponse, retrievedFilePaths, fileFlag)
    renderMarkdown(markdownContent)

    // Save the response to the markdown file with the provided filename
    tempFile, err := utils.CreateTempMarkdownFile(markdownContent, filename) // Pass the filename
    if err != nil {
        log.Fatalf("Error creating markdown file: %v", err)
    }

    fmt.Printf("Response saved to %s\n", tempFile)
}

// handleError prints the error message and exits the program.
func handleError(message string) {
    fmt.Fprintf(os.Stderr, "%s\n", message)
    os.Exit(1)
}

// runAicommit generates a commit message using aicommit and lets it perform the git commit.
func runAicommit(args []string) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    // Define flags specific to aicommit
    fs := flag.NewFlagSet("aicommit", flag.ExitOnError)
    openaiKey := fs.String("openai-key", config.Environment.ModelAPIKey, "OpenAI API Key")
    modelFlag := fs.String("model", "gpt-4o-mini", "Model to use for generating messages")
    amend := fs.Bool("amend", false, "Amend the last commit instead of creating a new one")
    context := fs.String("context", "", "Additional context for generating the commit message")

    // Parse the provided arguments
    err = fs.Parse(args)
    if err != nil {
        handleError(fmt.Sprintf("Error parsing flags: %v", err))
    }

    // Construct aicommit arguments without --dry
    aicommitArgs := []string{
        "--openai-key", *openaiKey,
        "--model", *modelFlag,
    }
    if *amend {
        aicommitArgs = append(aicommitArgs, "--amend")
    }
    if *context != "" {
        aicommitArgs = append(aicommitArgs, "--context", *context)
    }

    // Handle dry-run mode by adding --dry to aicommit arguments if needed
    if utils.IsDryRunEnabled() {
        aicommitArgs = append(aicommitArgs, "--dry")
    }

    // Locate the aicommit binary
    binaryPath, err := exec.LookPath("aicommit")
    if err != nil {
        handleError(fmt.Sprintf("aicommit binary not found in PATH: %v", err))
    }

    // Create the command to run aicommit
    cmd := exec.Command(binaryPath, aicommitArgs...)

    // Set the working directory to the current directory
    cwd, err := os.Getwd()
    if err != nil {
        handleError(fmt.Sprintf("Failed to get current working directory: %v", err))
    }
    cmd.Dir = cwd

    // Inherit environment variables
    cmd.Env = os.Environ()

    // Attach stdout and stderr to display aicommit output directly
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Execute the aicommit command
    err = cmd.Run()
    if err != nil {
        handleError(fmt.Sprintf("Error running aicommit: %v", err))
    }

    // No need to perform git commit manually; aicommit handles it
}

func printVerboseInfo(markdown, model, matchStrength, mode, prompt string) {
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", markdown)
    fmt.Printf("Model: %s\n", model)
    fmt.Printf("Match strength: %s\n", matchStrength)
    fmt.Printf("Mode: %s\n", mode)
    fmt.Printf("Prompt: %s\n", prompt)
}


func createMarkdownContent(prompt, openAIResponse string, retrievedFilePaths []string, fileFlag string) string {
    var markdownContent string
    if fileFlag != "" {
        markdownContent = fmt.Sprintf("%s\n\n# Assistant\n\n%s", readMarkdownFile(fileFlag), openAIResponse)
    } else {
        markdownContent = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
    }

    if len(retrievedFilePaths) > 0 {
        markdownContent += "\n\n# Retrieved File Paths\n\n"
        for _, path := range retrievedFilePaths {
            markdownContent += fmt.Sprintf("- %s\n", path)
        }
    }

    return markdownContent
}

func renderMarkdown(content string) {
    renderer, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(120),
    )
    if err != nil {
        log.Fatalf("Error creating renderer: %v", err)
    }

    out, err := renderer.Render(content)
    if err != nil {
        log.Fatalf("Error rendering Markdown: %v", err)
    }

    fmt.Println(out)
}

func readMarkdownFile(path string) string {
    content, err := ioutil.ReadFile(path)
    if err != nil {
        log.Fatalf("Error reading markdown file: %v", err)
    }
    return string(content)
}

func printHelp() {
    helpText := `Usage: machtiani [flags] [prompt]

    Machtiani is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories.

    Commands:
      git-store                    Add a repository to the Machtiani system.
      git-sync                     Fetch and checkout a specific branch of the repository.
      git-delete                   Remove a repository from the Machtiani system.
      status                       Check the status of the current project.

    Global Flags:
      -file string                 Path to the markdown file (optional).
      -project string              Name of the project (optional).
      -model string                Model to use (options: gpt-4o, gpt-4o-mini; default: gpt-4o-mini).
      -match-strength string       Match strength (options: high, mid, low; default: mid).
      -mode string                 Search mode (options: pure-chat, commit, super; default: commit).
      --force                      Skip confirmation prompt and proceed with the operation.
      -verbose                     Enable verbose output.

    Subcommands:

    git-store:
      Usage: machtiani git-store --branch <default_branch> --remote <remote_name> [--force]
      Adds a repository to Machtiani system.
      Flags:
        --branch string            Name of the default branch (required).
        --remote string            Name of the remote repository (default: "origin").
        --force                    Skip confirmation prompt.

    git-sync:
      Usage: machtiani git-sync --branch <default_branch> --remote <remote_name> [--force]
      Syncs with a specific branch of the repository.
      Flags:
        --branch string            Name of the branch (required).
        --remote string            Name of the remote repository (default: "origin").
        --force                    Skip confirmation prompt.

    git-delete:
      Usage: machtiani git-delete --remote <remote_name> [--force]
      Removes a repository from Machtiani system.
      Flags:
        --remote string            Name of the remote repository (required).
        --force                    Skip confirmation prompt.

    Examples:
      Providing a direct prompt:
        machtiani "Add a new endpoint to get stats."

      Using an existing markdown chat file:
        machtiani --file .machtiani/chat/add_state_endpoint.md

      Specifying additional parameters:
        machtiani "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high

      Using the '--force' flag to skip confirmation:
        machtiani git-store --branch master --force

    `
    fmt.Println(helpText)
}

