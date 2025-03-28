package cli

import (
	"fmt"
	"log"
	"strings"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL string, apiKey *string, force bool, config utils.Config, headCommitHash string, useMockLLM bool) error {
    // Get the current branch name
    branchName, err := git.GetBranch()
    if err != nil {
        return fmt.Errorf("Error retrieving current branch name: %w", err)
    }

    _, err = api.CheckStatus(remoteURL)
    if err != nil {
        if strings.Contains(err.Error(), "does not exist") {
            // If the repository doesn't exist, add it

            log.Printf("[DEBUG] cli.handleGitSync: Repository does not exist, calling addRepo. useMockLLM is NOT passed to addRepo.\n") // DEBUG PRINT (Clarification)
            if err := addRepo(remoteURL, apiKey, force, config, headCommitHash, useMockLLM); err != nil {
                return fmt.Errorf("Error adding repository: %w", err)
            } else {
                return nil
            }
        } else {
            return fmt.Errorf("Error checking repository status: %w", err)
        }
    }

    message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force, headCommitHash, useMockLLM)
    log.Printf("[DEBUG] cli.handleGitSync: Calling api.FetchAndCheckoutBranch with useMockLLM = %v\n", useMockLLM) // DEBUG PRINT
    if err != nil {
        return fmt.Errorf("Error syncing repository: %w", err)
    }

    // Print the returned message
    fmt.Println(message)
    return nil
}

func addRepo(remoteURL string, apiKey *string, force bool, config utils.Config, headCommitHash string, useMockLLM bool) error {
    log.Printf("[DEBUG] cli.addRepo: Received useMockLLM = %v (Note: This function currently does not seem to use this flag for the AddRepository API call)\n", useMockLLM) // DEBUG PRINT
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
