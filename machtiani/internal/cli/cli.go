package cli

import (
    "flag"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
    "github.com/7db9a/machtiani/internal/git"
)

func Execute() {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
    remoteName := fs.String("remote", "origin", "Name of the remote repository")
    branchName := fs.String("branch-name", "", "Branch name")
    forceFlag := fs.Bool("force", false, "Skip confirmation prompt and proceed with the operation.")

    compatible, message, err := api.GetInstallInfo()
    if err != nil {
        log.Printf("Error getting install info: %v", err)
        os.Exit(1)
    }

    if !compatible {
        log.Printf("This CLI is no longer compatible with the current environment. Please update to the latest version by following the below instructions\n\n%v", message)
        os.Exit(1)
    }


    // Use the new remote URL function
    remoteURL, err := git.GetRemoteURL(remoteName)
    if err != nil {
        log.Printf("Error getting remote url: %v", err)
        os.Exit(1)
    }
    fmt.Printf("Using remote URL: %s\n", remoteURL)
    projectName :=  remoteURL

    var apiKey *string = utils.GetCodeHostAPIKey(config)

    // Check if no command is provided
    if len(os.Args) < 2 {
        printHelp()
        return // Exit after printing help
    }

    command := os.Args[1]
    switch command {
    case "status":
        handleStatus(&config, remoteURL, apiKey)
        return // Exit after handling status
    case "git-store":
        // Parse flags for git-store
        utils.ParseFlags(fs, os.Args[2:]) // Use the new helper function
        // Call the new function to handle git-store
        handleGitStore(remoteURL, apiKey, *forceFlag, config)
        return // Exit after handling git-store
    case "git-sync":
        utils.ParseFlags(fs, os.Args[2:]) // Use the new helper function
        // Call the HandleGitSync function
        if err := handleGitSync(remoteURL, *branchName, apiKey, *forceFlag, config); err != nil {
            log.Printf("Error handling git-sync: %v", err)
            os.Exit(1)
        }
        return
    case "git-delete":
        utils.ParseFlags(fs, os.Args[2:]) // Use the new helper function
        if remoteURL == "" {
            log.Printf("Error: --remote must be provided.")
            os.Exit(1)
        }
        // Define additional parameters for git-delete
        ignoreFiles := []string{} // Populate this list as needed
        vcsType := "git"          // Set the VCS type as needed
        openaiAPIKey := config.Environment.ModelAPIKey // Adjust as necessary
        // Call the handleGitDelete function
        handleGitDelete(remoteURL, projectName, ignoreFiles, vcsType, apiKey, &openaiAPIKey, *forceFlag, config)
        return
    case "help":
        printHelp()
        return // Exit after printing help
    default:
        startTime := time.Now() // Start the timer here
        args := os.Args[1:]
        handlePrompt(args, &config, &remoteURL, apiKey)
        duration := time.Since(startTime)
        fmt.Printf("Total response handling took %s\n", duration) // Print total duration
        return
    }
}

