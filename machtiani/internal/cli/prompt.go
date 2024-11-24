package cli
import (
    "encoding/json"
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
    "github.com/spf13/pflag"
)

var (
    MachtianiURL string = "http://localhost:5071"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

const (
    CONTENT_TYPE_KEY   = "Content-Type"
    CONTENT_TYPE_VALUE = "application/json"
    API_GATEWAY_HOST_KEY = "X-RapidAPI-Key"
)

func handlePrompt(args []string, config *utils.Config, remoteURL *string, apiKey *string) {
    fs := pflag.NewFlagSet("machtiani", pflag.ContinueOnError)
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

    // Call GenerateResponse to get the streamed response
    result, err := api.GenerateResponse(prompt, *remoteURL, *modeFlag, *modelFlag, *matchStrengthFlag, *forceFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    // Collect the final OpenAI response for further processing if needed
    openaiResponse := result.OpenAIResponse
    retrievedFilePaths := result.RetrievedFilePaths

    // Determine the filename to save the response
    filename := path.Base(*fileFlag)

    // Strip all extensions from the filename
    for ext := path.Ext(filename); ext != ""; ext = path.Ext(filename) {
        filename = strings.TrimSuffix(filename, ext)
    }

    // Generate a filename if necessary
    if filename == "" || filename == "." {
        filename, err = generateFilename(prompt, config.Environment.ModelAPIKey)
        if err != nil {
            log.Fatalf("Error generating filename: %v", err)
        }
    }

    // Handle the final API response with structured data
    handleAPIResponse(prompt, openaiResponse, retrievedFilePaths, filename, *fileFlag)
}

func handleAPIResponse(prompt, openaiResponse string, retrievedFilePaths []string, filename, fileFlag string) {
    // Create markdown content with the prompt, OpenAI response, and retrieved file paths
    markdownContent := createMarkdownContent(prompt, openaiResponse, retrievedFilePaths, fileFlag)

    // Render the markdown content (assuming renderMarkdown handles the display)
    //renderMarkdown(markdownContent)

    // Save the response to the markdown file with the provided filename
    tempFile, err := utils.CreateTempMarkdownFile(markdownContent, filename) // Pass the filename
    if err != nil {
        log.Fatalf("Error creating markdown file: %v", err)
    }

    fmt.Printf("\n\nResponse saved to %s\n", tempFile)

    // Optionally, handle retrieved file paths further if needed
    //if len(retrievedFilePaths) > 0 {
    //    log.Println("Retrieved File Paths:")
    //    for _, path := range retrievedFilePaths {
    //        log.Println(path)
    //    }
    //} else {
    //    log.Println("No file paths were retrieved.")
    //}
}

func generateFilename(context string, apiKey string) (string, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    endpoint := MachtianiURL
    if endpoint == "" {
        return "", fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }

    url := fmt.Sprintf("%s/generate-filename?context=%s&api_key=%s", endpoint, url.QueryEscape(context), url.QueryEscape(apiKey))

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", fmt.Errorf("failed to create request: %v", err)
    }

    if config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

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
    _, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    // Print the file paths
    fmt.Println("Parsed file paths from machtiani.ignore:")
    for _, path := range ignoreFiles {
        fmt.Printf(" %s\n", path)
    }
    fmt.Println("Arguments passed:")
    fmt.Printf("  Markdown file: %s\n", markdown)
    fmt.Printf("  Model: %s\n", model)
    fmt.Printf("  Match strength: %s\n", matchStrength)
    fmt.Printf("  Mode: %s\n", mode)
    fmt.Printf("  Prompt: %s\n", prompt)
}
