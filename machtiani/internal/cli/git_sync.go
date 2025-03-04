package cli

import (
    "fmt"
    "strings"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL, branchName string, apiKey *string, force bool, config utils.Config) error {
    if remoteURL == "" || branchName == "" {
        return fmt.Errorf("Error: all flags --remote and --branch-name must be provided.")
    }


    _, err := api.CheckStatus(remoteURL, apiKey)
    if err != nil {
        if strings.Contains(err.Error(), "does not exist") {
            // If the repository doesn't exist, add it
            if err := addRepo(remoteURL, apiKey, force, config); err != nil {
                return fmt.Errorf("Error adding repository: %w", err)
            } else {
                return nil
            }
        } else {
            return fmt.Errorf("Error checking repository status: %w", err)
        }
    }

    message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force)
    if err != nil {
        return fmt.Errorf("Error syncing repository: %w", err)
    }

    // Print the returned message
    fmt.Println(message)
    return nil
}

func addRepo(remoteURL string, apiKey *string, force bool, config utils.Config) error {
    response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, api.RepoManagerURL, force)
    if err != nil {
        return fmt.Errorf("Error adding repository: %w", err)
    }

    fmt.Println(response.Message)
    fmt.Println("---")
    fmt.Println("Your repo is getting added to machtiani is in progress!")
    fmt.Println("Please check back by running `machtiani status` to see if it completed.")
    return nil
}
