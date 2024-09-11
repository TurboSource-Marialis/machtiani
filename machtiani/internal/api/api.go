package api

import (
    "encoding/json"
    "fmt"
    "net/http"
    "net/url"
    "io/ioutil"
)

func CallOpenAIAPI(apiKey, prompt, project, mode, model, matchStrength string) (map[string]interface{}, error) {
    encodedPrompt := url.QueryEscape(prompt)
    apiURL := fmt.Sprintf("http://localhost:5071/generate-response?prompt=%s&project=%s&mode=%s&model=%s&api_key=%s&match_strength=%s",
        encodedPrompt, project, mode, model, apiKey, matchStrength)

    resp, err := http.Post(apiURL, "application/json", nil)
    if err != nil {
        return nil, err
    }
    defer resp.Body.Close()

    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
        return nil, err
    }

    var response map[string]interface{}
    if err := json.Unmarshal(body, &response); err != nil {
        return nil, err
    }
    return response, nil
}
