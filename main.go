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

const (
    defaultModel      = "gpt-4o-mini"
    defaultMatchStrength = "mid"
)

func main() {
    markdownFlag := false
    markdownFile := ""
    model := defaultModel
    matchStrength := defaultMatchStrength

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
        case "--model":
            if len(args) < 2 {
                log.Fatal("Error: --model flag requires a model name.")
            }
            model = args[1]
            if model != "gpt-4o" && model != "gpt-4o-mini" {
                log.Fatal("Error: Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")
            }
            args = args[2:]
        case "--match-strength":
            if len(args) < 2 {
                log.Fatal("Error: --match-strength flag requires a strength option.")
            }
            matchStrength = args[1]
            if matchStrength != "high" && matchStrength != "mid" && matchStrength != "low" {
                log.Fatal("Error: Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")
            }
            args = args[2:]
        default:
            log.Fatalf("Unknown option: %s\n", args[0])
        }
    }

    if len(args) < 2 {
        fmt.Println("Usage: go run script.go [--markdown <markdown_file>] <project> <match_strength>")
        os.Exit(1)
    }

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

    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    encodedPrompt := url.QueryEscape(prompt)

    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=commit&model=%s&api_key=%s&match_strength=%s",
        encodedPrompt, project, model, openAIAPIKey, matchStrength)
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

    openAIResponse, ok := response["openai_response"].(string)
    if !ok {
        log.Fatalf("Error: openai_response key missing or not a string in the response")
    }
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

    // Use glamour to render the Markdown content
    renderer, err := glamour.NewTermRenderer(
        glamour.WithAutoStyle(),
        glamour.WithWordWrap(120),
    )
    if err != nil {
        log.Fatalf("Error creating renderer: %v", err)
    }

    out, err := renderer.Render(markdownContent)
    if err != nil {
        log.Fatalf("Error rendering Markdown: %v", err)
    }

    fmt.Println(out)

    // Print out the path to the file
    fmt.Printf("Response saved to %s and opened in your default Markdown viewer.\n", tempFile)
}

