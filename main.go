package main

import (
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "net/http"
    "net/url"
    "os"
    "strings"
    "encoding/json"
    "github.com/charmbracelet/glamour"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
)

func main() {
    // Add command-line flags for optional arguments
    markdownFlag := flag.String("markdown", "", "Path to the markdown file")
    projectFlag := flag.String("project", "", "Name of the project (if not set, it will be fetched from git)")
    modelFlag := flag.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := flag.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")

    // Parse command-line flags
    flag.Parse()
    args := flag.Args()

    // Print the arguments passed
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", *markdownFlag)
    fmt.Printf("Project name: %s\n", *projectFlag)
    fmt.Printf("Model: %s\n", *modelFlag)
    fmt.Printf("Match strength: %s\n", *matchStrengthFlag)
    fmt.Printf("Remaining args (prompt): %v\n", args)

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

    // Determine the prompt source (file or command-line argument)
    var prompt string
    if *markdownFlag != "" {
        content, err := ioutil.ReadFile(*markdownFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(content)
    } else if len(args) > 0 {
        prompt = args[0]
    } else {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    // Ensure the OpenAI API key is set
    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    // Prepare API request
    encodedPrompt := url.QueryEscape(prompt)
    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=commit&model=%s&api_key=%s&match_strength=%s",
        encodedPrompt, project, model, openAIAPIKey, matchStrength)

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

    openAIResponse, ok := response["openai_response"].(string)
    if !ok {
        log.Fatalf("Error: openai_response key missing or not a string in the response")
    }
    openAIResponse = strings.ReplaceAll(openAIResponse, "\\n", "\n")
    openAIResponse = strings.ReplaceAll(openAIResponse, "\\\"", "\"")

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

