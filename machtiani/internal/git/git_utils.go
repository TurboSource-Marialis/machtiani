package git

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

)

// GetLatestMachtianiSystemMessage fetches the latest system message from the machtiani remote git repository
func GetLatestMachtianiSystemMessage(machtianiGitRemoteURL string) (string, error) {
	// Try both methods to get the system message
	msgGit, errGit := getSystemMessageWithGit(machtianiGitRemoteURL)
	if errGit == nil {
		return msgGit, nil
	}

	return "", fmt.Errorf("failed to fetch system message: git error: %v, http error: %v", errGit)
}

// getSystemMessageWithGit fetches the system message using a shallow clone
func getSystemMessageWithGit(machtianiGitRemoteURL string) (string, error) {
	remoteURL := machtianiGitRemoteURL
	filePath := "system-message.txt"
	branch := "master"

	// Create temporary directory that will be automatically cleaned up
	tmpDir, err := os.MkdirTemp("", "machtiani-system-message")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpDir) // Ensures cleanup when function exits

	// Run git clone with depth 1 (shallow clone - only gets latest commit)
	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, remoteURL, tmpDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("git clone command failed: %v, output: %s", err, output)
	}

	// Read the file directly from the cloned repository
	content, err := os.ReadFile(filepath.Join(tmpDir, filePath))
	if err != nil {
		return "", fmt.Errorf("failed to read system message file: %v", err)
	}

	return string(content), nil
}

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
	branchNameCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := branchNameCmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch name: %w", err)
	}
	branchName := strings.TrimSpace(string(output))

	if branchName == "HEAD" {
		// Detached HEAD state, try to find a branch from origin
		branchesCmd := exec.Command("git", "branch", "--remotes", "--contains", "HEAD")
		branchesOutput, branchesErr := branchesCmd.CombinedOutput()
		if branchesErr != nil {
			return "", fmt.Errorf("error getting remote branches: %w", branchesErr)
		}

		branches := bytes.Split(branchesOutput, []byte{'\n'})
		for _, branch := range branches {
			branchStr := string(bytes.TrimSpace(branch))
			if strings.HasPrefix(branchStr, "origin/") && !strings.Contains(branchStr, "HEAD") {
				// Found a branch on origin, remove "origin/" prefix
				return strings.TrimPrefix(branchStr, "origin/"), nil
			}
		}

		// If no suitable remote branch is found on origin, return an error indicating detached HEAD
		return "", fmt.Errorf("detached HEAD state and no remote branch on origin found for current commit")
	}

	return branchName, nil
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
