package cli

import (
    "fmt"
)

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

