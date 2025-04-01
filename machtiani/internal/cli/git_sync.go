package cli

import (
	"fmt"
	"strings"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL string, apiKey *string, force bool, config utils.Config, headCommitHash string) error {
    // Get the current branch name
    branchName, err := git.GetBranch()
    if err != nil {
        return fmt.Errorf("Error retrieving current branch name: %w", err)
    }

    _, err = api.CheckStatus(remoteURL)
    if err != nil {
        if strings.Contains(err.Error(), "does not exist") {
            // If the repository doesn't exist, add it

            if err := addRepo(remoteURL, apiKey, force, config, headCommitHash, true); err != nil {
                return fmt.Errorf("Error adding repository: %w", err)
            }

            tokenCountEmbedding, tokenCountInference, err := api.EstimateTokenCount(remoteURL, remoteURL, apiKey)

	        if err != nil {
	            return fmt.Errorf("error getting token count: %w", err)
	        }
	        // Print the token counts separately
	        fmt.Printf("Estimated embedding tokens: %d\n", tokenCountEmbedding)
	        fmt.Printf("Estimated inference tokens: %d\n", tokenCountInference)

            if force || utils.ConfirmProceed() {
                if err := addRepo(remoteURL, apiKey, force, config, headCommitHash, false); err != nil {
                    return fmt.Errorf("Error adding repository: %w", err)
                } else {
                    return nil
                }

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
    } else {
        api.EstimateTokenCount(remoteURL, remoteURL, apiKey)
    }

    if force || utils.ConfirmProceed() {
        message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, false)
        if err != nil {
            return fmt.Errorf("Error syncing repository: %w", err)
        }

        // Print the returned message
        fmt.Println(message)
    } else{
        return nil
    }
    return nil
}

func addRepo(remoteURL string, apiKey *string, force bool, config utils.Config, headCommitHash string, useMockLLM bool) error {
    response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, config.Environment.ModelBaseURL, force, headCommitHash, useMockLLM)
    if err != nil {
        return fmt.Errorf("Error adding repository: %w", err)
    }

    fmt.Println(response.Message)
    fmt.Println("---")
    fmt.Println("Your repo is getting added to machtiani is in progress!")
    fmt.Println("Please check back by running `machtiani status` to see if it completed.")
    return nil
}
