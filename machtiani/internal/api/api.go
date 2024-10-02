package api

import (
    "strings"
    "encoding/json"
    "fmt"
    "bytes"
    "log"
    "net/http"
    "io/ioutil"

    "github.com/7db9a/machtiani/internal/utils"
)

type AddRepositoryResponse struct {
    Message        string `json:"message"`
    FullPath       string `json:"full_path"`
    ApiKeyProvided bool   `json:"api_key_provided"`
    OpenAiApiKeyProvided bool   `json:"openai_api_key_provided"`
}



func AddRepository(codeURL string, name string, apiKey *string, openAIAPIKey string, repoManagerURL string) (AddRepositoryResponse, error) {
	ignoreFilePath := ".machtiani.ignore"
	ignoreFiles, err := utils.ReadIgnoreFile(ignoreFilePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print the file paths
	fmt.Println("Parsed file paths from machtiani.ignore:")
	for _, path := range ignoreFiles {
		fmt.Println(path)
	}
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }
    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url":   codeURL,
        "project_name":   name,
        "vcs_type":       "git",  // Default value for VCS type
        "api_key":        apiKey, // This can be nil
        "model_api_key": openAIAPIKey, // This can also be nil
        "ignore_files": ignoreFiles,
    }

    // Convert data to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return AddRepositoryResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
    }

    tokenCount, err := getTokenCount(fmt.Sprintf("%s/add-repository/", repoManagerURL), bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error getting token count: %v\n", err)
        // Return zero value and error
        return AddRepositoryResponse{}, err
    }
    fmt.Printf("Estimated input tokens: %d\n", tokenCount)

    // Check if the user wants to proceed
    if confirmProceed() {
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

        client := &http.Client{}
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
            OpenAiApiKeyProvided: false,
        }
        return abortedResponse, nil
    }
}

// FetchAndCheckoutBranch sends a request to fetch and checkout a branch.
func FetchAndCheckoutBranch(codeURL string, name string, branchName string, apiKey *string, openAIAPIKey string) (string, error) {
	ignoreFilePath := ".machtiani.ignore"
	ignoreFiles, err := utils.ReadIgnoreFile(ignoreFilePath)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Print the file paths
	fmt.Println("Parsed file paths from machtiani.ignore:")
	for _, path := range ignoreFiles {
		fmt.Println(path)
	}
    config, err := utils.LoadConfig()
    if err != nil {
        return "", fmt.Errorf("error loading config: %w", err)
    }

    data := map[string]interface{}{
        "codehost_url":    codeURL,
        "project_name":    name,
        "branch_name":     branchName,
        "api_key":         apiKey,
        "model_api_key": openAIAPIKey,
        "ignore_files": ignoreFiles,
    }

    jsonData, err := json.Marshal(data)
    if err != nil {
        return "", fmt.Errorf("error marshaling JSON: %w", err)
    }

    repoManagerURL := config.Environment.RepoManagerURL
    if repoManagerURL == "" {
        return "", fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
    }

    tokenCount, err := getTokenCount(fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(jsonData))
    if err != nil {
        fmt.Printf("Error getting token count: %v\n", err)
        return "", err
    }
    fmt.Printf("Estimated input tokens: %d\n", tokenCount)

    if confirmProceed() {
        req, err := http.NewRequest("POST", fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(jsonData))
        if err != nil {
            return "", fmt.Errorf("error creating request: %w", err)
        }

        // Set API Gateway headers if not blank
        if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
            req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
        }
        req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

        client := &http.Client{}
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

        return fmt.Sprintf("Successfully synced the repository: %s.\nServer response: %s", name, string(body)), nil
    } else {
        return "Operation aborted by user", nil
    }
}

func CallOpenAIAPI(prompt, project, mode, model, matchStrength string) (map[string]interface{}, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    payload := map[string]interface{}{
        "prompt":         prompt,
        "project":        project,
        "mode":           mode,
        "model":          model,
        "match_strength": matchStrength,
        "api_key": config.Environment.ModelAPIKey,
    }

    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }

    endpoint := config.Environment.MachtianiURL
    if endpoint == "" {
        return nil, fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }

    repoManagerURL := config.Environment.RepoManagerURL

    // Get token count
    tokenCount, err := getTokenCount(fmt.Sprintf("%s/generate-response/", repoManagerURL), bytes.NewBuffer(payloadBytes))

    req, err := http.NewRequest("POST", fmt.Sprintf("%s/generate-response", endpoint), bytes.NewBuffer(payloadBytes))

    if err != nil {
        fmt.Printf("Error getting token count: %v\n", err)
        return nil, err
    }

    fmt.Printf("Estimated input tokens: %d\n", tokenCount)

    // Step 2: Confirm to proceed
    if !confirmProceed() {
        return nil, fmt.Errorf("operation aborted by user")
    }

    // Set API Gateway headers if not blank
    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("failed to make API request: %w", err)
    }
    defer resp.Body.Close()

    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode JSON response: %w", err)
    }

    return result, nil
}

// getTokenCount calls the /load/token-count endpoint to get the token count
func getTokenCount(endpoint string, buffer *bytes.Buffer) (int, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }
    // Create a new request instead of using http.Post
    req, err := http.NewRequest("POST", fmt.Sprintf("%stoken-count", endpoint), buffer)
    if err != nil {
        return 0, fmt.Errorf("error creating request: %w", err)
    }

    // Set Content-Type header
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    // Optionally set the RapidAPI headers if configured
    if config.Environment.APIGatewayHostKey != "" && config.Environment.APIGatewayHostValue != "" {
        req.Header.Set(config.Environment.APIGatewayHostKey, config.Environment.APIGatewayHostValue)
    }
    req.Header.Set(config.Environment.ContentTypeKey, config.Environment.ContentTypeValue)

    // Create a new HTTP client and send the request
    client := &http.Client{}
    response, err := client.Do(req)
    if err != nil {
        return 0, fmt.Errorf("error sending request to token count endpoint: %w", err)
    }
    defer response.Body.Close()

    // Check the status code of the response
    if response.StatusCode != http.StatusOK {
        body, _ := ioutil.ReadAll(response.Body)
        return 0, fmt.Errorf("error getting token count: %s", body)
    }

    // Decode the JSON response into a map
    var result map[string]int
    if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
        return 0, fmt.Errorf("error decoding response: %w", err)
    }

    // Return the token count from the result map
    return result["token_count"], nil
}

// confirmProceed prompts the user for confirmation to proceed
func confirmProceed() bool {
    var response string
    fmt.Print("Do you wish to proceed? (y/n): ")
    fmt.Scanln(&response)
    return strings.ToLower(response) == "y"
}
