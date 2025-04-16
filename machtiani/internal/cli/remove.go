package cli

import (
	"fmt"
	"log"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/utils"
)

var RepoManagerURL string = "http://localhost:5070"


func handleRemove(remoteURL string, projectName string, vcsType string, apiKey *string, forceFlag bool, config utils.Config) {
	response, err := api.DeleteStore(projectName, remoteURL, vcsType, apiKey, RepoManagerURL, forceFlag)
	if err != nil {
		log.Fatalf("Error removing store: %v", err)
	}

	fmt.Println(response.Message)
}
