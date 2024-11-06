package api

import (
    "strings"
    "encoding/json"
    "fmt"
    "bytes"
    "log"
    "net/http"
    "io/ioutil"
    "time"

    "github.com/7db9a/machtiani/internal/utils"
)

type AddRepositoryResponse struct {
    Message        string `json:"message"`
    FullPath       string `json:"full_path"`
    ApiKeyProvided bool   `json:"api_key_provided"`
    OpenAiApiKeyProvided bool   `json:"openai_api_key_provided"`
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


func AddRepository(codeURL string, name string, apiKey *string, openAIAPIKey string, repoManagerURL string, force bool) (AddRepositoryResponse, error) {
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
        "model_api_key":  openAIAPIKey,
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
        // Start the spinner
        done := make(chan bool)
        go utils.Spinner(done)

        // Proceed with sending the POST request
        req, err := http.NewRequest("POST", fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(jsonData))
        if err != nil {
            return AddRepositoryResponse{}, fmt.Errorf("error creating request: %w", err)
        }

        // Set API Gateway headers if not blank
        if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

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
        // Stop the spinner
        done <- true

        // Clear the spinner on completion
        fmt.Print("\r ") // Clear the spinner output

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
            OpenAiApiKeyProvided: false,
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

    // Print the file paths
    fmt.Println("Parsed file paths from machtiani.ignore:")
    for _, path := range ignoreFiles {
        fmt.Println(path)
    }

    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url":   codeURL,
        "project_name":   name,
        "branch_name":    branchName,
        "api_key":       apiKey,
        "model_api_key": openAIAPIKey,
        "ignore_files":  ignoreFiles,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("error marshaling JSON: %w", err)
    }

    repoManagerURL := config.Environment.RepoManagerURL
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
        if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

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

func DeleteStore(projectName string, codehostURL string, ignoreFiles []string, vcsType string, apiKey *string, openaiAPIKey *string, repoManagerURL string, force bool) (DeleteStoreResponse, error) {
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
            "ignore_files":    ignoreFiles,
            "vcs_type":       vcsType,
            "api_key":        apiKey,
            "openai_api_key": openaiAPIKey,
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
        if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

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

func CallOpenAIAPI(prompt, project, mode, model, matchStrength string, force bool) (map[string]interface{}, error) {

    config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    // Print the file paths
    fmt.Println("Parsed file paths from machtiani.ignore:")
    for _, path := range ignoreFiles {
        fmt.Println(path)
    }

    // Retrieve the codehost URL based on the current Git project.
    codehostURL, err := utils.GetCodehostURLFromCurrentRepository()
    if err != nil {
        return nil, fmt.Errorf("failed to get codehost URL: %w", err)
    }

    payload := map[string]interface{}{
        "prompt":          prompt,
        "project":         project,
        "mode":            mode,
        "model":           model,
        "match_strength":  matchStrength,
        "api_key":        config.Environment.ModelAPIKey,
        "codehost_api_key": config.Environment.CodeHostAPIKey,
        "codehost_url":   codehostURL, // Include the codehost_url here
        "ignore_files": ignoreFiles,
    }

    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }

    endpoint := config.Environment.MachtianiURL
    if endpoint == "" {
        return nil, fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/generate-response", endpoint), bytes.NewBuffer(payloadBytes))
    if err != nil {
        return nil, fmt.Errorf("failed to create request: %w", err)
    }

    // Set API Gateway headers if not blank
    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    // Create a new HTTP client with a timeout
    client := &http.Client{
        Timeout: 20 * time.Minute,
    }

    // Start the spinner (if needed)
    done := make(chan bool)
    go utils.Spinner(done)

    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to make API request: %w", err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode JSON response: %w", err)
    }

    // Stop the spinner
    done <- true

    // Clear the spinner on completion
    fmt.Print("\r ") // Clear the spinner output

    return result, nil
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

    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
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

func CheckStatus(codehostURL string, apiKey *string) (StatusResponse, error) {
    config, _, err := utils.LoadConfigAndIgnoreFiles()
    if err != nil {
        return StatusResponse{}, err
    }

    repoManagerURL := config.Environment.RepoManagerURL
    if repoManagerURL == "" {
        return StatusResponse{}, fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
    }

    // Prepare the request URL
    statusURL := fmt.Sprintf("%s/status?codehost_url=%s", repoManagerURL, codehostURL)
    if apiKey != nil {
        statusURL += fmt.Sprintf("&api_key=%s", *apiKey)
    }

    // Create the HTTP GET request
    req, err := http.NewRequest("GET", statusURL, nil)
    if err != nil {
        return StatusResponse{}, fmt.Errorf("error creating request: %w", err)
    }

    // Set API Gateway headers if not blank
    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

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

// confirmProceed prompts the user for confirmation to proceed
func confirmProceed() bool {
    var response string
    fmt.Print("Do you wish to proceed? (y/n): ")
    fmt.Scanln(&response)
    return strings.ToLower(response) == "y"
}
