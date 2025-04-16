package cli

import (
    "fmt"
)

func printHelp() {
    helpText := `Usage: mct [flags] [prompt]

    Machtiani is a command-line interface (CLI) tool designed to facilitate code chat and information retrieval from code repositories.


    Commands:
      sync                        Add or sync a repository with the Machtiani system.
      remove                      Remove a repository from the Machtiani system.
      status                      Check the status of the current project.

    Global Flags:
      -file string                 Path to the markdown file (optional).
      -project string              Name of the project (optional).
      -model string                Model to use (options: gpt-4o, gpt-4o-mini; default: gpt-4o-mini).
      -match-strength string       Match strength (options: high, mid, low; default: mid).
      -mode string                 Search mode (options: pure-chat, commit, super; default: commit).
      --force                      Skip confirmation prompt and proceed with the operation.
      -verbose                     Enable verbose output.

    Subcommands:


    sync:
      Usage: mct sync --remote <remote_name> [--force]
      Syncs a repository with Machtiani system.
      Flags:
        --remote string            Name of the remote repository (default: "origin").
        --force                    Skip confirmation prompt.

    remove:
      Usage: mct remove --remote <remote_name> [--force]
      Removes a repository from Machtiani system.
      Flags:
        --remote string            Name of the remote repository (required).
        --force                    Skip confirmation prompt.

    Examples:
      Providing a direct prompt:
        mct "Add a new endpoint to get stats."

      Using an existing markdown chat file:
        mct --file .machtiani/chat/add_state_endpoint.md

      Specifying additional parameters:
        mct "Add a new endpoint to get stats." --model gpt-4o --mode pure-chat --match-strength high


      Using the '--force' flag to skip confirmation:
        mct sync --force

    `
    fmt.Println(helpText)
}

