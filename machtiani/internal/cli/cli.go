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

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/git"
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
func generateFilename(context string) (string, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    endpoint := config.Environment.MachtianiURL
    if endpoint == "" {
        return "", fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }

    url := fmt.Sprintf("%s/generate-filename?context=%s", endpoint, url.QueryEscape(context))
    resp, err := http.Get(url)
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
    markdownFlag := fs.String("markdown", "", "Path to the markdown file")
    projectFlag := fs.String("project", "", "Name of the project (if not set, it will be fetched from git)")
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: pure-chat, commit, or super")
    verboseFlag := fs.Bool("verbose", false, "Enable verbose output.")

    codeURL := fs.String("code-url", "", "URL of the code repository")
    name := fs.String("name", "", "Project name")
    branchName := fs.String("branch-name", "", "Branch name")

    args := os.Args[1:]

    if len(os.Args) >= 2 && os.Args[1] == "git-store" {
        err := fs.Parse(args[1:]) // Parse flags after the command
        if err != nil {
            log.Fatalf("Error parsing flags: %v", err)
        }

        // Use the code host URL and API key from config
        if *codeURL == "" {
            *codeURL = config.Environment.CodeHostURL // Use the configuration value
        }
        if *name == "" || config.Environment.CodeHostAPIKey == "" {
            log.Fatal("Error: project name must be provided and API key must be set in config.")
        }

        // Call the new function to add the repository
        responseMessage, err := api.AddRepository(*codeURL, *name, config.Environment.CodeHostAPIKey, config.Environment.RepoManagerURL)
        if err != nil {
            log.Fatalf("Error adding repository: %v", err)
        }

        fmt.Printf("Response from server: %s\n", responseMessage.Message)
        fmt.Printf("Full Path: %s\n", responseMessage.FullPath)
        fmt.Printf("API Key Provided: %v\n", responseMessage.ApiKeyProvided)
        return // Exit after handling git-store
    }

    // Check if the command is git-sync
    if len(os.Args) >= 2 && os.Args[1] == "git-sync" {
        err := fs.Parse(args[1:]) // Parse flags after the command
        if err != nil {
            log.Fatalf("Error parsing flags: %v", err)
        }

        // Use the code host URL and API key from config
        if *codeURL == "" {
            *codeURL = config.Environment.CodeHostURL // Use the configuration value
        }
        if *name == "" || *branchName == "" || config.Environment.CodeHostAPIKey == "" {
            log.Fatal("Error: all flags --name, --branch-name must be provided and API key must be set in config.")
        }

        // Call the new function to fetch and checkout the branch
        err = api.FetchAndCheckoutBranch(*codeURL, *name, *branchName, config.Environment.CodeHostAPIKey)
        if err != nil {
            log.Fatalf("Error syncing repository: %v", err)
        }

        log.Printf("Successfully synced the repository: %s", *name)
        return
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
    project, err := getProjectOrDefault(projectFlag)
    if err != nil {
        log.Fatalf("Error getting project name: %v", err)
    }

    validateFlags(modelFlag, matchStrengthFlag, modeFlag)

    if *markdownFlag != "" {
        content, err := ioutil.ReadFile(*markdownFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(content)
    } else if prompt == "" {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    if *verboseFlag {
        printVerboseInfo(*markdownFlag, *projectFlag, *modelFlag, *matchStrengthFlag, *modeFlag, prompt)
    }

    // Only needed if generating embeddings for the prompt, client side, otherwise, server will do it if allowed.
    openAIAPIKey := config.Environment.OpenAIAPIKey
    if openAIAPIKey != "" {
        log.Println("Warning: Using OPENAI_MACHTIANI_API_KEY. This may incur costs for generating embeddings.")
    }

    // Call OpenAI API to generate response
    apiResponse, err := api.CallOpenAIAPI(prompt, project, *modeFlag, *modelFlag, *matchStrengthFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    // Check for error in response
    if errorMsg, ok := apiResponse["error"].(string); ok {
        log.Fatalf("Error from API: %s", errorMsg)
    }

    // Determine the filename to save the response
    filename := path.Base(*markdownFlag) // Extract filename with extension

    // Strip all extensions from the filename
    for ext := path.Ext(filename); ext != ""; ext = path.Ext(filename) {
        filename = strings.TrimSuffix(filename, ext)
    }

    // Check if the filename is empty or just "."
    if filename == "" || filename == "." {
        // If no markdown file provided, generate a filename
        filename, err = generateFilename(prompt)
        if err != nil {
            log.Fatalf("Error generating filename: %v", err)
        }
    }

    // Handle API response and save it to a markdown file with the generated filename
    handleAPIResponse(prompt, apiResponse, filename, *markdownFlag) // Pass filename here
}

func handleAPIResponse(prompt string, apiResponse map[string]interface{}, filename string, markdownFlag string) {
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
    openaiKey := fs.String("openai-key", config.Environment.OpenAIAPIKey, "OpenAI API Key")
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

func getProjectOrDefault(projectFlag *string) (string, error) {
    if *projectFlag == "" {
        return git.GetProjectName()
    }
    return *projectFlag, nil
}

func validateFlags(modelFlag, matchStrengthFlag, modeFlag *string) {
    model := *modelFlag
    if model != "gpt-4o" && model != "gpt-4o-mini" {
        log.Fatalf("Error: Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")
    }

    matchStrength := *matchStrengthFlag
    if matchStrength != "high" && matchStrength != "mid" && matchStrength != "low" {
        log.Fatalf("Error: Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")
    }

    mode := *modeFlag
    if mode != "pure-chat" && mode != "commit" && mode != "super" {
        log.Fatalf("Error: Invalid mode selected. Choose either 'chat', 'commit', or 'super'.")
    }
}

func printVerboseInfo(markdown, project, model, matchStrength, mode, prompt string) {
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", markdown)
    fmt.Printf("Project name: %s\n", project)
    fmt.Printf("Model: %s\n", model)
    fmt.Printf("Match strength: %s\n", matchStrength)
    fmt.Printf("Mode: %s\n", mode)
    fmt.Printf("Prompt: %s\n", prompt)
}


func createMarkdownContent(prompt, openAIResponse string, retrievedFilePaths []string, markdownFlag string) string {
    var markdownContent string
    if markdownFlag != "" {
        markdownContent = fmt.Sprintf("%s\n\n# Assistant\n\n%s", readMarkdownFile(markdownFlag), openAIResponse)
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
