package cli

import (
	"fmt"
    "log"
	"strings"
	"time"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)


func handleSync(remoteURL string, apiKey *string, force bool, verbose bool, cost bool, costOnly bool, config utils.Config, headCommitHash string, amplificationLevel string, depthLevel int) error {
	startTime := time.Now()

	// Get the current HEAD commit hash if not provided
	var err error
	if headCommitHash == "" {
		headCommitHash, err = git.GetHeadCommitHash()
		if err != nil {
			return fmt.Errorf("Error retrieving HEAD commit hash: %w", err)
		}
	}

	// Validate that the HEAD commit matches the remote branch
	//err = utils.ValidateHeadCommitExistsOnRemote(headCommitHash)
	//if err != nil && !force {
	//	return fmt.Errorf("Validation failed: %w. Use --force to bypass this validation.", err)
	//} else if err != nil && force {
	//	fmt.Printf("Warning: %v. Proceeding anyway due to --force flag.\n", err)
	//}

    var branchName string
    detached, err := git.IsDetachedHead()
    if err != nil {
        // we couldn't even tell—we'll warn and send no branch
        log.Printf("Warning: unable to determine detached HEAD status: %v", err)
    }
    if !detached {
        // only when we're *not* detached do we send a branch
        branchName, err = git.GetBranch()
        if err != nil {
            log.Printf("Warning: unable to read current branch name: %v", err)
            // leave branchName empty
            branchName = ""
        }
    }

	if err != nil {
		return fmt.Errorf("Error retrieving current branch name: %w", err)
	}
	config, ignoreFiles, err := utils.LoadConfigAndIgnoreFiles()
	if err != nil {
		return fmt.Errorf("Error retrieving ignore files: %w", err)
	}

	_, err = api.CheckStatus(remoteURL)
	repoExists := err == nil

	if !repoExists {
		if strings.Contains(err.Error(), "does not exist") {
			fmt.Println()
			fmt.Println("Repository not found on Machtiani. Preparing for initial sync.")
			fmt.Println("Ignoring files based on .machtiani.ignore:")
			if len(ignoreFiles) == 0 {
				fmt.Println("No files to ignore.")
			} else {
				fmt.Println()
				for _, path := range ignoreFiles {
					fmt.Println(path)
				}
			}

			if cost || costOnly {
				// Perform dry-run add to allow estimation
				_, err = api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, true, headCommitHash, true, amplificationLevel, depthLevel)
				if err != nil && !strings.Contains(err.Error(), "already exists") {
					return fmt.Errorf("error during initial repository check/add (dry run): %w", err)
				}

				if err := waitForRepoConfirmation(remoteURL, apiKey, remoteURL); err != nil {
					fmt.Printf("Warning: error waiting for repository confirmation after dry-run: %v\n", err)
					_, delErr := api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true)
					if delErr != nil {
						fmt.Printf("Warning: error cleaning up after failed dry-run confirmation: %v\n", delErr)
					}
					return fmt.Errorf("failed to confirm repository setup during dry-run, cannot proceed")
				}

				if cost || costOnly {
					if err := estimateAndPrintCost(remoteURL, apiKey, verbose, startTime); err != nil {
						_, delErr := api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true)
						if delErr != nil {
							fmt.Printf("Warning: error cleaning up after estimation failure: %v\n", delErr)
						}
						return fmt.Errorf("failed to estimate cost: %w", err)
					}
				}

				_, err = api.DeleteStore(remoteURL, remoteURL, "git", apiKey, api.RepoManagerURL, true)
				if err != nil {
					fmt.Printf("Warning: error cleaning up after dry run/estimation: %v\n", err)
				}

				if costOnly {
					fmt.Println("Cost estimation complete. Exiting as requested by --cost-only.")
					return nil
				}
			}

			if force || utils.ConfirmProceed() {
				response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, false, amplificationLevel, depthLevel)
				if err != nil {
					return fmt.Errorf("Error adding repository: %w", err)
				}
				fmt.Println(response.Message)
				fmt.Println("---")
				fmt.Println("Your repo is getting added to machtiani is in progress!")
				fmt.Println("Please check back by running `mct status` to see if it completed.")
				return nil
			} else {
				fmt.Println("Operation cancelled by user.")
				return nil
			}
		} else {
			return fmt.Errorf("Error checking repository status: %w", err)
		}
	}

	fmt.Println("Repository found. Preparing to sync branch:", branchName)

	if cost || costOnly {
		_, err = api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, true, headCommitHash, true, amplificationLevel, depthLevel)
		if err != nil {
			return fmt.Errorf("Error during repository sync check (dry run): %w", err)
		}

		if cost || costOnly {
			if err := estimateAndPrintCost(remoteURL, apiKey, verbose, startTime); err != nil {
				return fmt.Errorf("failed to estimate cost: %w", err)
			}
		}
	}

	if costOnly {
		fmt.Println("Cost estimation complete. Exiting as requested by --cost-only.")
		return nil
	}

	if force || utils.ConfirmProceed() {
		message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, false, amplificationLevel, depthLevel)
		if err != nil {
			return fmt.Errorf("Error syncing repository: %w", err)
		}
		fmt.Println(message)
	} else {
		fmt.Println("Operation cancelled by user.")
	}
	return nil
}

func estimateAndPrintCost(remoteURL string, apiKey *string, verbose bool, startTime time.Time) error {
	fmt.Println("---")
	fmt.Println("Estimating token cost...")
	tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
	if err != nil {
		return fmt.Errorf("error getting token count: %w", err)
	}
	fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
	fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)
	fmt.Println("---")

	if verbose {
		duration := time.Since(startTime)
		fmt.Printf("Time until cost estimation finished: %s\n", duration)
	}
	return nil
}

func waitForRepoConfirmation(remoteURL string, apiKey *string, codehostURL string) error {
	const maxRetries = 1800
	const waitDuration = 1 * time.Second

	for i := 0; i < maxRetries; i++ {
		status, err := api.CheckStatus(codehostURL)
		if err != nil {
			if strings.Contains(err.Error(), "does not exist") {
				return fmt.Errorf("repository disappeared during confirmation check")
			}
			fmt.Printf("Warning: transient error checking status during wait: %v\n", err)
		} else {
			if !status.LockFilePresent {
				fmt.Println("Repository confirmation received.")
				return nil
			}
		}

		time.Sleep(waitDuration)
		if (i+1)%10 == 0 {
			fmt.Printf("Still waiting for confirmation (%ds)...\n", (i+1)*int(waitDuration.Seconds()))
		}
	}
	fmt.Println("Timed out waiting for repository confirmation.")
	return fmt.Errorf("timed out waiting for repository confirmation")
}
