package cli

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"log"
	"net/http"
	"net/url"
	//"os/exec" // No longer needed here for git apply
	"path"
	"path/filepath"
	"strings"
	"time" // Added for generateFilename timeout
	"errors" // Added for generateFilename error handling

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git" // Import the git package
	"github.com/7db9a/machtiani/internal/utils"
	"github.com/charmbracelet/glamour"
	"github.com/spf13/pflag"
)

var (
	MachtianiURL string = "http://localhost:5071"
)

const (
	defaultModel         = "gpt-4o-mini"
	defaultMatchStrength = "mid"
	defaultMode          = "default"
)

const (
	CONTENT_TYPE_KEY     = "Content-Type"
	CONTENT_TYPE_VALUE   = "application/json"
	API_GATEWAY_HOST_KEY = "X-RapidAPI-Key"
)


// Function to create a visual separator
func createSeparator(message string) string {
	separator := strings.Repeat("=", 60)
	if message == "" { // Handle empty message for just a line break separator
		return fmt.Sprintf("\n%s\n", separator)
	}
	return fmt.Sprintf("\n%s\n%s\n%s\n", separator, message, separator)
}


func handlePrompt(args []string, config *utils.Config, remoteURL *string, apiKey *string, headCommitHash string) {
	fs := pflag.NewFlagSet("machtiani", pflag.ContinueOnError)
	modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
	matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")

	modeFlag := fs.String("mode", defaultMode, "Search mode: chat, pure-chat, answer-only, or default")
	fileFlag := fs.String("file", "", "Path to the markdown file")
	forceFlag := fs.Bool("force", false, "Force the operation")
	verboseFlag := fs.Bool("verbose", false, "Enable verbose output")

	// Parse the flags from args
	err := fs.Parse(args)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}


	// Check if we're in answer-only mode early
	isAnswerOnlyMode := *modeFlag == "answer-only"

	// Suppress all logging output if mode is answer-only
	if isAnswerOnlyMode {
		log.SetOutput(ioutil.Discard)
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

	if *verboseFlag && *modeFlag != "answer-only" {
		printVerboseInfo(*fileFlag, *modelFlag, *matchStrengthFlag, *modeFlag, prompt)
	}

	// Call GenerateResponse to get the streamed response
	result, err := api.GenerateResponse(prompt, *remoteURL, *modeFlag, *modelFlag, *matchStrengthFlag, *forceFlag, headCommitHash)

	if err != nil {
		log.Fatalf("Error making API call: %v", err)
	}



	// Only process patches in default mode
	if *modeFlag == defaultMode {
		utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%s", createSeparator("Writing & Applying File Patches"))

		patchDir := filepath.Join(".machtiani", "patches")

		// Get list of existing patch files BEFORE writing new ones
		existingPatchFiles, err := filepath.Glob(filepath.Join(patchDir, "*.patch"))
		if err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Warning: Error finding existing patch files")
			existingPatchFiles = []string{}
		}

		// Write the new patch files (this part remains here)
		if err := result.WritePatchToFile(); err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error writing patch file(s)")
			// Decide if we should proceed to apply patches even if writing failed?
			// Let's assume if writing failed, there are no *new* patches to apply.
		} else {
			// --- START REFACTOR ---
			// Call the function from git_utils to apply the patches
			err = git.ApplyGitPatches(patchDir, existingPatchFiles)
			if err != nil {
				utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error during patch application process")
			}
			// --- END REFACTOR ---
		}


		// New section for writing new files
		utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%s", createSeparator("Writing New Files"))

		// Call the modified WriteNewFiles function and print its output
		output, err := result.WriteNewFiles()
		if output != "" {
			utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%s", output)
		}
		if err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error writing new files")
		}

		// Add a separator after writing/applying patches and new files
		utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%s", createSeparator(""))
	}

	// Collect the final OpenAI response for further processing if needed
	rawResponse := result.RawResponse
	retrievedFilePaths := result.RetrievedFilePaths


	// In answer-only mode, we don't need to generate any filename
	var filename string
	if !isAnswerOnlyMode {
		// Determine the filename to save the response
		filename = path.Base(*fileFlag)

		// Strip all extensions from the filename
		for ext := path.Ext(filename); ext != ""; ext = path.Ext(filename) {
			filename = strings.TrimSuffix(filename, ext)
		}

		// Generate a filename if necessary
		if filename == "" || filename == "." {
			filename, err = generateFilename(prompt, config.Environment.ModelAPIKey, config.Environment.ModelBaseURL)
			if err != nil {
				log.Printf("Warning: Error generating filename: %v. Using default.", err)
				filename = "machtiani-response"
			}
		}
	}


	utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%s", createSeparator("Saving Chat Response"))


	// Handle the final API response with structured data, passing the isAnswerOnlyMode flag
	handleAPIResponse(prompt, rawResponse, retrievedFilePaths, filename, *fileFlag, isAnswerOnlyMode)
}




func handleAPIResponse(prompt, openaiResponse string, retrievedFilePaths []string, filename, fileFlag string, isAnswerOnlyMode bool) {
    // In answer-only mode, just print the raw response without any file operations
    if isAnswerOnlyMode {
        return
    }

    // For other modes, continue with file creation and structured output
    var finalContent string
    finalContent = openaiResponse

    tempFile, err := utils.CreateTempMarkdownFile(finalContent, filename)
    if err != nil {
        log.Printf("Error creating markdown file '%s': %v", filename+".md", err)
        fmt.Println("\n--- Start Fallback Response Output ---")
        fmt.Println(finalContent)
        fmt.Println("--- End Fallback Response Output ---")
        return
    }

    fmt.Printf("Response saved to %s\n", tempFile)
}


func generateFilename(context string, llmModelApiKey string, llmModelBaseUrl string) (string, error) {
	config, err := utils.LoadConfig()
	if err != nil {
		// Don't make this fatal within this specific function
		return "", fmt.Errorf("error loading config: %w", err)
	}
	endpoint := MachtianiURL // Assuming MachtianiURL is set globally or via config/env

	if endpoint == "" {
		return "", fmt.Errorf("MACHTIANI_URL environment variable is not set")
	}

	// Construct URL properly
	baseURL, err := url.Parse(endpoint)
	if err != nil {
		return "", fmt.Errorf("invalid MACHTIANI_URL: %w", err)
	}
	baseURL.Path = path.Join(baseURL.Path, "/generate-filename") // Use path.Join

	// Prepare query parameters
	params := url.Values{}
	params.Add("context", context)
	// Only add keys/URLs if they are actually configured/needed by the endpoint
	if llmModelApiKey != "" {
		params.Add("llm_model_api_key", llmModelApiKey)
	}
	if llmModelBaseUrl != "" {
		params.Add("llm_model_base_url", llmModelBaseUrl)
	}
	if config.Environment.ModelBaseURLOther != "" {
		params.Add("llm_model_base_url_other", config.Environment.ModelBaseURLOther)
	}
	if config.Environment.ModelAPIKeyOther != "" {
		params.Add("llm_model_api_key_other", config.Environment.ModelAPIKeyOther)
	}
	baseURL.RawQuery = params.Encode()

	req, err := http.NewRequest("GET", baseURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add headers
	if config.Environment.APIGatewayHostValue != "" && config.Environment.APIGatewayHostKey != "" {
		req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
	} else if config.Environment.APIGatewayHostValue != "" { // Fallback to default key if only value is set
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE) // Content-Type might not be strictly needed for GET

	client := &http.Client{Timeout: time.Second * 15} // Add a timeout
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to call generate-filename endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, readErr := ioutil.ReadAll(resp.Body) // Read body first for better error reporting
	if resp.StatusCode != http.StatusOK {
		if readErr != nil { // Handle error reading body as well
			return "", fmt.Errorf("generate-filename endpoint returned status %d. Failed to read response body: %w", resp.StatusCode, readErr)
		}
		return "", fmt.Errorf("generate-filename endpoint returned status %d: %s", resp.StatusCode, string(body))
	}
	if readErr != nil { // Handle read error even if status is OK
		return "", fmt.Errorf("failed to read response body from generate-filename endpoint: %w", readErr)
	}

	var responseData struct { // Expecting a JSON object like {"filename": "..."}
		Filename string `json:"filename"`
	}
	// Use json.Unmarshal instead of Decode for byte slice
	if err := json.Unmarshal(body, &responseData); err != nil {
		// Log the body content for debugging if unmarshal fails
		log.Printf("Failed to decode JSON response from generate-filename. Body: %s", string(body))
		return "", fmt.Errorf("failed to decode response from generate-filename endpoint: %w", err)
	}

	if responseData.Filename == "" {
		return "", errors.New("generate-filename endpoint returned an empty filename")
	}

	return responseData.Filename, nil
}

// createMarkdownContent - unchanged
func createMarkdownContent(prompt, openAIResponse string, retrievedFilePaths []string, fileFlag string) string {
	var markdownContent string
	if fileFlag != "" {
		// Ensure reading the file doesn't cause a fatal error if it fails here
		// It should have been read successfully earlier in handlePrompt
		content, err := ioutil.ReadFile(fileFlag)
		if err != nil {
			log.Printf("Warning: could not re-read markdown file %s for content creation: %v", fileFlag, err)
			// Fallback to just using the prompt string if file read fails here
			markdownContent = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
		} else {
			markdownContent = fmt.Sprintf("%s\n\n# Assistant\n\n%s", string(content), openAIResponse)
		}
	} else {
		markdownContent = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n%s", prompt, openAIResponse)
	}

	if len(retrievedFilePaths) > 0 {
		markdownContent += "\n\n# Retrieved File Paths\n\n"
		for _, path := range retrievedFilePaths {
			markdownContent += fmt.Sprintf("- `%s`\n", path) // Added backticks for code formatting
		}
	}

	return markdownContent
}

// renderMarkdown - unchanged
func renderMarkdown(content string) {
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120), // Adjust wrap width as needed
	)
	if err != nil {
		// Log error but perhaps fallback to plain print
		log.Printf("Error creating glamour renderer: %v. Printing raw content.", err)
		fmt.Println(content)
		return
	}

	out, err := renderer.Render(content)
	if err != nil {
		// Log error but perhaps fallback to plain print
		log.Printf("Error rendering Markdown with glamour: %v. Printing raw content.", err)
		fmt.Println(content)
		return
	}

	fmt.Println(out)
}

// readMarkdownFile - unchanged (though maybe make it return error instead of fatal)
func readMarkdownFile(path string) string {
	content, err := ioutil.ReadFile(path)
	if err != nil {
		// This is called from createMarkdownContent which now handles potential errors
		log.Fatalf("Error reading markdown file: %v", err) // Keep fatal here if initial read must succeed
	}
	return string(content)
}

// printVerboseInfo - unchanged

func printVerboseInfo(markdown, model, matchStrength, mode, prompt string) {
	_, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		log.Printf("Warning: Error loading config/ignore files for verbose info: %v", err)
	} else {
		// Print the file paths
		fmt.Println("Parsed file paths from machtiani.ignore:")
		if len(ignoreFiles) > 0 {
			for _, path := range ignoreFiles {
				fmt.Printf("  %s\n", path)
			}
		} else {
			fmt.Println("  (No ignore rules found or file doesn't exist)")
		}
	}

	fmt.Println("Arguments passed:")
	fmt.Printf("  Markdown file: %s\n", markdown)
	fmt.Printf("  Model: %s\n", model)
	fmt.Printf("  Match strength: %s\n", matchStrength)
	fmt.Printf("  Mode: %s\n", mode)
	// Truncate long prompts in verbose output?
	maxPromptLen := 200
	truncatedPrompt := prompt
	if len(prompt) > maxPromptLen {
		truncatedPrompt = prompt[:maxPromptLen] + "..."
	}
	fmt.Printf("  Prompt: %s\n", truncatedPrompt)
	// If you want to print token counts here, use:
	// fmt.Printf("  Embedding tokens: %s\n", utils.FormatIntWithCommas(embeddingTokens))
	// fmt.Printf("  Inference tokens: %s\n", utils.FormatIntWithCommas(inferenceTokens))
}
