package cli

import (
    "fmt"
    "log"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
)

func handleGitDelete(remoteURL string, projectName string, ignoreFiles []string, vcsType string, apiKey *string, openaiAPIKey *string, forceFlag bool, config utils.Config) {
    // Call the updated DeleteStore function
    response, err := api.DeleteStore(projectName, remoteURL, ignoreFiles, vcsType, apiKey, openaiAPIKey, config.Environment.RepoManagerURL, forceFlag)
    if err != nil {
        log.Fatalf("Error deleting store: %v", err)
    }

    fmt.Println(response.Message)
}
