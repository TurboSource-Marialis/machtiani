package api

import (
    "encoding/json"
    "fmt"
    "bytes"
    "net/http"
)

func CallOpenAIAPI(apiKey, prompt, project, mode, model, matchStrength string, embeddings []float64) (map[string]interface{}, error) {
    // Construct the request payload
    payload := map[string]interface{}{
        "prompt":         prompt,
        "project":        project,
        "mode":           mode,
        "model":          model,
        "api_key":        apiKey,
        "match_strength": matchStrength,
        "embeddings":     embeddings,
    }

    // Convert the payload to JSON
    payloadBytes, err := json.Marshal(payload)
    if err != nil {
        return nil, fmt.Errorf("failed to marshal JSON: %w", err)
    }

    // Make the POST request
    resp, err := http.Post("http://localhost:5071/generate-response", "application/json", bytes.NewBuffer(payloadBytes))
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

