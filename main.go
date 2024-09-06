package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"

    "github.com/charmbracelet/glamour"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

func main() {
    // Define custom flag set
    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)

    // Add command-line flags for optional arguments
    markdownFlag := fs.String("markdown", "", "Path to the markdown file")
    projectFlag := fs.String("project", "", "Name of the project (if not set, it will be fetched from git)")
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: content, commit, or super")

    // Custom argument parsing
    args := os.Args[1:]
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

    // Parse flags
    if err := fs.Parse(flagArgs); err != nil {
        log.Fatalf("Error parsing flags: %v", err)
    }

    // Join prompt parts
    prompt := strings.Join(promptParts, " ")

    // Retrieve project name from Git if not provided
    var project string
    var err error
    if *projectFlag == "" {
        project, err = getProjectName()
        if err != nil {
            log.Fatalf("Error getting project name: %v", err)
        }
    } else {
        project = *projectFlag
    }

    // Validate model argument
    model := *modelFlag
    if model != "gpt-4o" && model != "gpt-4o-mini" {
        log.Fatalf("Error: Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")
    }

    // Validate match strength argument
    matchStrength := *matchStrengthFlag
    if matchStrength != "high" && matchStrength != "mid" && matchStrength != "low" {
        log.Fatalf("Error: Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")
    }

    // Validate mode argument
    mode := *modeFlag
    if mode != "content" && mode != "commit" && mode != "super" {
        log.Fatalf("Error: Invalid mode selected. Choose either 'content', 'commit', or 'super'.")
    }
    fmt.Printf("Debug: Mode selected: %s\n", mode)

    // Determine the prompt source (file or command-line argument)
    if *markdownFlag != "" {
        fileContent, err := ioutil.ReadFile(*markdownFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(fileContent)
    } else if prompt == "" {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    // Print the arguments passed
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", *markdownFlag)
    fmt.Printf("Project name: %s\n", *projectFlag)
    fmt.Printf("Model: %s\n", *modelFlag)
    fmt.Printf("Match strength: %s\n", *matchStrengthFlag)
    fmt.Printf("Mode: %s\n", *modeFlag)
    fmt.Printf("Prompt: %s\n", prompt)

    // Ensure the OpenAI API key is set
    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    // Prepare API request
    encodedPrompt := url.QueryEscape(prompt)
    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=%s&model=%s&api_key=%s&match_strength=%s",
        encodedPrompt, project, mode, model, openAIAPIKey, matchStrength)
    fmt.Printf("Debug: API URL: %s\n", apiURL)

    // Make API request
    resp, err := http.Post(apiURL, "application/json", nil)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }
    defer resp.Body.Close()

    // Read API response
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Error reading response body: %v", err)
    }

    // Parse JSON response
    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        log.Fatalf("Error parsing JSON response: %v", err)
    }
    fmt.Printf("Debug: Full API response: %+v\n", response)

    openAIResponse, ok := response["openai_response"].(string)
    if !ok {
        // Check for the "machtiani" key if "openai_response" is not found
        if machtianiResponse, exists := response["machtiani"].(string); exists && machtianiResponse == "no files found" {
            log.Fatalf("Fatal error: %s", machtianiResponse)
        } else {
            log.Fatalf("Error: openai_response key missing and no fatal machtiani condition met")
        }
    }

    retrievedFilePaths, ok := response["retrieved_file_paths"].([]interface{})
    if !ok {
        log.Fatalf("Error: retrieved_file_paths key missing or invalid")
    }

    // Convert retrieved file paths to a slice of strings
    var filePaths []string
    for _, path := range retrievedFilePaths {
        filePath, ok := path.(string)
        if !ok {
            log.Fatalf("Error: invalid file path in retrieved_file_paths")
        }
        filePaths = append(filePaths, filePath)
    }

    // Create a temporary directory
    tempDir, err := ioutil.TempDir("", "response")
    if err != nil {
        log.Fatalf("Error creating temporary directory: %v", err)
    }

    // Save the Markdown content to a file in the temporary directory
    tempFile := fmt.Sprintf("%s/response.md", tempDir)
    var markdownContent string
    if *markdownFlag != "" {
        markdownContent = fmt.Sprintf("%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
    } else {
        markdownContent = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
    }

    fmt.Printf("Debug: filePaths: %+v\n", filePaths)

    if len(filePaths) > 0 {
        markdownContent += "\n\n# Retrieved Files\n\n"
        for _, filePath := range filePaths {
            markdownContent += fmt.Sprintf("- %s\n", filePath)
        }
    }

    if err := ioutil.WriteFile(tempFile, []byte(markdownContent), 0644); err != nil {
        log.Fatalf("Error writing to temporary file: %v", err)
    }

    // Use glamour to render the Markdown content
    renderer, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(120),
    )
    if err != nil {
        log.Fatalf("Error creating renderer: %v", err)
    }

    // Render the Markdown content
    out, err := renderer.Render(markdownContent)
    if err != nil {
        log.Fatalf("Error rendering Markdown: %v", err)
    }

    // Print the rendered content to the terminal
    fmt.Println(out)

    // Print out the path to the file
    fmt.Printf("Response saved to %s and opened in your default Markdown viewer.\n", tempFile)
}

