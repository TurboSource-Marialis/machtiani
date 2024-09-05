package main

import (
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

func main() {
    // Initialize variables
    markdownFlag := false
    markdownFile := ""

    // Parse command line arguments
    args := os.Args[1:]
    for len(args) > 0 && strings.HasPrefix(args[0], "--") {
        switch args[0] {
        case "--markdown":
            if len(args) < 2 {
                log.Fatal("Error: --markdown flag requires a file path argument.")
            }
            markdownFlag = true
            markdownFile = args[1]
            args = args[2:]
        default:
            log.Fatalf("Unknown option: %s\n", args[0])
        }
    }

    // Check if required arguments are provided
    if len(args) < 2 {
        fmt.Println("Usage: go run script.go [--markdown <markdown_file>] <project> <match_strength>")
        os.Exit(1)
    }

    // Assign arguments to variables
    var prompt string
    if markdownFlag {
        content, err := ioutil.ReadFile(markdownFile)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v\n", err)
        }
        prompt = string(content)
    } else {
        prompt = args[0]
        args = args[1:]
    }

    project := args[0]
    matchStrength := args[1]

    // Check if OPENAI_API_KEY is set
    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    // URL encode the prompt
    encodedPrompt := url.QueryEscape(prompt)

    // Make the API call
    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=commit&model=gpt-4o-mini&api_key=%s&match_strength=%s",
        encodedPrompt, project, openAIAPIKey, matchStrength)
    resp, err := http.Post(apiURL, "application/json", nil)
    if err != nil {
        log.Fatalf("Error making API call: %v\n", err)
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Error reading response body: %v\n", err)
    }

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        log.Fatalf("Error parsing JSON response: %v\n", err)
    }

    openAIResponse := response["openai_response"].(string)
    openAIResponse = strings.ReplaceAll(openAIResponse, "\\n", "\n")
    openAIResponse = strings.ReplaceAll(openAIResponse, "\\\"", "\"")

    // Create a temporary directory
    tempDir, err := ioutil.TempDir("", "response")
    if err != nil {
        log.Fatalf("Error creating temporary directory: %v\n", err)
    }

    // Save the Markdown content to a file in the temporary directory
    tempFile := fmt.Sprintf("%s/response.md", tempDir)
    var markdownContent string
    if markdownFlag {
        markdownContent = fmt.Sprintf("%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
    } else {
        markdownContent = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
    }

    if err := ioutil.WriteFile(tempFile, []byte(markdownContent), 0644); err != nil {
        log.Fatalf("Error writing to temporary file: %v\n", err)
    }

    // Use the default "dark" style and adjust word wrap and margins
    renderer, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(120), // Adjust word wrap to make the text 1/3 wider
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

