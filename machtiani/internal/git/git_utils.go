package git

import (
	"bytes"
	"errors"
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

func GetBranch() (string, error) {
    // Run the git command to get the current branch name
    // FYI it returns 'HEAD' if not in a branch, if detached.
    cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
    output, err := cmd.Output()
    if err != nil {
        return "", fmt.Errorf("failed to get current branch name: %w", err)
    }
    return strings.TrimSpace(string(output)), nil
}

// GetHeadCommitHash returns the current HEAD commit hash of the git repository.
func GetHeadCommitHash() (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// Check if the error is related to not being in a git repository
		if strings.Contains(stderr.String(), "not a git repository") {
			return "", errors.New("not a git repository")
		}
		return "", fmt.Errorf("failed to execute git rev-parse HEAD: %w, stderr: %s", err, stderr.String())
	}

	commitHash := strings.TrimSpace(stdout.String())
	if commitHash == "" {
		return "", errors.New("empty commit hash returned from git rev-parse HEAD")
	}
	return commitHash, nil
}
