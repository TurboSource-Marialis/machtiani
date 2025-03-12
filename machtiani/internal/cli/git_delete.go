package cli

import (
	"fmt"
	"log"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/utils"
)

var RepoManagerURL string = "http://localhost:5070"

func handleGitDelete(remoteURL string, projectName string, vcsType string, apiKey *string, forceFlag bool, config utils.Config) {
	// Call the updated DeleteStore function
	response, err := api.DeleteStore(projectName, remoteURL, vcsType, apiKey, RepoManagerURL, forceFlag)
	if err != nil {
		log.Fatalf("Error deleting store: %v", err)
	}

	fmt.Println(response.Message)
}
