package api

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "net/http"
    "net/url"
)

func CallOpenAIAPI(apiKey, prompt, project, mode, model, matchStrength string) (map[string]interface{}, error) {
    // URL encode the prompt to safely include it in the URL
    encodedPrompt := url.QueryEscape(prompt)

    // Construct the API URL with the provided parameters
    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=%s&model=%s&api_key=%s&match_strength=%s",
        encodedPrompt, project, mode, model, apiKey, matchStrength)

    // Make the POST request
    resp, err := http.Post(apiURL, "application/json", nil)
    if err != nil {
        return nil, fmt.Errorf("failed to make request: %v", err)
    }
    defer resp.Body.Close()

    // Read the response body
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("failed to read response body: %v", err)
    }

    // Check if the response status is not OK (200)
    if resp.StatusCode != http.StatusOK {
        var errorResponse map[string]interface{}
        if err := json.Unmarshal(body, &errorResponse); err != nil {
            return nil, fmt.Errorf("failed to parse error response: %v", err)
        }
        return nil, fmt.Errorf("API error: %v", errorResponse["detail"])
    }

    // Parse the successful response
    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, fmt.Errorf("failed to decode response: %v", err)
    }

    return response, nil
}
