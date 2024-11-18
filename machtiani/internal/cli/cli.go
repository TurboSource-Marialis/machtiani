package cli

import (
    "flag"
    "fmt"
    "log"
    "os"
    "os/exec"
    "context"
    "time"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
    "github.com/7db9a/machtiani/internal/git"
    "github.com/sashabaranov/go-openai" // Ensure you import the OpenAI package
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

func float32ToFloat64(input []float32) []float64 {
    output := make([]float64, len(input))
    for i, v := range input {
        output[i] = float64(v)
    }
    return output
}


func generateEmbeddings(apiKey, prompt string) ([]float64, error) {
    client := openai.NewClient(apiKey)
    req := openai.EmbeddingRequest{
        Model: "text-embedding-3-large",
        Input: []string{prompt},
    }
    resp, err := client.CreateEmbeddings(context.Background(), req)
    if err != nil {
        return nil, fmt.Errorf("failed to create embeddings: %w", err)
    }

    if len(resp.Data) > 0 {
        return float32ToFloat64(resp.Data[0].Embedding), nil
    }
    return nil, fmt.Errorf("no embeddings returned")
}

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
        log.Fatalf("Error checking HEAD OIDs match: %v", err)
    }

    if !compatible {
        log.Fatalf("This CLI is no longer compatible with the current environment. Please update to the latest version by following the below instructions\n\n%v", message)
    }


    // Use the new remote URL function
    remoteURL, err := git.GetRemoteURL(remoteName)
    if err != nil {
        log.Fatalf(err.Error())
    }
    fmt.Printf("Using remote URL: %s\n", remoteURL)
    projectName :=  remoteURL

    var apiKey *string = utils.GetAPIKey(config)

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
        parseFlags(fs, os.Args[2:]) // Use the new helper function
        // Call the new function to handle git-store
        handleGitStore(remoteURL, apiKey, *forceFlag, config)
        return // Exit after handling git-store
    case "git-sync":
        parseFlags(fs, os.Args[2:]) // Use the new helper function
        // Call the HandleGitSync function
        if err := handleGitSync(remoteURL, *branchName, apiKey, *forceFlag, config); err != nil {
            log.Fatalf("Error handling git-sync: %v", err)
        }
        return
    case "git-delete":
        parseFlags(fs, os.Args[2:]) // Use the new helper function
        if remoteURL == "" {
            log.Fatal("Error: --remote must be provided.")
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

// handleError prints the error message and exits the program.
func handleError(message string) {
    fmt.Fprintf(os.Stderr, "%s\n", message)
    os.Exit(1)
}

func parseFlags(fs *flag.FlagSet, args []string) {
    err := utils.ParseFlags(fs, args) // Parse flags after the command
    if err != nil {
        log.Fatalf("Error parsing flags: %v", err)
    }
}

// runAicommit generates a commit message using aicommit and lets it perform the git commit.
func runAicommit(args []string) {
    config, err := utils.LoadConfig()
    if err != nil {
        log.Fatalf("Error loading config: %v", err)
    }

    // Define flags specific to aicommit
    fs := flag.NewFlagSet("aicommit", flag.ExitOnError)
    openaiKey := fs.String("openai-key", config.Environment.ModelAPIKey, "OpenAI API Key")
    modelFlag := fs.String("model", "gpt-4o-mini", "Model to use for generating messages")
    amend := fs.Bool("amend", false, "Amend the last commit instead of creating a new one")
    context := fs.String("context", "", "Additional context for generating the commit message")

    // Parse the provided arguments
    err = fs.Parse(args)
    if err != nil {
        handleError(fmt.Sprintf("Error parsing flags: %v", err))
    }

    // Construct aicommit arguments without --dry
    aicommitArgs := []string{
        "--openai-key", *openaiKey,
        "--model", *modelFlag,
    }
    if *amend {
        aicommitArgs = append(aicommitArgs, "--amend")
    }
    if *context != "" {
        aicommitArgs = append(aicommitArgs, "--context", *context)
    }

    // Handle dry-run mode by adding --dry to aicommit arguments if needed
    if utils.IsDryRunEnabled() {
        aicommitArgs = append(aicommitArgs, "--dry")
    }

    // Locate the aicommit binary
    binaryPath, err := exec.LookPath("aicommit")
    if err != nil {
        handleError(fmt.Sprintf("aicommit binary not found in PATH: %v", err))
    }

    // Create the command to run aicommit
    cmd := exec.Command(binaryPath, aicommitArgs...)

    // Set the working directory to the current directory
    cwd, err := os.Getwd()
    if err != nil {
        handleError(fmt.Sprintf("Failed to get current working directory: %v", err))
    }
    cmd.Dir = cwd

    // Inherit environment variables
    cmd.Env = os.Environ()

    // Attach stdout and stderr to display aicommit output directly
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr

    // Execute the aicommit command
    err = cmd.Run()
    if err != nil {
        handleError(fmt.Sprintf("Error running aicommit: %v", err))
    }

    // No need to perform git commit manually; aicommit handles it
}

func printHelp() {
    helpText := `Usage: machtiani [flags] [prompt]

    Machtiani is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories.

    Commands:
      git-store                    Add a repository to the Machtiani system.
      git-sync                     Fetch and checkout a specific branch of the repository.
      git-delete                   Remove a repository from the Machtiani system.
      status                       Check the status of the current project.

    Global Flags:
      -file string                 Path to the markdown file (optional).
      -project string              Name of the project (optional).
      -model string                Model to use (options: gpt-4o, gpt-4o-mini; default: gpt-4o-mini).
      -match-strength string       Match strength (options: high, mid, low; default: mid).
      -mode string                 Search mode (options: pure-chat, commit, super; default: commit).
      --force                      Skip confirmation prompt and proceed with the operation.
      -verbose                     Enable verbose output.

    Subcommands:

    git-store:
      Usage: machtiani git-store --branch <default_branch> --remote <remote_name> [--force]
      Adds a repository to Machtiani system.
      Flags:
        --branch string            Name of the default branch (required).
        --remote string            Name of the remote repository (default: "origin").
        --force                    Skip confirmation prompt.

    git-sync:
      Usage: machtiani git-sync --branch <default_branch> --remote <remote_name> [--force]
      Syncs with a specific branch of the repository.
      Flags:
        --branch string            Name of the branch (required).
        --remote string            Name of the remote repository (default: "origin").
        --force                    Skip confirmation prompt.

    git-delete:
      Usage: machtiani git-delete --remote <remote_name> [--force]
      Removes a repository from Machtiani system.
      Flags:
        --remote string            Name of the remote repository (required).
        --force                    Skip confirmation prompt.

    Examples:
      Providing a direct prompt:
        machtiani "Add a new endpoint to get stats."

      Using an existing markdown chat file:
        machtiani --file .machtiani/chat/add_state_endpoint.md

      Specifying additional parameters:
        machtiani "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high

      Using the '--force' flag to skip confirmation:
        machtiani git-store --branch master --force

    `
    fmt.Println(helpText)
}

