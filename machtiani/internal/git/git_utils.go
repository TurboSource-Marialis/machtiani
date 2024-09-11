package git

import (
    "os/exec"
    "strings"
)

func getProjectName() (string, error) {
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
