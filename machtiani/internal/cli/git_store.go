package cli

import (
    "fmt"
    "log"
    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
)

func handleGitStore(remoteURL string, apiKey *string, forceFlag bool, config utils.Config) {
    // Call the new function to add the repository
    response, err := api.AddRepository(remoteURL, remoteURL, apiKey, config.Environment.ModelAPIKey, config.Environment.RepoManagerURL, forceFlag)
    if err != nil {
        log.Fatalf("Error adding repository: %v", err)
    }

    fmt.Println(response.Message)
    // Print the success message
    fmt.Println("---")
    fmt.Println("Your repo is getting added to machtiani is in progress!")
    fmt.Println("Please check back by running `machtiani status` to see if it completed.")
}
