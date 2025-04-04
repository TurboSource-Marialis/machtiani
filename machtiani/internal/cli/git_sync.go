package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL string, apiKey *string, force bool, verbose bool, cost bool, config utils.Config, headCommitHash string) error {
    startTime := time.Now()
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
            // Repository doesn't exist
            if !force {
                fmt.Println()
                fmt.Println("Ignoring files based on .machtiani.ignore:")
                if len(ignoreFiles) == 0 {
                    fmt.Println("No files to ignore.")
                } else {
                    fmt.Println()
                    for _, path := range ignoreFiles {
                        fmt.Println(path)
                    }
                }

                _, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, true)
                if err != nil && !strings.Contains(err.Error(), "already exists") {
                    return fmt.Errorf("error during initial repository check/add (dry run): %w", err)
                }

                if err := waitForRepoConfirmation(remoteURL, apiKey, remoteURL); err != nil {
                    return fmt.Errorf("error waiting for repository confirmation after initial check: %w", err)
                }

                // Estimate tokens and check cost
                if err := estimateAndExitIfCost(remoteURL, apiKey, verbose, cost, startTime); err != nil {
                    return err
                }

                _, err = api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true)
                if err != nil {
                    return fmt.Errorf("error cleaning up after estimate: %v", err)
                }
            }

            if cost {
                return nil
            }

            if force || utils.ConfirmProceed() {
                response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, false)
                if err != nil {
                    return fmt.Errorf("Error adding repository: %w", err)
                }
                fmt.Println(response.Message)
                fmt.Println("---")
                fmt.Println("Your repo is getting added to machtiani is in progress!")
                fmt.Println("Please check back by running `machtiani status` to see if it completed.")
                return nil
            } else {
                fmt.Println("Operation cancelled by user.")
                return nil
            }
        } else {
            return fmt.Errorf("Error checking repository status: %w", err)
        }
    }

    // Repository exists
    _, err = api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, true)
    if err != nil {
        return fmt.Errorf("Error syncing repository: %w", err)
    }

    // Estimate tokens and check cost
    if err := estimateAndExitIfCost(remoteURL, apiKey, verbose, cost, startTime); err != nil {
        return err
    }

    if cost {
        return nil
    }

    if force || utils.ConfirmProceed() {
        message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, false)
        if err != nil {
            return fmt.Errorf("Error syncing repository: %w", err)
        }
        fmt.Println(message)
    } else {
        fmt.Println("Operation cancelled by user.")
    }
    return nil
}

// Helper function to estimate tokens and exit if cost flag is set
func estimateAndExitIfCost(remoteURL string, apiKey *string, verbose bool, cost bool, startTime time.Time) error {
    tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
    if err != nil {
        return fmt.Errorf("error getting token count: %w", err)
    }
    fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
    fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

    if cost {
        if verbose {
            duration := time.Since(startTime)
            fmt.Printf("Time to estimate tokens: %s\n", duration)
        }
    }
    return nil
}

var ErrCostExit = fmt.Errorf("cost estimation complete")

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
