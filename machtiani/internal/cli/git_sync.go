package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL string, apiKey *string, force bool, verbose bool, config utils.Config, headCommitHash string) error {
	// Start timer
	startTime := time.Now()
	// Get the current branch name
	branchName, err := git.GetBranch()
	if err != nil {
		return fmt.Errorf("Error retrieving current branch name: %w", err)
	}
	config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return fmt.Errorf("Error retrieving ignore files: %w", err)
	}

	_, err = api.CheckStatus(remoteURL)
	if err != nil {
		if strings.Contains(err.Error(), "does not exist") {
			// Repository doesn't exist.
			// If force flag is NOT present, perform the initial checks and prompt.
			if !force {
				fmt.Println() // Prints a new line
				fmt.Println("Ignoring files based on .machtiani.ignore:")
				if len(ignoreFiles) == 0 {
					fmt.Println("No files to ignore.")
				} else {
					fmt.Println() // Prints another new line
					for _, path := range ignoreFiles {
						fmt.Println(path)
					}
				}

				// Call AddRepository with dryRun=true to potentially set up initial state and get estimates
				// We ignore the response message here as it's just for estimation/setup.
				_, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, true) // dryRun = true
				if err != nil {
					// Don't wrap specific "already exists" errors if they occur during dry run setup
					if !strings.Contains(err.Error(), "already exists") {
						return fmt.Errorf("error during initial repository check/add (dry run): %w", err)
					}
					// If it "already exists" during dry run, maybe another process added it. Proceed cautiously.
					fmt.Println("Warning: Repository detected during initial check, proceeding to final confirmation.")
				}

				// Wait for the repository lock file to disappear (indicates dry run/setup is done)
				if err := waitForRepoConfirmation(remoteURL, apiKey, remoteURL); err != nil {
					return fmt.Errorf("error waiting for repository confirmation after initial check: %w", err)
				}

				tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
				if err != nil {
					return fmt.Errorf("error getting token count: %w", err)
				}
				fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
				fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

				// This delete seems tied to the dry run / estimation process
				_, err = api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true) // ignoreIfNotExists = true
				if err != nil {
					return fmt.Errorf("error cleaning up after estimate: %v", err)
				}

				// Print timing if verbose flag is set, before the prompt
				if verbose {
					duration := time.Since(startTime)
					fmt.Printf("Time to 'count tokens' point: %s\n", duration)
				}
			} // end if !force

			// This check happens regardless of whether the above block ran.
			// If force=true, it proceeds directly.
			// If force=false, it prompts after showing estimates.
			if force || utils.ConfirmProceed() {
				response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, false) // dryRun = false

				if err != nil {
					return fmt.Errorf("Error adding repository: %w", err)
				}

				fmt.Println(response.Message)
				fmt.Println("---")
				fmt.Println("Your repo is getting added to machtiani is in progress!")
				fmt.Println("Please check back by running `machtiani status` to see if it completed.")
				return nil
			} else {
				// User chose not to proceed (only reachable if force=false)
				fmt.Println("Operation cancelled by user.")
				return nil
			}
		} else {
			// Handle other errors from CheckStatus
			return fmt.Errorf("Error checking repository status: %w", err)
		}
	}
	// If force isn't true, then do
	_, err = api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, true)
	if err != nil {
		return fmt.Errorf("Error syncing repository: %w", err)
	}

	tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
	if err != nil {
		return fmt.Errorf("error getting token count: %w", err)
	}
	fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
	fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

	if force || utils.ConfirmProceed() {
		message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, false)
		if err != nil {
			return fmt.Errorf("Error syncing repository: %w", err)
		}

		// Print the returned message
		fmt.Println(message)
	} else {
		fmt.Println("Operation cancelled by user.")
		return nil
	}
	return nil
}

func waitForRepoConfirmation(remoteURL string, apiKey *string, codehostURL string) error {
	const maxRetries = 100
	const waitDuration = 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		status, err := api.CheckStatus(codehostURL)
		if err != nil {
			// Error occurred while checking status
			return fmt.Errorf("error checking repository status: %w", err)
		}

		if !status.LockFilePresent {
			// Repository is confirmed as added (lock file is no longer present)
			return nil
		}

		// Wait before retrying
		time.Sleep(waitDuration)
	}
	return fmt.Errorf("timed out waiting for repository confirmation")
}
