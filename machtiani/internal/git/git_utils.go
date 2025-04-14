package git

import (
	"bytes"

	"errors"
	"fmt"
	"log" // Added import
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// GetLatestMachtianiSystemMessage fetches the latest system message from the machtiani remote git repository
func GetLatestMachtianiSystemMessage(machtianiGitRemoteURL string) (string, error) {
	// Try both methods to get the system message
	msgGit, errGit := getSystemMessageWithGit(machtianiGitRemoteURL)
	if errGit == nil {

		return msgGit, nil
	}

	// If git fails, potentially add HTTP fallback here if needed
	// msgHttp, errHttp := getSystemMessageWithHttp(machtianiGitRemoteURL) ...

	// Return combined error if all methods fail
	return "", fmt.Errorf("failed to fetch system message using git: %v", errGit) // Simplified error
}

// getSystemMessageWithGit fetches the system message using a shallow clone
func getSystemMessageWithGit(machtianiGitRemoteURL string) (string, error) {
	remoteURL := machtianiGitRemoteURL
	filePath := "system-message.txt"
	branch := "master" // Or main, depending on the repo

	// Create temporary directory that will be automatically cleaned up
	tmpDir, err := os.MkdirTemp("", "machtiani-system-message-*") // Added pattern for clarity
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err) // Use %w for error wrapping
	}
	defer os.RemoveAll(tmpDir) // Ensures cleanup when function exits

	// Run git clone with depth 1 (shallow clone - only gets latest commit)
	// Added --no-tags and --filter=blob:none for efficiency if supported by git version
	cloneCmd := exec.Command("git", "clone", "--depth", "1", "--branch", branch, "--no-tags", "--filter=blob:none", remoteURL, tmpDir)
	if output, err := cloneCmd.CombinedOutput(); err != nil {
		// Check if the branch doesn't exist (common error)
		if strings.Contains(string(output), fmt.Sprintf("Remote branch %s not found", branch)) {
			// Try 'main' branch as a fallback
			branch = "main"
			cloneCmd = exec.Command("git", "clone", "--depth", "1", "--branch", branch, "--no-tags", "--filter=blob:none", remoteURL, tmpDir)
			if output, err = cloneCmd.CombinedOutput(); err != nil {
				return "", fmt.Errorf("git clone command failed for branches 'master' and 'main': %w, output: %s", err, output)
			}
		} else {
			return "", fmt.Errorf("git clone command failed: %w, output: %s", err, output)
		}
	}

	// Read the file directly from the cloned repository
	content, err := os.ReadFile(filepath.Join(tmpDir, filePath))
	if err != nil {
		// Check if file not found specifically
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file '%s' not found in the repository on branch '%s'", filePath, branch)
		}
		return "", fmt.Errorf("failed to read system message file '%s': %w", filePath, err)
	}

	return string(content), nil
}


// getSystemMessageLastDisplayedPath returns the path to the file that tracks when the system message was last displayed
func getSystemMessageLastDisplayedPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	machtianiDir := filepath.Join(homeDir, ".machtiani")
	if err := os.MkdirAll(machtianiDir, 0755); err != nil {
		// Check if permission denied
		if os.IsPermission(err) {
			return "", fmt.Errorf("permission denied creating directory %s: %w", machtianiDir, err)
		}
		return "", fmt.Errorf("failed to create .machtiani directory %s: %w", machtianiDir, err)
	}

	return filepath.Join(machtianiDir, "system-message-last-displayed"), nil
}

// getLastSystemMessagePath returns the path to the file that stores the last shown system message
func getLastSystemMessagePath() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", fmt.Errorf("failed to get user home directory: %w", err)
    }

    machtianiDir := filepath.Join(homeDir, ".machtiani")
    if err := os.MkdirAll(machtianiDir, 0755); err != nil {
        if os.IsPermission(err) {
            return "", fmt.Errorf("permission denied creating directory %s: %w", machtianiDir, err)
        }
        return "", fmt.Errorf("failed to create .machtiani directory: %w", err)
    }

    return filepath.Join(machtianiDir, ".last_system_message"), nil
}

// GetLastSystemMessage reads the content of the last shown system message
func GetLastSystemMessage() (string, error) {
    path, err := getLastSystemMessagePath()
    if err != nil {
        return "", err
    }

    content, err := os.ReadFile(path)
    if os.IsNotExist(err) {
        return "", nil // No last message exists yet
    } else if err != nil {
        return "", fmt.Errorf("failed to read last system message: %w", err)
    }

    return string(content), nil
}

// SaveSystemMessage saves the given message as the last shown system message
func SaveSystemMessage(message string) error {
    path, err := getLastSystemMessagePath()
    if err != nil {
        return err
    }

    err = os.WriteFile(path, []byte(message), 0640)
    if err != nil {
        return fmt.Errorf("failed to write system message to %s: %w", path, err)
    }
    return nil
}

// RecordSystemMessageDisplayed records the current time as when the system message was last displayed
func RecordSystemMessageDisplayed() error {
	path, err := getSystemMessageLastDisplayedPath()
	if err != nil {
		return err // Error already descriptive from getSystemMessageLastDisplayedPath
	}

	// Current Unix timestamp
	now := strconv.FormatInt(time.Now().Unix(), 10)

	// Write with slightly more restrictive permissions potentially
	err = os.WriteFile(path, []byte(now), 0640) // Changed permissions slightly
	if err != nil {
		return fmt.Errorf("failed to write timestamp to %s: %w", path, err)
	}
	return nil
}

// ShouldDisplaySystemMessage checks if the system message should be displayed
func ShouldDisplaySystemMessage(frequencyHours int) (bool, error) {
	path, err := getSystemMessageLastDisplayedPath()
	if err != nil {
		// If there's an error getting the path (e.g., permissions), log it but display message
		log.Printf("Warning: could not get system message last displayed path (%s): %v. Displaying message.", path, err)
		return true, nil // Return nil error as we handled it by deciding to display
	}

	// If the file doesn't exist, show the message
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return true, nil
	} else if err != nil {
		// For other read errors, log warning and display message
		log.Printf("Warning: could not read system message last displayed file (%s): %v. Displaying message.", path, err)
		return true, nil // Return nil error as we handled it
	}

	// Parse the timestamp from the file
	timestamp, err := strconv.ParseInt(strings.TrimSpace(string(content)), 10, 64) // Added TrimSpace
	if err != nil {
		// If parsing fails, log warning and display the message
		log.Printf("Warning: failed to parse timestamp '%s' from %s: %v. Displaying message.", string(content), path, err)
		return true, nil // Return nil error as we handled it
	}

	lastDisplayed := time.Unix(timestamp, 0).UTC()
	now := time.Now().UTC()

	// Calculate today's midnight UTC
	todayMidnight := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// If the last display was before today's midnight, show the message
	if lastDisplayed.Before(todayMidnight) {
		return true, nil
	}

	// Check if it's been frequencyHours since the last display (optional check, might be redundant with daily check)
	// If frequencyHours is 0 or less, disable this check
	if frequencyHours > 0 && now.Sub(lastDisplayed) >= time.Duration(frequencyHours)*time.Hour {
		return true, nil
	}

	return false, nil
}

func GetProjectName() (string, error) {
	// Run the git command to get the remote URL
	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	output, err := cmd.CombinedOutput() // Use CombinedOutput to capture stderr
	if err != nil {
		// Check if not a git repo or no remote named origin
		stderr := string(output)
		if strings.Contains(stderr, "not a git repository") {
			return "", errors.New("not a git repository")
		}
		if strings.Contains(stderr, "No such remote") {
			return "", errors.New("remote 'origin' not found")
		}
		return "", fmt.Errorf("failed to get remote.origin.url: %w, output: %s", err, stderr)
	}

	// Parse the remote URL
	url := string(output)
	trimmedURL := strings.TrimSpace(url)
	if trimmedURL == "" {
		return "", errors.New("remote 'origin' URL is empty")
	}

	// Extract the project name (assuming it's the last part of the URL before .git)
	// More robust parsing for different URL formats (SSH, HTTPS)
	var projectName string
	if strings.Contains(trimmedURL, "/") {
		parts := strings.Split(trimmedURL, "/")
		lastPart := parts[len(parts)-1]
		projectName = strings.TrimSuffix(lastPart, ".git")
	} else if strings.Contains(trimmedURL, ":") { // Handle SSH syntax like git@github.com:user/repo.git
		parts := strings.Split(trimmedURL, ":")
		lastPart := parts[len(parts)-1]
		projectName = strings.TrimSuffix(lastPart, ".git")
	} else {
		// Fallback or error if format is unexpected
		projectName = strings.TrimSuffix(trimmedURL, ".git") // Simple fallback
	}

	if projectName == "" {
		return "", fmt.Errorf("could not parse project name from remote URL: %s", trimmedURL)
	}

	return projectName, nil
}

func GetRemoteURL(remoteName *string) (string, error) {
	if remoteName == nil || *remoteName == "" {
		defaultRemote := "origin" // Default to origin if not provided
		remoteName = &defaultRemote
		// return "", fmt.Errorf("remote name cannot be empty") // Or default to origin? Let's default.
	}
	remoteURL, err := getRemoteURL(*remoteName)
	if err != nil {
		return "", fmt.Errorf("error fetching remote URL for '%s': %w", *remoteName, err) // Use %w
	}
	return remoteURL, nil
}

func getRemoteURL(remoteName string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", remoteName)
	output, err := cmd.CombinedOutput() // Use CombinedOutput
	if err != nil {
		// Check specific errors
		stderr := string(output)
		if strings.Contains(stderr, "No such remote") {
			return "", fmt.Errorf("remote '%s' not found", remoteName)
		}
		return "", fmt.Errorf("failed to get remote URL for %s: %w, output: %s", remoteName, err, stderr)
	}
	return strings.TrimSpace(string(output)), nil
}

func GetBranch() (string, error) {
	// Run the git command to get the current branch name
	// FYI it returns 'HEAD' if not in a branch, if detached.
	branchNameCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	output, err := branchNameCmd.CombinedOutput() // Use CombinedOutput
	if err != nil {
		stderr := string(output)
		if strings.Contains(stderr, "not a git repository") {
			return "", errors.New("not a git repository")
		}
		// Check for detached HEAD specifically? 'HEAD' output is the indicator.
		return "", fmt.Errorf("failed to get current branch name: %w, output: %s", err, stderr)
	}
	branchName := strings.TrimSpace(string(output))

	if branchName == "HEAD" {
		// Detached HEAD state, try to find a branch from origin
		// Use 'git branch --show-current' first, might be simpler if git version >= 2.22
		showCurrentCmd := exec.Command("git", "branch", "--show-current")
		showCurrentOutput, showCurrentErr := showCurrentCmd.Output()
		if showCurrentErr == nil && strings.TrimSpace(string(showCurrentOutput)) != "" {
			// If --show-current works and gives a non-empty result, use it
			return strings.TrimSpace(string(showCurrentOutput)), nil
		}
		// Fallback to older method if --show-current fails or is empty (might happen in detached HEAD)

		// Try to find a branch that points to HEAD
		symbolicRefCmd := exec.Command("git", "symbolic-ref", "-q", "--short", "HEAD")
		symbolicOutput, symbolicErr := symbolicRefCmd.Output()
		if symbolicErr == nil {
			sOutput := strings.TrimSpace(string(symbolicOutput))
			if sOutput != "" {
				return sOutput, nil // Found a symbolic ref (branch name)
			}
		}

		// If still 'HEAD', try the remote contains method (original logic)
		branchesCmd := exec.Command("git", "branch", "--remotes", "--contains", "HEAD")
		branchesOutput, branchesErr := branchesCmd.CombinedOutput() // Use CombinedOutput
		if branchesErr != nil {
			// Log the error but continue, maybe HEAD doesn't exist on remote yet
			log.Printf("Warning: error getting remote branches containing HEAD: %v, output: %s", branchesErr, string(branchesOutput))
			// Fallback to just returning "HEAD" or an error? Let's return an error.
			return "", fmt.Errorf("detached HEAD state, and failed to check remote branches: %w", branchesErr)
		}

		branches := bytes.Split(branchesOutput, []byte{'\n'})
		for _, branch := range branches {
			branchStr := string(bytes.TrimSpace(branch))
			// Prefer origin, but accept others if origin isn't found
			if strings.Contains(branchStr, "/") && !strings.Contains(branchStr, "->") { // Avoid lines like 'origin/HEAD -> origin/main'
				// Return the first suitable remote branch found
				parts := strings.SplitN(branchStr, "/", 2)
				if len(parts) == 2 {
					// Return branch name without remote part
					return parts[1], nil
				}
			}
		}

		// If no suitable remote branch is found on origin, return an error indicating detached HEAD
		return "", errors.New("detached HEAD state and no associated branch found locally or on remotes")
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
		// Check if repo exists but has no commits yet
		if strings.Contains(stderr.String(), "ambiguous argument 'HEAD'") {
			return "", errors.New("git repository exists but has no commits yet")
		}
		return "", fmt.Errorf("failed to execute git rev-parse HEAD: %w, stderr: %s", err, stderr.String())
	}

	commitHash := strings.TrimSpace(stdout.String())
	if commitHash == "" {
		// This case should ideally be caught by the stderr checks above
		return "", errors.New("empty commit hash returned from git rev-parse HEAD")
	}
	return commitHash, nil
}

// --- START NEW FUNCTION ---

// ApplyGitPatches finds patch files in patchDir newer than existingPatchFiles and applies them using `git apply`.
// It logs the success or failure of applying each patch.
func ApplyGitPatches(patchDir string, existingPatchFiles []string) error {
	// Create a map of existing patch files for quick lookup
	existingPatchMap := make(map[string]bool)
	for _, file := range existingPatchFiles {
		existingPatchMap[filepath.Base(file)] = true // Store only basename for comparison
	}

	// Get all patch files AFTER writing (caller ensures writing happened before this call)
	allPatchFiles, err := filepath.Glob(filepath.Join(patchDir, "*.patch"))
	if err != nil {
		// This is a more fundamental error than individual patch failures
		return fmt.Errorf("error finding patch files in %s: %w", patchDir, err)
	}

	// Find which files are new by comparing with existing files
	var newPatchFiles []string
	for _, file := range allPatchFiles {
		// Compare basenames to handle potential relative vs absolute path differences
		if !existingPatchMap[filepath.Base(file)] {
			newPatchFiles = append(newPatchFiles, file)
		}
	}

	if len(newPatchFiles) == 0 {
		log.Printf("No new patch files found in %s to apply.", patchDir)
		return nil // Not an error
	}

	log.Printf("Found %d new patch file(s) to apply.", len(newPatchFiles))

	appliedCount := 0
	failedCount := 0
	// Apply only the newly created patch files
	for _, patchPath := range newPatchFiles {
		patchBaseName := filepath.Base(patchPath)
		log.Printf("Attempting to apply patch: %s", patchBaseName)
		// Use --verbose for potentially more info, or keep it simple
		// Consider adding --check first? Or --stat?
		// Let's add --stat to show what would be applied before applying
		statCmd := exec.Command("git", "apply", "--stat", patchPath)
		statOutput, statErr := statCmd.CombinedOutput()
		if statErr != nil {
			log.Printf("Warning: 'git apply --stat %s' failed: %v\nOutput:\n%s", patchBaseName, statErr, string(statOutput))
			// Decide whether to proceed with apply despite stat failure? Let's proceed but log warning.
		} else {
			log.Printf("Patch stats for %s:\n%s", patchBaseName, string(statOutput))
		}


		// Now actually apply
		// Use --reject to leave rejected hunks in .rej files instead of failing outright?
		// Use --allow-empty to avoid errors if the patch is empty?
		// Let's try with --reject for more robustness.
		applyCmd := exec.Command("git", "apply", "--reject", patchPath)
		applyOutput, applyErr := applyCmd.CombinedOutput()

		if applyErr != nil {
			failedCount++
			// Log detailed error including output
			log.Printf("Error applying patch %s: %v\nGit output:\n%s",
				patchBaseName, applyErr, string(applyOutput))
			log.Printf("Rejected hunks (if any) may be found in .rej files.")
		} else {
			appliedCount++
			log.Printf("Successfully applied patch: %s", patchBaseName)
			// Optionally log the output even on success if it's useful (e.g., warnings)
			if len(applyOutput) > 0 && strings.TrimSpace(string(applyOutput)) != "" {
				log.Printf("Git apply output for %s:\n%s", patchBaseName, string(applyOutput))
			}
		}
	}

	log.Printf("Patch application summary: %d applied, %d failed.", appliedCount, failedCount)

	// Decide if any failure constitutes an error return for the whole function.
	// Currently, it logs failures but returns nil, mirroring the original logic.
	// If we want to signal that *some* patches failed, we could return a custom error.
	// For now, let's stick to nil return on completion.
	return nil
}

// --- END NEW FUNCTION ---
