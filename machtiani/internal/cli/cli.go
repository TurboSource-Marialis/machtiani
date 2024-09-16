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

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/git"
    "github.com/7db9a/machtiani/internal/utils"
    "github.com/charmbracelet/glamour"
    "context"
    "github.com/coder/aicommit"
    "github.com/sashabaranov/go-openai"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

// Call the /generate-filename endpoint
func generateFilename(apiKey, context string) (string, error) {
    url := fmt.Sprintf("http://localhost:5071/generate-filename?api_key=%s&context=%s", apiKey, url.QueryEscape(context))
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
    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
    markdownFlag := fs.String("markdown", "", "Path to the markdown file")
    projectFlag := fs.String("project", "", "Name of the project (if not set, it will be fetched from git)")
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: pure-chat, commit, or super")
    verboseFlag := fs.Bool("verbose", false, "Enable verbose output.")

    args := os.Args[1:]

    if len(os.Args) >= 2 && os.Args[1] == "aicommit" {
        // Handle the aicommit subcommand
        runAicommit(args)
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

    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    // Call OpenAI API to generate response
    apiResponse, err := api.CallOpenAIAPI(openAIAPIKey, prompt, project, *modeFlag, *modelFlag, *matchStrengthFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    // Check for error in response
    if errorMsg, ok := apiResponse["error"].(string); ok {
        log.Fatalf("Error from API: %s", errorMsg)
    }

    // Generate filename using the /generate-filename endpoint
    filename, err := generateFilename(openAIAPIKey, prompt)
    if err != nil {
        log.Fatalf("Error generating filename: %v", err)
    }

    // Handle API response and save it to a markdown file with the generated filename
    handleAPIResponse(prompt, apiResponse, filename, *markdownFlag)  // Pass markdownFlag here
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

    markdownContent := createMarkdownContent(prompt, openAIResponse, retrievedFilePaths, markdownFlag) // Pass the correct number of arguments
    renderMarkdown(markdownContent)

    // Save the response to the markdown file with the generated filename
    tempFile, err := utils.CreateTempMarkdownFile(markdownContent, filename) // Pass the filename
    if err != nil {
        log.Fatalf("Error creating markdown file: %v", err)
    }

    fmt.Printf("Response saved to %s\n", tempFile)
}

func runAicommit(args []string) {
    // Create a new flag set for aicommit
    fs := flag.NewFlagSet("aicommit", flag.ExitOnError)

    // Define optional flags with default values
    amendFlag := fs.Bool("amend", false, "Amend the last commit")
    dryRunFlag := fs.Bool("dry-run", false, "Dry run the command")
    modelFlag := fs.String("model", "gpt-4o-mini", "OpenAI model to use")
    maxTokensFlag := fs.Int("max-tokens", 128000, "Maximum number of tokens")
    contextFlag := fs.String("context", "", "Additional context for the commit message")

    // Parse the flags
    fs.Parse(args)

    // Remaining arguments after flags
    remainingArgs := fs.Args()

    // Get the OpenAI API key
    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    // Create an OpenAI client
    client := openai.NewClient(openAIAPIKey)

    // Get the working directory
    workdir, err := os.Getwd()
    if err != nil {
        log.Fatalf("Error getting working directory: %v", err)
    }

    // Determine the commit hash (if any)
    var hash string
    if *amendFlag {
        hash, err = getLastCommitHash()
        if err != nil {
            log.Fatalf("Error getting last commit hash: %v", err)
        }
    } else if len(remainingArgs) > 0 {
        // If a ref is provided
        hash, err = resolveRef(remainingArgs[0])
        if err != nil {
            log.Fatalf("Error resolving ref %q: %v", remainingArgs[0], err)
        }
    } else {
        // No specific hash; use current staged changes
        hash = ""
    }

    // Build the prompt using aicommit's exported function
    msgs, err := aicommit.BuildPrompt(os.Stdout, workdir, hash, *amendFlag, *maxTokensFlag)
    if err != nil {
        log.Fatalf("Error building prompt: %v", err)
    }

    // Add additional context if provided
    if *contextFlag != "" {
        msgs = append(msgs, openai.ChatCompletionMessage{
            Role:    openai.ChatMessageRoleUser,
            Content: *contextFlag,
        })
    }

    // Create a chat completion request
    ctx := context.Background()
    resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
        Model:    *modelFlag,
        Messages: msgs,
    })
    if err != nil {
        log.Fatalf("Error creating chat completion: %v", err)
    }

    // Get the assistant's reply
    assistantMsg := resp.Choices[0].Message.Content

    // Clean up the assistant's message
    assistantMsg = strings.TrimSpace(assistantMsg)
    if strings.HasPrefix(assistantMsg, "```") {
        assistantMsg = strings.Trim(assistantMsg, "` \n")
    }

    // Output the commit message
    fmt.Println("Generated commit message:")
    fmt.Println(assistantMsg)

    if *dryRunFlag {
        fmt.Println("Dry run enabled; not committing changes.")
        return
    }

    // Prepare the git commit command
    cmdArgs := []string{"commit", "-m", assistantMsg}
    if *amendFlag {
        cmdArgs = append(cmdArgs, "--amend")
    }

    // Execute the git commit command
    cmd := exec.Command("git", cmdArgs...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    if err := cmd.Run(); err != nil {
        log.Fatalf("Error running git commit: %v", err)
    }
}

func getLastCommitHash() (string, error) {
    cmd := exec.Command("git", "rev-parse", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
}

func resolveRef(ref string) (string, error) {
    cmd := exec.Command("git", "rev-parse", ref)
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(output)), nil
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
