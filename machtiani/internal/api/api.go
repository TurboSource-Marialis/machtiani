package api

import (
    "encoding/json"
    "fmt"
    "bytes"
    "net/http"
    "os"
)

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

