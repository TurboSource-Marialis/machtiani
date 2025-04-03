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
			// If the repository doesn't exist, add it

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

	        response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, true)
	        if err != nil {
                return fmt.Errorf("Error adding repository: %w", err)
	        }

		    // Wait for the repository to be confirmed as added
		    if err := waitForRepoConfirmation(remoteURL, apiKey, remoteURL); err != nil {
                return fmt.Errorf("Error waiting for repository confirmation: %w", err)
		    }

		    tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
		    if err != nil {
                return fmt.Errorf("error getting token count: %w", err)
		    }
		    fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
		    fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

            _, err = api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true)
            if err != nil {
                return fmt.Errorf("Error deleting store: %v", err)
            }

            // Print timing if verbose flag is set, before the prompt/force check
            if verbose {
                duration := time.Since(startTime)
                fmt.Printf("Time to 'count tokens' point: %s\n", duration)
            }

		    if force || utils.ConfirmProceed() {

	            response, err = api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, false)
	            if err != nil {
	                return fmt.Errorf("Error adding repository: %w", err)
	            }

	            fmt.Println(response.Message)
	            fmt.Println("---")
	            fmt.Println("Your repo is getting added to machtiani is in progress!")
	            fmt.Println("Please check back by running `machtiani status` to see if it completed.")
	            return nil
		    } else {
                return nil
		    }
		} else {
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

