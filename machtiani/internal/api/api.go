package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/7db9a/machtiani/internal/utils"
	"github.com/charmbracelet/glamour"
)
var (
    HeadOID    string = "none"
    BuildDate string = "unknown"
    MachtianiURL string = "http://localhost:5071"
    RepoManagerURL string = "http://localhost:5070"
)

var (
    renderer     *glamour.TermRenderer
    rendererOnce sync.Once
    rendererErr  error
)

const (
    CONTENT_TYPE_KEY   = "Content-Type"
    CONTENT_TYPE_VALUE = "application/json"
    API_GATEWAY_HOST_KEY = "X-RapidAPI-Key"
)

type AddRepositoryResponse struct {
    Message        string `json:"message"`
    FullPath       string `json:"full_path"`
    ApiKeyProvided bool   `json:"api_key_provided"`
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
    LockFilePresent  bool   `json:"lock_file_present"`
    LockTimeDuration float64 `json:"lock_time_duration"` // New field added
}


func AddRepository(codeURL string, name string, apiKey *string, openAIAPIKey string, repoManagerURL string, llmModelBaseURL string, force bool) (AddRepositoryResponse, error) {
    config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        return AddRepositoryResponse{}, err
    }

    fmt.Println() // Prints a new line
    fmt.Println("Ignoring files based on .machtiani.ignore:")
    if len(ignoreFiles) == 0 {
        fmt.Println("No files to ignore.")
    } else {
        fmt.Println() // Prints another new line
        for _, path := range ignoreFiles {
            fmt.Println(path)
        }
    }

    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url":   codeURL,
        "project_name":   name,
        "vcs_type":       "git",
        "api_key":        apiKey,
        "llm_model_api_key":  openAIAPIKey,
        "llm_model_base_url": llmModelBaseURL,
        "ignore_files":   ignoreFiles,
    }

    // Convert data to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return AddRepositoryResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
    }

    tokenCountEmbedding, tokenCountInference, err := getTokenCount(fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error getting token count: %v\n", err)
        return AddRepositoryResponse{}, err
    }

    // Print the token counts separately
    fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
    fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

    // Check if the user wants to proceed
    // Check if the user wants to proceed or if force is enabled
    if force || confirmProceed() {

        // Proceed with sending the POST request
        req, err := http.NewRequest("POST", fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(jsonData))
        if err != nil {
            return AddRepositoryResponse{}, fmt.Errorf("error creating request: %w", err)
        }

        // Set API Gateway headers if not blank
        if config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

        client := &http.Client{
            Timeout: 20 * time.Minute,
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
    } else {
        // User chose not to proceed, return an AddRepositoryResponse with fields indicating operation aborted
        abortedResponse := AddRepositoryResponse{
            Message:              "Operation aborted by user",
            FullPath:             "Operation aborted",
            ApiKeyProvided:       false,
            LlmModelApiKeyProvided: false,
        }
        return abortedResponse, nil
    }
}

// FetchAndCheckoutBranch sends a request to fetch and checkout a branch.
func FetchAndCheckoutBranch(codeURL string, name string, branchName string, apiKey *string, openAIAPIKey string, force bool) (string, error) {
    config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        return "", err
    }

    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url":   codeURL,
        "project_name":   name,
        "branch_name":    branchName,
        "api_key":       apiKey,
        "llm_model_api_key": openAIAPIKey,
        "llm_model_base_url": config.Environment.ModelBaseURL,
        "ignore_files":  ignoreFiles,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("error marshaling JSON: %w", err)
    }

    repoManagerURL := RepoManagerURL
    if repoManagerURL == "" {
        return "", fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
    }

    tokenCountEmbedding, tokenCountInference , err := getTokenCount(fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error getting token count: %v\n", err)
        return "", err
    }

    // Print the token counts separately
    fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
    fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

    // Check if the user wants to proceed or if force is enabled
    if force || confirmProceed() {
        // Start the spinner
        done := make(chan bool)
        go utils.Spinner(done)

        req, err := http.NewRequest("POST", fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(jsonData))
        if err != nil {
            return "", fmt.Errorf("error creating request: %w", err)
        }

        // Set API Gateway headers if not blank
        if config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(API_GATEWAY_HOST_KEY, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(CONTENT_TYPE_KEY, CONTENT_TYPE_VALUE)

        client := &http.Client{
            Timeout: 20 * time.Minute,
        }
        resp, err := client.Do(req)
        if err != nil {
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

        // Stop the spinner
        done <- true

        // Clear the spinner on completion
        fmt.Print("\r ") // Clear the spinner output

        return fmt.Sprintf("Successfully synced the repository: %s.\nServer response: %s", name, string(body)), nil
    } else {
        return "Operation aborted by user", nil
    }
}

func DeleteStore(projectName string, codehostURL string, vcsType string, apiKey *string, repoManagerURL string, force bool) (DeleteStoreResponse, error) {
    config, _, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        return DeleteStoreResponse{}, err
    }

    if force || confirmProceed() {
        done := make(chan bool)
        go utils.Spinner(done)

        // Prepare the data to be sent in the request
        data := map[string]interface{}{
            "project_name":   projectName,
            "codehost_url":   codehostURL,
            "vcs_type":       vcsType,
            "api_key":        apiKey,
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
            Timeout: 20 * time.Minute,
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

type GenerateResponseResult struct {
    LlmModelResponse     string   `json:"llm_model_response"`
    RawResponse     string   `json:"llm_model_response"`
    RetrievedFilePaths []string `json:"retrieved_file_paths"`
}

func init() {
    rendererOnce.Do(func() {
        renderer, rendererErr = glamour.NewTermRenderer(
            glamour.WithAutoStyle(),
            glamour.WithWordWrap(120),
        )
        if rendererErr != nil {
            log.Fatalf("Error creating renderer: %v", rendererErr)
        }
    })
}

func GenerateResponse(prompt, project, mode, model, matchStrength string, force bool) (*GenerateResponseResult, error) {
    config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    codehostURL, err := utils.GetCodehostURLFromCurrentRepository()
    if err != nil {
        return nil, fmt.Errorf("failed to get codehost URL: %w", err)
    }

    payload := map[string]interface{}{
        "prompt":             prompt,
        "project":            project,
        "mode":               mode,
        "model":              model,
        "match_strength":     matchStrength,
        "llm_model_api_key":  config.Environment.ModelAPIKey,
        "llm_model_base_url": config.Environment.ModelBaseURL,
        "codehost_api_key":   config.Environment.CodeHostAPIKey,
        "codehost_url":       codehostURL,
        "ignore_files":       ignoreFiles,
        "llm_model_base_url_other": config.Environment.ModelBaseURLOther,
    }

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
        Timeout: 20 * time.Minute,
    }

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to make API request: %w", err)
    }
    defer resp.Body.Close()

    // Initialize variables
    var completeResponse strings.Builder
    var rawResponse strings.Builder         // Initialize rawResponse
    var retrievedFilePaths []string
    var tokenBuffer bytes.Buffer // Buffer to accumulate tokens
    var inCodeBlock bool        // Track if we're inside a code block
    var codeBlockBuffer bytes.Buffer // Buffer to accumulate code block content

    // Use a JSON decoder to read multiple JSON objects from the response stream
    decoder := json.NewDecoder(resp.Body)

    // Check if the prompt already starts with the User header
    var header string
    if strings.HasPrefix(strings.TrimSpace(prompt), "# User") {
        // If the prompt already contains the User header, use it as is
        header = fmt.Sprintf("%s\n# Assistant\n\n", prompt)
    } else {
        // Otherwise, prepend the User header
        header = fmt.Sprintf("# User\n\n%s\n\n# Assistant\n\n", prompt)
    }

    if err := renderMarkdown(header); err != nil {
        return nil, fmt.Errorf("failed to render header: %w", err)
    }
    completeResponse.WriteString(header)
    rawResponse.WriteString(header)

    // Initialize SpinnerController
    spinner := NewSpinnerController()

    // Start the initial spinner
    spinner.Start()

    // Ensure the spinner is stopped when the function exits
    defer spinner.Stop()

    for {
        var chunk map[string]interface{}
        if err := decoder.Decode(&chunk); err == io.EOF {
            // End of response stream
            break
        } else if err != nil {
            return nil, fmt.Errorf("failed to decode JSON response: %w", err)
        }

        // Handle error messages
        if errMsg, ok := chunk["error"].(string); ok {
            return nil, fmt.Errorf("API error: %s", errMsg)
        }

        // Handle tokens from OpenAI response
        if token, ok := chunk["token"].(string); ok {
            tokenBuffer.WriteString(token)
            rawResponse.WriteString(token) // Append to rawResponse

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
                            // Optionally, you can choose to return the error instead of continuing
                        }

                        // Append to complete response
                        completeResponse.WriteString(trimmedBlock)
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
    }

    // Render any remaining content in the buffer after the stream ends
    if tokenBuffer.Len() > 0 {
        remainingContent := tokenBuffer.String()
        trimmedContent := strings.TrimRight(remainingContent, "\r\n")
        trimmedContent = strings.ReplaceAll(trimmedContent, "\r\n", "\n")
        if err := renderMarkdown(trimmedContent); err != nil {
            log.Printf("Error rendering remaining content: %v", err)
            // Optionally, you can choose to return the error or continue
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

        // Render the Markdown so it appears in the stream
        if err := renderMarkdown(retrievedFilePathsMarkdown); err != nil {
            log.Printf("Error rendering retrieved file paths: %v", err)
            // You can choose to handle the error differently if needed
        }
    }

    return &GenerateResponseResult{
        LlmModelResponse:      completeResponse.String(),
        RawResponse:         rawResponse.String(), // Include rawResponse
        RetrievedFilePaths: retrievedFilePaths,
    }, nil
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

    client := &http.Client{Timeout: 20 * time.Minute}
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

    client := &http.Client{Timeout: 20 * time.Minute}
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

// confirmProceed prompts the user for confirmation to proceed
func confirmProceed() bool {
    var response string
    fmt.Print("Do you wish to proceed? (y/n): ")
    fmt.Scanln(&response)
    return strings.ToLower(response) == "y"
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
    }
}

