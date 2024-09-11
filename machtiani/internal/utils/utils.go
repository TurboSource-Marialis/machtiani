package utils

import (
    "fmt"
    "os"
)

func CreateTempMarkdownFile(content string) (string, error) {
    tempDir, err := ioutil.TempDir("", "response")
    if err != nil {
        return "", err
    }

    tempFile := fmt.Sprintf("%s/response.md", tempDir)
    if err := ioutil.WriteFile(tempFile, []byte(content), 0644); err != nil {
        return "", err
    }

    return tempFile, nil
}
