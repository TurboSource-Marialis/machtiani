package git

import (
    "os/exec"
    "strings"
    "fmt"
)

func GetProjectName() (string, error) {
    // Run the git command to get the remote URL
    cmd := exec.Command("git", "config", "--get", "remote.origin.url")
    output, err := cmd.Output()
    if err != nil {
        return "", err
    }

    // Parse the remote URL
    url := string(output)
    // Extract the project name (assuming it's the last part of the URL before .git)
    parts := strings.Split(strings.TrimSpace(url), "/")
    projectName := strings.TrimSuffix(parts[len(parts)-1], ".git")

    return projectName, nil
}

func GetRemoteURL(remoteName *string) (string, error) {
    remoteURL, err := getRemoteURL(*remoteName)
    if remoteName == nil || *remoteName == "" {
        return "", fmt.Errorf("remote name cannot be empty")
    }
    if err != nil {
        return "", fmt.Errorf("Error fetching remote URL: %v", err)
    }
    return remoteURL, nil
}

func getRemoteURL(remoteName string) (string, error) {
    cmd := exec.Command("git", "remote", "get-url", remoteName)
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get remote URL for %s: %w", remoteName, err)
    }
    return strings.TrimSpace(string(output)), nil
}
