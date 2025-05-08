package api

import (
	"bytes"
   "encoding/json"
   "errors"
   "fmt"
   "io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/turboSource-marialis/machtiani/mct/internal/utils"
	"github.com/charmbracelet/glamour"
)

var (
	HeadOID               string = "none"
	BuildDate             string = "unknown"
	MachtianiURL          string = "http://localhost:5071"
	RepoManagerURL        string = "http://localhost:5070"
	MachtianiGitRemoteURL string = "none"
)

func extractRepoName(projectURL string) string {
	parts := strings.Split(projectURL, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if parts[i] != "" {
			return parts[i]
		}
	}
	return projectURL
}

var (
	renderer     *glamour.TermRenderer
	rendererOnce sync.Once
	rendererErr  error
)

const (
	CONTENT_TYPE_KEY     = "Content-Type"
	CONTENT_TYPE_VALUE   = "application/json"
	API_GATEWAY_HOST_KEY = "X-RapidAPI-Key"
)

type AddRepositoryResponse struct {
	Message                string `json:"message"`
	FullPath               string `json:"full_path"`
	ApiKeyProvided         bool   `json:"api_key_provided"`
	LlmModelApiKeyProvided bool   `json:"llm_model_api_key_provided"`
}

type DeleteStoreResponse struct {
	Message string `json:"message"`
}

type LoadResponse struct {
	EmbeddingTokens int `json:"embedding_tokens"`
	InferenceTokens int `json:"inference_tokens"`
}


type StatusResponse struct {
	LockFilePresent  bool    `json:"lock_file_present"`
	LockTimeDuration float64 `json:"lock_time_duration"`
	ErrorLogs        string  `json:"error_logs"`         // New field added
}

// EstimateTokenCount calls the token-count endpoint and returns embedding and inference token counts.
func EstimateTokenCount(codeURL string, name string, apiKey *string) (int, int, error) {
	countTokenRequestData := map[string]interface{}{
		"codehost_url": codeURL,
		"project_name": name,
		"vcs_type":     "git",
		"api_key":      apiKey,
	}

	repoManagerURL := RepoManagerURL
	if repoManagerURL == "" {
		return 0, 0, fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
	}
	// Convert data to JSON
	countTokenRequestJson, err := json.Marshal(countTokenRequestData)
	if err != nil {
		return 0, 0, fmt.Errorf("error marshaling JSON: %w", err)
	}

	tokenCountEmbedding, tokenCountInference, err := getTokenCount(fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(countTokenRequestJson))
	if err != nil {
		return 0, 0, fmt.Errorf("error getting token count: %w", err)
	}

	return tokenCountEmbedding, tokenCountInference, nil
}


func AddRepository(codeURL, name string, apiKey *string, openAIAPIKey, repoManagerURL, llmModelBaseURL string, force bool, headCommitHash string, useMockLLM bool, amplificationLevel string, depthLevel int, llmThreads int, model string) (AddRepositoryResponse, error) {
	// Load config and ignore files first
	config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return AddRepositoryResponse{}, err
	}

	addRepositoryRequestData := map[string]interface{}{
		"codehost_url":        codeURL,
		"project_name":        name,
		"vcs_type":            "git",
		"api_key":             apiKey,
		"llm_model_api_key":   openAIAPIKey,
		"llm_model_base_url":  llmModelBaseURL,
                "llm_model":           model,
		"ignore_files":        ignoreFiles,
		"head":                headCommitHash,
		"use_mock_llm":        useMockLLM,
		"amplification_level": amplificationLevel,
		"depth_level":         depthLevel,
        "llm_threads":         llmThreads,

	}

	// Only add llm_threads if it's greater than 0
	if llmThreads > 0 {
		addRepositoryRequestData["llm_threads"] = llmThreads
	}

	// Convert data to JSON
	addRepositoryRequestJson, err := json.Marshal(addRepositoryRequestData)
	if err != nil {
		return AddRepositoryResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
	}

	// Proceed with sending the POST request
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(addRepositoryRequestJson))
	if err != nil {
		return AddRepositoryResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// Set API Gateway headers if not blank
	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	client := &http.Client{
		Timeout: 60 * time.Minute, // Increased to 60 minutes
	}
	resp, err := client.Do(req) // Use the client to execute the request
	if err != nil {
		return AddRepositoryResponse{}, fmt.Errorf("error sending request to add repository: %w", err)
	}
	defer resp.Body.Close()

	// Handle the response
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return AddRepositoryResponse{}, fmt.Errorf("error adding repository: %s", body)
	}
	// Successfully added the repository, decode the response into the defined struct
	var responseMessage AddRepositoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&responseMessage); err != nil {
		return AddRepositoryResponse{}, fmt.Errorf("error decoding response: %w", err)
	}

	return responseMessage, nil

}


// FetchAndCheckoutBranch sends a request to fetch and checkout a branch.
func FetchAndCheckoutBranch(codeURL, name, branchName string, apiKey *string, modelAPIKey *string, modelBaseURL *string, model *string, force bool, headCommitHash string, useMockLLM bool, amplificationLevel string, depthLevel int, llmThreads int) (string, error) {
	config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return "", err
	}

	repoManagerURL := RepoManagerURL
	if repoManagerURL == "" {
		return "", fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
	}

	// Initialize and start the spinner
	spinner := NewSpinnerController()
	spinner.Start()
	defer spinner.Stop() // Ensure spinner stops on exit

	fetchAndCheckoutBranchRequestData := map[string]interface{}{
		"codehost_url":        codeURL,
		"project_name":        name,
		"api_key":             apiKey,
		"llm_model_api_key":   modelAPIKey,
		"llm_model_base_url":  modelBaseURL,
                "llm_model":           model,
		"ignore_files":        ignoreFiles,
		"head":                headCommitHash,
		"use_mock_llm":        useMockLLM,
		"amplification_level": amplificationLevel,
		"depth_level":         depthLevel,
		"commit_oid":          headCommitHash,
	}

	// Only add branch_name to the request if it's provided and not empty

	if branchName != "" {
		fetchAndCheckoutBranchRequestData["branch_name"] = branchName
	}

	// Only add llm_threads if it's greater than 0
	if llmThreads > 0 {
		fetchAndCheckoutBranchRequestData["llm_threads"] = llmThreads
	}

	fetchAndCheckoutBranchRequestJson, err := json.Marshal(fetchAndCheckoutBranchRequestData)
	if err != nil {
		return "", fmt.Errorf("error marshaling JSON: %w", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(fetchAndCheckoutBranchRequestJson))
	if err != nil {
		return "", fmt.Errorf("error creating request: %w", err)
	}

	// Set API Gateway headers if not blank
	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	client := &http.Client{
		Timeout: 60 * time.Minute, // Increased to 60 minutes
	}
	resp, err := client.Do(req)
	if err != nil {
		// Detect connection errors and provide a helpful message
		if errors.Is(err, io.EOF) || strings.Contains(err.Error(), "connection refused") {
			return "", fmt.Errorf("could not connect to repository manager at %s. Is the service running? Underlying error: %w", repoManagerURL, err)
		}
		return "", fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("error: received status code %d from the server: %s", resp.StatusCode, body)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	type SyncResponse struct {
		Message     string `json:"message"`
		BranchName  string `json:"branch_name"`
		ProjectName string `json:"project_name"`
	}

	var syncResp SyncResponse
	if err := json.Unmarshal(body, &syncResp); err != nil {
		// Fallback to original format if parsing fails
		log.Printf("Error parsing sync response: %v", err)
		return fmt.Sprintf("Successfully synced the repository: %s.\nServer response: %s", name, string(body)), nil
	}

	repoName := extractRepoName(syncResp.ProjectName)
	formattedMessage := fmt.Sprintf("Successfully synced '%s' branch of %s to the chat service\n - service message: %s",
		syncResp.BranchName, repoName, syncResp.Message)

	return formattedMessage, nil
}

func DeleteStore(projectName string, codehostURL string, vcsType string, apiKey *string, repoManagerURL string, force bool) (DeleteStoreResponse, error) {
	config, _, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return DeleteStoreResponse{}, err
	}

	if force || utils.ConfirmProceed() {
		done := make(chan bool)
		go utils.Spinner(done)

		// Prepare the data to be sent in the request
		data := map[string]interface{}{
			"project_name": projectName,
			"codehost_url": codehostURL,
			"vcs_type":     vcsType,
			"api_key":      apiKey,
		}

		jsonData, err := json.Marshal(data)
		if err != nil {
			return DeleteStoreResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
		}

		req, err := http.NewRequest("POST", fmt.Sprintf("%s/delete-store/", repoManagerURL), bytes.NewBuffer(jsonData))
		if err != nil {
			return DeleteStoreResponse{}, fmt.Errorf("error creating request: %w", err)
		}

		// Set API Gateway headers if not blank
		if config.Environment.APIGatewayHostValue != "" {
			req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
		}
		req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

		client := &http.Client{
			Timeout: 60 * time.Minute, // Increased to 60 minutes
		}
		resp, err := client.Do(req)
		if err != nil {
			return DeleteStoreResponse{}, fmt.Errorf("error sending request to delete store: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := ioutil.ReadAll(resp.Body)
			return DeleteStoreResponse{}, fmt.Errorf("error deleting store: %s", body)
		}

		done <- true

		var responseMessage DeleteStoreResponse
		if err := json.NewDecoder(resp.Body).Decode(&responseMessage); err != nil {
			return DeleteStoreResponse{}, fmt.Errorf("error decoding response: %w", err)
		}

		return responseMessage, nil

	} else {
		abortedResponse := DeleteStoreResponse{
			Message: "Operation aborted by user",
		}

		return abortedResponse, nil
	}
}

type UpdateFileContent struct {
	UpdatedContent string   `json:"updated_content"`
	Errors         []string `json:"errors"`
}

// NewFilesData stores information about suggested new files
type NewFilesData struct {
	NewContent   map[string]string `json:"new_content"`
	NewFilePaths []string          `json:"new_file_paths"`
	Errors       []string          `json:"errors"`
}

type GenerateResponseResult struct {
	LlmModelResponse      string                       `json:"llm_model_response"`
	RawResponse           string                       `json:"llm_model_response"`
	RetrievedFilePaths    []string                     `json:"retrieved_file_paths"`
	UpdateContentResponse map[string]UpdateFileContent `json:"update_content_response"`
	HeadCommitHash        string                       `json:"head_commit_hash"`
	spinner               *SpinnerController
	NewFiles              *NewFilesData `json:"new_files,omitempty"`
}

func init() {
	rendererOnce.Do(func() {
		renderer, rendererErr = glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			//glamour.WithWordWrap(100),        // Reduce width slightly for better readability
			glamour.WithPreservedNewLines(), // This helps with preserving intentional line breaks
		)
		if rendererErr != nil {
			log.Fatalf("Error creating renderer: %v", rendererErr)
		}
	})
}

func GenerateResponse(prompt, project, mode, model, matchStrength string, force bool, headCommitHash string) (*GenerateResponseResult, error) {
	config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	codehostURL, err := utils.GetCodehostURLFromCurrentRepository()
	if err != nil {
		return nil, fmt.Errorf("failed to get codehost URL: %w", err)
	}

	payload := map[string]interface{}{
		"prompt":                   prompt,
		"project":                  project,
		"mode":                     mode,
		"model":                    model,
		"match_strength":           matchStrength,
		"llm_model_api_key":        config.Environment.ModelAPIKey,
		"llm_model_api_key_other":  config.Environment.ModelAPIKeyOther,
		"llm_model_base_url":       config.Environment.ModelBaseURL,
		"codehost_api_key":         config.Environment.CodeHostAPIKey,
		"codehost_url":             codehostURL,
		"ignore_files":             ignoreFiles,
		"head_commit_hash":         headCommitHash,
		"llm_model_base_url_other": config.Environment.ModelBaseURLOther,
	}

	// Log the payload being sent
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}

	endpoint := MachtianiURL

	if endpoint == "" {
		return nil, fmt.Errorf("MACHTIANI_URL environment variable is not set")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/generate-response/", endpoint), bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set API Gateway headers if not blank
	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	// Create a new HTTP client with a timeout

	client := &http.Client{
		Timeout: 60 * time.Minute, // Increased to 60 minutes
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()

	// Check for 422 Unprocessable Entity
	if resp.StatusCode == http.StatusUnprocessableEntity {
		body, _ := ioutil.ReadAll(resp.Body)
		log.Printf("Response body: %s", body) // Log the response body for debugging
		return nil, fmt.Errorf("unprocessable entity: %s", body)
	}

	// Initialize variables
	var completeResponse strings.Builder
	var rawResponse strings.Builder // Initialize rawResponse
	var retrievedFilePaths []string
	var tokenBuffer bytes.Buffer         // Buffer to accumulate tokens
	var inCodeBlock bool                 // Track if we're inside a code block
	var codeBlockBuffer bytes.Buffer     // Buffer to accumulate code block content
	var answerOnlyBuffer strings.Builder // Buffer for answer-only mode

	updateContentResponse := make(map[string]UpdateFileContent)

	// Check if we're in answer-only mode
	answerOnlyMode := mode == "answer-only"
	var spinner *SpinnerController

	// Use a JSON decoder to read multiple JSON objects from the response stream
	decoder := json.NewDecoder(resp.Body)

	// Check if we're in answer-only mode
	var header string
	if answerOnlyMode {
		// In answer-only mode, don't add the User/Assistant headers
		header = prompt
	} else {
		// In other modes, use the existing header logic
		if strings.HasPrefix(strings.TrimSpace(prompt), "# User") {
			// If the prompt already contains the User header, use it as is
			header = fmt.Sprintf("%s\n# Assistant\n\n", prompt)
		} else {
			// Otherwise, prepend the User header
			header = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n", prompt)
		}
	}

	if !answerOnlyMode {
		if err := renderMarkdown(header); err != nil {
			return nil, fmt.Errorf("failed to render header: %w", err)
		}
	}
	completeResponse.WriteString(header)
	rawResponse.WriteString(header)

	// If in answer-only mode, add the header to the buffer
	if answerOnlyMode {
		answerOnlyBuffer.WriteString(header)
	}

	var newFilesResult *NewFilesData

	// Only create and start spinner if not in answer-only mode
	if !answerOnlyMode {
		spinner = NewSpinnerController()
		spinner.Start()
	}

	for {
		var chunk map[string]interface{}
		if err := decoder.Decode(&chunk); err == io.EOF {
			// End of response stream
			break
		} else if err != nil {
			return nil, fmt.Errorf("failed to decode JSON response: %w", err)
		}

		// ────────────────────────────────────────────────────────────────────────
		// catch the file_edit_start event and print a waiting banner
		if ev, ok := chunk["event"].(string); ok && ev == "file_edit_start" {
			if !answerOnlyMode {
				fmt.Println("\n\n\n→ waiting on file‑edits …")
			}
			continue
		}
		// ────

		// Handle error messages
		if errMsg, ok := chunk["error"].(string); ok {
			return nil, fmt.Errorf("API error: %s", errMsg)
		}

		// Handle tokens from OpenAI response
		if token, ok := chunk["token"].(string); ok {
			tokenBuffer.WriteString(token)
			rawResponse.WriteString(token) // Append to rawResponse

			// In answer-only mode, just accumulate tokens
			if answerOnlyMode {
				answerOnlyBuffer.WriteString(token)
				continue // Skip the rendering in answer-only mode
			}

			// Check if buffer contains two consecutive newlines indicating a potential block end
			content := tokenBuffer.String()

			for {
				idx := strings.Index(content, "\n\n")
				if idx == -1 {
					break // No complete block yet
				}

				// Extract the block up to the delimiter
				block := content[:idx]
				remainingContent := content[idx+2:] // Skip the "\n\n"

				// Update inCodeBlock state based on the current block
				lines := strings.Split(block, "\n")
				for _, line := range lines {
					trimmedLine := strings.TrimSpace(line)
					if strings.HasPrefix(trimmedLine, "```") {
						inCodeBlock = !inCodeBlock
						if inCodeBlock {
							// Start spinner when entering a code block
							spinner.Start()
						} else {
							// Stop spinner when exiting a code block
							spinner.Stop()
						}
					}
				}

				if inCodeBlock {
					// Accumulate code block content
					codeBlockBuffer.WriteString(block + "\n\n")
					// Do not render yet; wait until the code block ends
				} else {
					if codeBlockBuffer.Len() > 0 {
						// We're exiting a code block; accumulate the last part
						codeBlockBuffer.WriteString(block)

						// Stop the spinner (in case it's not already stopped)
						spinner.Stop()

						// Render the entire code block
						if err := renderMarkdown(codeBlockBuffer.String()); err != nil {
							log.Printf("Error rendering code block: %v", err)
						}
						// Append to complete response
						completeResponse.WriteString(codeBlockBuffer.String())
						// Reset the code block buffer
						codeBlockBuffer.Reset()
						spinner.Start()
					} else {
						// Trim any trailing newline characters
						trimmedBlock := strings.TrimRight(block, "\r\n")
						// Normalize line endings to Unix-style
						trimmedBlock = strings.ReplaceAll(trimmedBlock, "\r\n", "\n")

						// Stop the spinner
						spinner.Stop()

						// Render the complete block
						if err := renderMarkdown(trimmedBlock); err != nil {
							// Log the error and continue
							log.Printf("Error rendering block: %v", err)
						}

						// Append to complete response
						completeResponse.WriteString(trimmedBlock)
						spinner.Start()
					}
				}

				// Remove the rendered block and the delimiter from the buffer
				content = remainingContent
				tokenBuffer.Reset()
				tokenBuffer.WriteString(content)
			}
		}

		// Handle retrieved file paths
		if paths, ok := chunk["retrieved_file_paths"]; ok {
			pathsJSON, err := json.Marshal(paths)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal retrieved_file_paths: %w", err)
			}

			pathList := []string{}
			if err := json.Unmarshal(pathsJSON, &pathList); err != nil {
				return nil, fmt.Errorf("failed to unmarshal retrieved_file_paths: %w", err)
			}

			retrievedFilePaths = append(retrievedFilePaths, pathList...)
			// Optionally log retrieved file paths
			// log.Printf("Retrieved file paths: %v", retrievedFilePaths)
		}

		// NEW: handle updated files
		if updated, ok := chunk["updated_file_contents"]; ok {
			updatedJSON, err := json.Marshal(updated)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal updated_file_contents: %w", err)
			}
			updatedMap := map[string]UpdateFileContent{}
			if err := json.Unmarshal(updatedJSON, &updatedMap); err != nil {
				return nil, fmt.Errorf("failed to unmarshal updated_file_contents: %w", err)
			}

			// Print and collect updated file contents
			for path, updateObj := range updatedMap {
				updateContentResponse[path] = updateObj
			}
		}

		// Handle new files suggestions
		if newFilesData, ok := chunk["new_files"]; ok {
			newFilesJSON, err := json.Marshal(newFilesData)
			if err != nil {
				log.Printf("Error marshaling new_files: %v", err)
				continue
			}

			var newFiles NewFilesData
			if err := json.Unmarshal(newFilesJSON, &newFiles); err != nil {
				log.Printf("Error unmarshaling new_files: %v", err)
				continue
			}

			// ADD THIS CODE: If NewFilePaths is empty but NewContent has entries, populate NewFilePaths
			if len(newFiles.NewFilePaths) == 0 && len(newFiles.NewContent) > 0 {
				for path := range newFiles.NewContent {
					newFiles.NewFilePaths = append(newFiles.NewFilePaths, path)
				}
			}

			// Stash new files data locally
			newFilesResult = &newFiles

			// Print information about suggested new files
			if !answerOnlyMode {
				spinner.Stop()
				fmt.Printf("\nSuggested new files:\n")
				for _, path := range newFiles.NewFilePaths {
					fmt.Printf("- %s\n", path)
				}
				spinner.Start()
			}
		}
	}

	// Render any remaining content in the buffer after the stream ends
	if tokenBuffer.Len() > 0 {
		remainingContent := tokenBuffer.String()
		trimmedContent := strings.TrimRight(remainingContent, "\r\n")
		trimmedContent = strings.ReplaceAll(trimmedContent, "\r\n", "\n")
		if !answerOnlyMode {
			if err := renderMarkdown(trimmedContent); err != nil {
				log.Printf("Error rendering remaining content: %v", err)
			}
		}
		completeResponse.WriteString(trimmedContent)
	}

	// Append Retrieved File Paths to the Stream if any
	if len(retrievedFilePaths) > 0 {
		retrievedFilePathsMarkdown := "\n\n---\n\n# Retrieved File Paths\n\n"
		for _, path := range retrievedFilePaths {
			retrievedFilePathsMarkdown += fmt.Sprintf("- %s\n", path)
		}

		// Append to the complete response
		completeResponse.WriteString(retrievedFilePathsMarkdown)
		rawResponse.WriteString(retrievedFilePathsMarkdown)

		if answerOnlyMode {
			answerOnlyBuffer.WriteString(retrievedFilePathsMarkdown)
		} else {
			// Render the Markdown so it appears in the stream
			if err := renderMarkdown(retrievedFilePathsMarkdown); err != nil {
				log.Printf("Error rendering retrieved file paths: %v", err)
			}
		}
	}

	// If in answer-only mode, output the accumulated content now
	if answerOnlyMode {
		fmt.Println(answerOnlyBuffer.String())
	}

	// Before returning the result, attach the spinner
	result := &GenerateResponseResult{
		LlmModelResponse:      completeResponse.String(),
		RawResponse:           rawResponse.String(),
		RetrievedFilePaths:    retrievedFilePaths,
		UpdateContentResponse: updateContentResponse,
		HeadCommitHash:        headCommitHash,
		spinner:               spinner, // Will be nil for answer-only mode
		NewFiles:              newFilesResult,
	}

	// Write patch files for updated files - REMOVED TO FIX DUPLICATE PATCH ISSUE
	// if err := result.WritePatchToFile(); err != nil {
	//	log.Printf("Error writing patch files for updated files: %v", err)
	// }

	// ADD THIS NEW CODE: Write patch files for new files
	_, err = result.WriteNewFiles()
	if err != nil {
		log.Printf("Error writing patch files for new files: %v", err)
	}

	// Modify the defer to conditionally stop the spinner
	defer func() {
		if !answerOnlyMode && len(updateContentResponse) == 0 && (newFilesResult == nil || len(newFilesResult.NewContent) == 0) {
			spinner.Stop()
		}
	}()

	return result, nil
}

func renderMarkdown(content string) error {
	if rendererErr != nil {
		return rendererErr
	}

	// Debug: Log the content being rendered
	//log.Printf("Rendering Markdown Block: '%s'\n", content)

	out, err := renderer.Render(content)
	if err != nil {
		log.Printf("Error rendering Markdown: %v", err)
		return err
	}

	// Debug: Log the rendered output
	//log.Printf("Rendered Output: '%s'\n", out)

	// Trim trailing newline to prevent double newlines between blocks
	out = strings.TrimRight(out, "\n")

	fmt.Print(out) // Print the rendered content without adding extra newlines
	return nil
}

func getTokenCount(endpoint string, buffer *bytes.Buffer) (int, int, error) {
	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%stoken-count", endpoint), buffer)
	if err != nil {
		return 0, 0, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}

	client := &http.Client{Timeout: 60 * time.Minute} // Increased to 60 minutes
	response, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("error sending request to token count endpoint: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(response.Body)
		return 0, 0, fmt.Errorf("error getting token count: %s", body)
	}

	// Log response body for debugging
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return 0, 0, fmt.Errorf("error reading response body: %v", err)
	}

	// Decode the JSON response into the new struct
	var tokenCountResponse LoadResponse
	if err := json.Unmarshal(body, &tokenCountResponse); err != nil {
		return 0, 0, fmt.Errorf("error decoding response: %w", err)
	}

	// Return both token counts
	return tokenCountResponse.EmbeddingTokens, tokenCountResponse.InferenceTokens, nil
}

func CheckStatus(codehostURL string) (StatusResponse, error) {
	config, _, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return StatusResponse{}, err
	}

	repoManagerURL := RepoManagerURL
	if repoManagerURL == "" {
		return StatusResponse{}, fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
	}

	// Prepare the request URL
	statusURL := fmt.Sprintf("%s/status?codehost_url=%s", repoManagerURL, codehostURL)

	// Create the HTTP GET request
	req, err := http.NewRequest("GET", statusURL, nil)
	if err != nil {
		return StatusResponse{}, fmt.Errorf("error creating request: %w", err)
	}

	// Set API Gateway headers if not blank
	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	client := &http.Client{Timeout: 60 * time.Minute} // Increased to 60 minutes
	resp, err := client.Do(req)
	if err != nil {
		return StatusResponse{}, fmt.Errorf("error sending request to status endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return StatusResponse{}, fmt.Errorf("error checking status: %s", body)
	}

	var statusResponse StatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResponse); err != nil {
		return StatusResponse{}, fmt.Errorf("error decoding status response: %w", err)
	}

	return statusResponse, nil
}

func GetInstallInfo() (bool, string, error) {
	config, _, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return false, "", fmt.Errorf("error loading config: %w", err)
	}

	machtianiURL := MachtianiURL
	if machtianiURL == "" {
		return false, "", fmt.Errorf("MACHTIANI_URL environment variable is not set")
	}
	// Define the URL for the get-head-oid endpoint
	endpoint := fmt.Sprintf("%s/get-head-oid", machtianiURL) // Change this URL based on your FastAPI server configuration

	// Create a new HTTP GET request
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return false, "", fmt.Errorf("error creating request: %w", err)
	}

	// Set API Gateway headers if not blank
	if config.Environment.APIGatewayHostValue != "" {
		req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
	}
	req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

	// Create a new HTTP client with a timeout
	client := &http.Client{
		Timeout: 20 * time.Second, // Set an appropriate timeout
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("error sending request: %w", err)
	}
	defer resp.Body.Close()

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return false, "", fmt.Errorf("error: received status code %d from the server: %s", resp.StatusCode, body)
	}

	// Decode the response body
	var response map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return false, "", fmt.Errorf("error decoding response: %w", err)
	}

	// Compare the returned head_oid with HeadOID
	returnedHeadOID, ok := response["head_oid"]
	if !ok {
		return false, "", fmt.Errorf("response does not contain head_oid")
	}
	message, ok := response["message"]
	if !ok {
		return false, "", fmt.Errorf("response does not contain message")
	}

	return returnedHeadOID == HeadOID, message, nil
}

func (res *GenerateResponseResult) WritePatchToFile() error {

	if len(res.UpdateContentResponse) == 0 {
		if res.spinner != nil {
			res.spinner.Stop()
		}
		return nil
	}

	// Ensure the spinner stops running during file writes
	if res.spinner != nil {
		res.spinner.Stop()
	}

	var outputBuffer bytes.Buffer // Use a buffer to collect messages

	// Add a newline before the patch messages start for clear separation
	outputBuffer.WriteString("\n")

	// Get current timestamp for unique filenames
	timestamp := time.Now().Format("20060102_150405") // YYYYMMDD_HHMMSS format

	for filename, update := range res.UpdateContentResponse {
		skip := false
		// Check if there are any non-empty error messages
		for _, errMsg := range update.Errors {
			if len(strings.TrimSpace(errMsg)) > 0 {
				log.Printf("Error received for file %s: %s", filename, errMsg) // Log the specific error
				skip = true
				break
			}
		}
		if skip {
			// Append message to buffer instead of printing directly
			outputBuffer.WriteString(fmt.Sprintf("Skipping patch creation for %s due to errors during generation.\n", filename))
			continue
		}

		// If UpdatedContent is empty, skip writing the patch file
		if len(strings.TrimSpace(update.UpdatedContent)) == 0 {
			outputBuffer.WriteString(fmt.Sprintf("Skipping patch creation for %s as updated content is empty.\n", filename))
			continue
		}

		// Sanitize filename for patch file
		safeFilename := strings.ReplaceAll(filename, "/", "_")
		safeFilename = strings.ReplaceAll(safeFilename, ":", "_") // Add more sanitization if needed

		// Include timestamp in the filename for uniqueness
		patchFileName := fmt.Sprintf("%s_%s.patch", safeFilename, timestamp)

		// Ensure the .machtiani/patches directory exists
		patchesDir := ".machtiani/patches"
		if err := utils.EnsureDirExists(patchesDir); err != nil {
			return fmt.Errorf("failed to ensure patches directory exists: %w", err)
		}
		fullPatchPath := fmt.Sprintf("%s/%s", patchesDir, patchFileName)

		err := ioutil.WriteFile(fullPatchPath, []byte(update.UpdatedContent), 0644)
		if err != nil {
			// Log the error but collect a message for the user
			log.Printf("Failed to write patch to file %s: %v", fullPatchPath, err)
			outputBuffer.WriteString(fmt.Sprintf("Error writing patch for %s to %s\n", filename, fullPatchPath))
			// Decide if you want to return the error immediately or just report it
			// return fmt.Errorf("failed to write patch to file %s: %w", fullPatchPath, err) // Option: Stop on first error
		} else {
			// Append success message to buffer instead of printing directly
			outputBuffer.WriteString(fmt.Sprintf("Wrote patch for %s to %s\n", filename, fullPatchPath))
		}
	}

	// Print the collected messages all at once after the loop
	// Only print if the buffer actually contains messages (beyond the initial newline)
	if outputBuffer.Len() > 1 {
		fmt.Print(outputBuffer.String())
	}

	return nil
}

type SpinnerController struct {
	done     chan bool
	spinning bool
	mutex    sync.Mutex
}

func NewSpinnerController() *SpinnerController {
	return &SpinnerController{
		done: make(chan bool),
	}
}

func (s *SpinnerController) Start() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if !s.spinning {
		s.spinning = true
		s.done = make(chan bool)
		go utils.Spinner(s.done)
	}
}

func (s *SpinnerController) Stop() {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.spinning {
		s.done <- true
		s.spinning = false
		fmt.Print("\r ") // Clear the spinner line
	}
}

// WriteNewFiles creates new files with provided content if they don't already exist
func (res *GenerateResponseResult) WriteNewFiles() (string, error) {
	var outputBuffer bytes.Buffer
	var filesWritten int
	var filesSkipped int

	if res.NewFiles == nil {
		outputBuffer.WriteString("No new files data available - skipping\n")
		return outputBuffer.String(), nil
	}

	if len(res.NewFiles.NewContent) == 0 {
		outputBuffer.WriteString("New files content is empty - nothing to write\n")
		return outputBuffer.String(), nil
	}

	// Ensure spinner is stopped if it exists (might be nil in answer-only mode)
	if res.spinner != nil {
		res.spinner.Stop()
	}

	outputBuffer.WriteString(fmt.Sprintf("Creating new files for %d entries\n", len(res.NewFiles.NewContent)))

	for path, content := range res.NewFiles.NewContent {
		if strings.TrimSpace(content) == "" {
			outputBuffer.WriteString(fmt.Sprintf("- %s (skipped - empty content)\n", path))
			filesSkipped++
			continue
		}

		// Check if the file exists in the filesystem
		if _, err := os.Stat(path); err == nil {
			outputBuffer.WriteString(fmt.Sprintf("- %s (skipped - file already exists)\n", path))
			filesSkipped++
			continue
		} else if !os.IsNotExist(err) {
			// Some other error checking the file
			log.Printf("Error checking file %s: %v", path, err)
			outputBuffer.WriteString(fmt.Sprintf("- %s (error checking existence: %v)\n", path, err))
			continue
		}

		// Create parent directories
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Error creating directories for %s: %v", path, err)
			outputBuffer.WriteString(fmt.Sprintf("- %s (error creating directories: %v)\n", path, err))
			continue
		}

		// Write the file
		if err := ioutil.WriteFile(path, []byte(content), 0644); err != nil {
			log.Printf("Error writing file %s: %v", path, err)
			outputBuffer.WriteString(fmt.Sprintf("- %s (error writing file: %v)\n", path, err))
			continue
		}

		outputBuffer.WriteString(fmt.Sprintf("- Created %s\n", path))
		filesWritten++
	}

	outputBuffer.WriteString(fmt.Sprintf("\nNew files processing complete - %d written, %d skipped\n", filesWritten, filesSkipped))
	return outputBuffer.String(), nil
}
