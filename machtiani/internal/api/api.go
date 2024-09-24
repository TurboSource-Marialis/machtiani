package api

import (
    "encoding/json"
    "fmt"
    "bytes"
    "net/http"
    "os"
    "io/ioutil"
)

type AddRepositoryResponse struct {
    Message        string `json:"message"`
    FullPath       string `json:"full_path"`
    ApiKeyProvided bool   `json:"api_key_provided"`
}

// AddRepository sends a request to add a repository.
func AddRepository(codeURL, name, apiKey string) (AddRepositoryResponse, error) {
    // Prepare the data to be sent in the request
    data := map[string]interface{}{
        "codehost_url": codeURL,
        "project_name": name,
        "vcs_type":     "git",  // Default value for VCS type
        "api_key":      apiKey,
    }

    // Convert data to JSON
    jsonData, err := json.Marshal(data)
    if err != nil {
        return AddRepositoryResponse{}, fmt.Errorf("error marshaling JSON: %w", err)
    }

    // Get the base URL from the environment variable
    repoManagerURL := os.Getenv("MACHTIANI_REPO_MANAGER_URL")
    if repoManagerURL == "" {
        return AddRepositoryResponse{}, fmt.Errorf("MACHTIANI_REPO_MANAGER_URL environment variable is not set")
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

func CallOpenAIAPI(prompt, project, mode, model, matchStrength string, embeddings []float64) (map[string]interface{}, error) {
    // Construct the request payload
    payload := map[string]interface{}{
        "prompt":         prompt,
        "project":        project,
        "mode":           mode,
        "model":          model,
        "match_strength": matchStrength,
    }

    // Only add embeddings to the payload if they are provided
    if len(embeddings) > 0 {
        payload["embeddings"] = embeddings
    }
    // Convert the payload to JSON
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }

    // Retrieve the MACHTIANI_URL from environment variables
    endpoint := os.Getenv("MACHTIANI_URL")
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

