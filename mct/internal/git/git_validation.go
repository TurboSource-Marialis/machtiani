package git

import (
	"fmt"
	"os/exec"
	"strings"
)


// ValidateHeadCommitExistsOnRemote checks if the given commit exists on any remote branch
// by checking priority branches first (current branch if not detached, then master and main),
// then checking all other branches.
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

	// Extract branch names from ls-remote output and collect references
	remoteBranches := make(map[string]struct{})
	var branchRefs []string
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		branchRef := parts[1]
		branchName := strings.TrimPrefix(branchRef, "refs/heads/")
		remoteBranches[branchName] = struct{}{}
		branchRefs = append(branchRefs, branchRef)
	}

	// Determine priority branches to check first
	priorityBranches, err := getPriorityBranches()
	if err != nil {
		return fmt.Errorf("failed to determine priority branches: %w", err)
	}

	// Check priority branches first
	for _, pb := range priorityBranches {
		if _, exists := remoteBranches[pb]; !exists {
			fmt.Printf("Priority branch %q does not exist on remote, skipping\n", pb)
			continue
		}

		fmt.Printf("Checking priority branch: %s\n", pb)
		fetchCmd := exec.Command("git", "fetch", "origin", pb)
		if fetchOutput, fetchErr := fetchCmd.CombinedOutput(); fetchErr != nil {
			fmt.Printf("Warning: failed to fetch branch %s: %v\nOutput: %s", pb, fetchErr, string(fetchOutput))
			continue
		}

		contained, err := isCommitInBranch(commitHash, "origin/"+pb)
		if err != nil {
			fmt.Printf("Error checking commit in branch %s: %v\n", pb, err)
			continue
		}
		if contained {
			fmt.Printf("Commit found in priority branch: %s\n", pb)
			return nil
		}
	}

	// Check remaining branches
	for _, branchRef := range branchRefs {
		branchName := strings.TrimPrefix(branchRef, "refs/heads/")

		// Skip branches already checked in priority list
		if isPriorityBranch(branchName, priorityBranches) {
			fmt.Printf("Skipping already checked priority branch: %s\n", branchName)
			continue
		}

		fmt.Printf("Fetching branch: %s\n", branchName)
		fetchCmd := exec.Command("git", "fetch", "origin", branchName)
		if _, err := fetchCmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: failed to fetch branch %s: %v\n", branchName, err)
			continue
		}

		contained, err := isCommitInBranch(commitHash, "origin/"+branchName)
		if err != nil {
			fmt.Printf("Warning: error checking commit presence in %s: %v\n", branchName, err)
			continue
		}
		if contained {
			fmt.Printf("Commit found in branch: %s\n", branchName)
			return nil
		}
	}

	return fmt.Errorf("commit %s not found in any remote branch", commitHash)
}

// getPriorityBranches returns the list of branches to check first
func getPriorityBranches() ([]string, error) {
	var branches []string

	// Add current branch if not detached
	detached, err := IsDetachedHead()
	if err != nil {
		return nil, fmt.Errorf("failed to check detached HEAD state: %w", err)
	}
	if !detached {
		currentBranch, err := GetBranch()
		if err != nil {
			return nil, fmt.Errorf("failed to get current branch: %w", err)
		}
		branches = append(branches, currentBranch)
	}

	// Add standard default branches
	return append(branches, "master", "main"), nil
}

// isCommitInBranch checks if the commit exists in the specified branch
func isCommitInBranch(commitHash, branchRef string) (bool, error) {
	cmd := exec.Command("git", "merge-base", "--is-ancestor", commitHash, branchRef)
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 0:
				return true, nil
			case 1:
				return false, nil
			default:
				return false, fmt.Errorf("merge-base failed with unexpected code %d", exitErr.ExitCode())
			}
		}
		return false, fmt.Errorf("merge-base failed: %w", err)
	}
	return true, nil
}

// isPriorityBranch checks if a branch is in the priority list
func isPriorityBranch(branchName string, priorityBranches []string) bool {
	for _, pb := range priorityBranches {
		if pb == branchName {
			return true
		}
	}
	return false
}
