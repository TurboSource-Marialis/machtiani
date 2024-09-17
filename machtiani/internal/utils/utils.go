package utils

import (
    "fmt"
    "io/ioutil"
    "os"
)

func CreateTempMarkdownFile(content string, filename string) (string, error) {
    // Define the directory where files will be saved
    chatDir := ".machtiani/chat"

    // Check if the directory exists, create if it doesn't
    if _, err := os.Stat(chatDir); os.IsNotExist(err) {
        if err := os.MkdirAll(chatDir, 0755); err != nil {
            return "", fmt.Errorf("failed to create directory: %v", err)
        }
    }

    // Create a markdown file with the provided filename
    tempFile := fmt.Sprintf("%s/%s.md", chatDir, filename)
    if err := ioutil.WriteFile(tempFile, []byte(content), 0644); err != nil {
        return "", err
    }

    return tempFile, nil
}

var dryRun bool

// SetDryRun sets the dry-run state.
func SetDryRun(state bool) {
    dryRun = state
}

// IsDryRunEnabled returns true if dry-run mode is enabled.
func IsDryRunEnabled() bool {
    return dryRun
}

