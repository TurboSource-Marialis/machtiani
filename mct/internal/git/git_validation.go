package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ValidateHeadCommitExistsOnRemote checks if the given commit exists on any remote branch
// by fetching one branch at a time rather than fetching all branches at once
func ValidateHeadCommitExistsOnRemote(commitHash string) error {
	fmt.Println("Checking if commit exists on remote branches...")

	// Step 1: List all remote branch references without fetching content
	cmd := exec.Command("git", "ls-remote", "--heads", "origin")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list remote branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("no remote branches found")
	}

	// Step 2 & 3: Fetch one branch at a time and check for the target commit
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		// Extract the branch name from the reference
		branchRef := parts[1]
		branchName := strings.TrimPrefix(branchRef, "refs/heads/")

		// Fetch just this specific branch
		fmt.Printf("Fetching branch: %s\n", branchName)
		fetchCmd := exec.Command("git", "fetch", "origin", branchName)
		// fetchOutput is not used; discard output and check error
		_, err := fetchCmd.CombinedOutput()
		if err != nil {
			// Log error but continue with next branch
			fmt.Printf("Warning: failed to fetch branch %s: %v\n", branchName, err)
			continue
		}

		// Check if the commit is part of this branch
		containsCmd := exec.Command("git", "merge-base", "--is-ancestor", commitHash, "origin/"+branchName)
		err = containsCmd.Run()

		if err == nil {
			// Exit code 0 means the commit is an ancestor of the branch tip
			fmt.Printf("Commit found in branch: %s\n", branchName)
			return nil // Found the commit!
		}

		// An exit code of 1 means the commit is not an ancestor, which is expected
		// Any other error is unexpected and should be logged
		if exitErr, ok := err.(*exec.ExitError); !ok || exitErr.ExitCode() != 1 {
			fmt.Printf("Warning: error checking if commit %s is in branch %s: %v\n",
				commitHash[:8], branchName, err)
		}
	}

	// If we get here, the commit wasn't found in any branch
	return fmt.Errorf("commit %s not found in any remote branch", commitHash)
}
