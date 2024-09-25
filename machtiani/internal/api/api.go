package api

import (
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

// AddRepository sends a request to add a repository.
func AddRepository(codeURL, name, apiKey, openAIAPIKey string, repoManagerURL string) (AddRepositoryResponse, error) {
    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url": codeURL,
        "project_name": name, "vcs_type":     "git",  // Default value for VCS type
        "api_key":      apiKey,
        "openai_api_key": openAIAPIKey, // Add the OpenAI API key here
    }

    // Convert data to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return AddRepositoryResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
    }

    // Send the POST request to the specified endpoint
    resp, err := http.Post(fmt.Sprintf("%s/add-repository/", repoManagerURL), "application/json", bytes.NewBuffer(jsonData))
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
func FetchAndCheckoutBranch(codeURL, name, branchName, apiKey, openAIAPIKey string) error {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }
    // Prepare the data for the request
    data := map[string]interface{}{
        "codehost_url": codeURL,
        "project_name": name,
        "branch_name":  branchName,
        "api_key":     apiKey,
        "openai_api_key": openAIAPIKey, // Add the OpenAI API key here
    }

    // Convert data to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return fmt.Errorf("error marshaling JSON: %w", err)
    }

    repoManagerURL := config.Environment.RepoManagerURL
    if repoManagerURL == "" {
        return fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
    }

    // Create the POST request
    req, err := http.NewRequest("POST", fmt.Sprintf("%s/fetch-and-checkout/", repoManagerURL), bytes.NewBuffer(jsonData))
    if err != nil {
        return fmt.Errorf("error creating request: %w", err)
    }
    req.Header.Set("Content-Type", "application/json")

    // Execute the request
    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        return fmt.Errorf("error making request: %w", err)
    }
    defer resp.Body.Close()

    // Check the response status
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("error: received status code %d from the server.", resp.StatusCode)
    }

    return nil
}

func CallOpenAIAPI(prompt, project, mode, model, matchStrength string) (map[string]interface{}, error) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    // Construct the request payload
    payload := map[string]interface{}{
        "prompt":         prompt,
        "project":        project,
        "mode":           mode,
        "model":          model,
        "match_strength": matchStrength,
    }

    // Convert the payload to JSON
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }

    // Retrieve the MACHTIANI_URL from environment variables
    endpoint := config.Environment.MachtianiURL
    if endpoint == "" {
        return nil, fmt.Errorf("MACHTIANI_URL environment variable is not set")
    }

    // Make the POST request
    resp, err := http.Post(fmt.Sprintf("%s/generate-response", endpoint), "application/json", bytes.NewBuffer(payloadBytes))
    if err != nil {
        return nil, fmt.Errorf("failed to make API request: %w", err)
    }
    defer resp.Body.Close()

    // Handle the response
    var result map[string]interface{}
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, fmt.Errorf("failed to decode JSON response: %w", err)
    }

    return result, nil
}

