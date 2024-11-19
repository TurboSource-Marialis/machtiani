package cli

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "path"
    "strings"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
    "github.com/charmbracelet/glamour"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

func handlePrompt(args []string, config *utils.Config, remoteURL *string, apiKey *string) {
    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: pure-chat, commit, or super")
    fileFlag := fs.String("file", "", "Path to the markdown file")
    forceFlag := fs.Bool("force", false, "Force the operation")
    verboseFlag := fs.Bool("verbose", false, "Enable verbose output")

    // Parse the flags from args
    err := fs.Parse(args)
    if err != nil {
        log.Fatalf("Error parsing flags: %v", err)
    }

    // Collect non-flag arguments (the prompt)
    promptParts := fs.Args()
    prompt := strings.Join(promptParts, " ")

    // If --file flag is provided, read the content from the file
    if *fileFlag != "" {
        content, err := ioutil.ReadFile(*fileFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(content) // Set prompt to the content of the file
    } else if prompt == "" {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    if *verboseFlag {
        printVerboseInfo(*fileFlag, *modelFlag, *matchStrengthFlag, *modeFlag, prompt)
    }

    apiResponse, err := api.GenerateResponse(prompt, *remoteURL, *modeFlag, *modelFlag, *matchStrengthFlag, *forceFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    if errorMsg, ok := apiResponse["error"].(string); ok {
        log.Fatalf("Error from API: %s", errorMsg)
    }

    // Determine the filename to save the response
    filename := path.Base(*fileFlag)

    // Strip all extensions from the filename
    for ext := path.Ext(filename); ext != ""; ext = path.Ext(filename) {
        filename = strings.TrimSuffix(filename, ext)
    }

    if filename == "" || filename == "." {
        filename, err = generateFilename(prompt, config.Environment.ModelAPIKey)
        if err != nil {
            log.Fatalf("Error generating filename: %v", err)
        }
    }

    handleAPIResponse(prompt, apiResponse, filename, *fileFlag)
}

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

func printVerboseInfo(markdown, model, matchStrength, mode, prompt string) {
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", markdown)
    fmt.Printf("Model: %s\n", model)
    fmt.Printf("Match strength: %s\n", matchStrength)
    fmt.Printf("Mode: %s\n", mode)
    fmt.Printf("Prompt: %s\n", prompt)
}
