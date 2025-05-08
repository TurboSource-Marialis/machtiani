package cli

import (
	"fmt"
    "log"
	"strings"
	"time"

	"github.com/turboSource-marialis/machtiani/mct/internal/api"
	"github.com/turboSource-marialis/machtiani/mct/internal/git"
	"github.com/turboSource-marialis/machtiani/mct/internal/utils"
)




func handleSync(remoteURL string, apiKey *string, force bool, verbose bool, cost bool, costOnly bool, config utils.Config, headCommitHash string, amplificationLevel string, depthLevel int, model string, modelThreads int) error {

	startTime := time.Now()

    llmModel     := model                            // model name (eg. gpt‑4o)
    llmModelKey  := config.Environment.ModelAPIKey   // real API key

	modelBaseURL := config.Environment.ModelBaseURL

	// Get the current HEAD commit hash if not provided
	var err error
	if headCommitHash == "" {
		headCommitHash, err = git.GetHeadCommitHash()
		if err != nil {
			return fmt.Errorf("Error retrieving HEAD commit hash: %w", err)
		}
	}

	// Validate that the HEAD commit matches the remote branch
	err = utils.ValidateHeadCommitExistsOnRemote(headCommitHash)
	if err != nil && !force {
		return fmt.Errorf("Validation failed: %w. Use --force to bypass this validation.", err)
	} else if err != nil && force {
		fmt.Printf("Warning: %v. Proceeding anyway due to --force flag.\n", err)
	}

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


                _, err = api.AddRepository(remoteURL, remoteURL, apiKey, llmModelKey, api.RepoManagerURL, modelBaseURL, true, headCommitHash, true, amplificationLevel, depthLevel, modelThreads, model)
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
                response, err := api.AddRepository(remoteURL, remoteURL, apiKey, llmModelKey, api.RepoManagerURL, modelBaseURL, force, headCommitHash, false, amplificationLevel, depthLevel, modelThreads, model)
				if err != nil {
					return fmt.Errorf("Error adding repository: %w", err)
				}
				fmt.Println(response.Message)
				fmt.Println("---")

				if force {
					// If force flag is passed, maintain current behavior
					fmt.Println("Your repo is getting added to machtiani is in progress!")
					fmt.Println("Please check back by running `mct status` to see if it completed.")
					return nil
				} else {
					// Otherwise, wait for sync to complete like subsequent syncs
					fmt.Println("Waiting for initial sync to complete. This may take some time...")
					if err := waitForRepoConfirmation(remoteURL, apiKey, remoteURL); err != nil {
						return fmt.Errorf("Error waiting for initial sync: %w", err)
					}
					fmt.Println("Initial sync completed successfully.")
					return nil
				}
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


        _, err = api.FetchAndCheckoutBranch(
            remoteURL,
            remoteURL,
            branchName,
            apiKey,
            &llmModelKey,    // <- real key (string)
            &modelBaseURL,
            &llmModel,       // <- model name (string)
            force,           // force flag
            headCommitHash,  // headCommitHash
            true,            // useMockLLM = true for dry‑run
            amplificationLevel,
            depthLevel,
            modelThreads,
        )
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
       var message string

       message, err = api.FetchAndCheckoutBranch(
           remoteURL,
           remoteURL,
           branchName,
           apiKey,
           &llmModelKey,
           &modelBaseURL,
           &llmModel,       // <- model name
           force,           // respect --force here
           headCommitHash,
           false,           // useMockLLM = false for real sync
           amplificationLevel,
           depthLevel,
           modelThreads,
       )
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
	_, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
	if err != nil {
		return fmt.Errorf("error getting token count: %w", err)
	}
	// Only print inference tokens as requested
	fmt.Printf("Estimated tokens: %s\n", utils.FormatIntWithCommas(tokenCountInference))
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
            // If error logs present, abort with error
            if status.ErrorLogs != "" {
                return fmt.Errorf("Error during repository processing: %s", status.ErrorLogs)
            }
            // Proceed when lock file removed with no errors
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
