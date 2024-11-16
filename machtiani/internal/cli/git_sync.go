package cli

import (
    "fmt"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
)

func handleGitSync(remoteURL, branchName string, apiKey *string, force bool, config utils.Config) error {
    if remoteURL == "" || branchName == "" {
        return fmt.Errorf("Error: all flags --remote and --branch-name must be provided.")
    }

    // Call the function to fetch and checkout the branch
    message, err := api.FetchAndCheckoutBranch(remoteURL, remoteURL, branchName, apiKey, config.Environment.ModelAPIKey, force)
    if err != nil {
        return fmt.Errorf("Error syncing repository: %w", err)
    }

    // Print the returned message
    fmt.Println(message)
    return nil
}
