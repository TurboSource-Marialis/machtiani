package cli

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"
	"strconv"

	"github.com/7db9a/machtiani/internal/api"
	"github.com/7db9a/machtiani/internal/git"
	"github.com/7db9a/machtiani/internal/utils"
)

// Build-time variable for system message frequency
var SystemMessageFrequencyHours = "24" // Default 24 hours, will be set via ldflags

func Execute() {
	// Parse the system message frequency
	frequencyHours, err := strconv.Atoi(SystemMessageFrequencyHours)
	if err != nil {
		frequencyHours = 24 // Default to 24 hours if parsing fails
		log.Printf("Warning: failed to parse system message frequency, using default 24 hours: %v", err)
	}

	// Check if we should display the system message
	shouldDisplay, err := git.ShouldDisplaySystemMessage(frequencyHours)
	if err != nil {
		log.Printf("Warning: error checking if system message should be displayed: %v", err)
		// Default to displaying the message if there's an error
		shouldDisplay = true
	}


	if shouldDisplay {
		// Try to fetch and display the system message
		systemMsg, err := git.GetLatestMachtianiSystemMessage(api.MachtianiGitRemoteURL)
		if err == nil && systemMsg != "" {
			// Check if the message is different from the last one shown
			lastMsg, err := git.GetLastSystemMessage()
			if err != nil {
				log.Printf("Warning: failed to read last system message: %v", err)
				lastMsg = "" // Continue with displaying the message
			}

			// Only show if the message is different
			if systemMsg != lastMsg {
				fmt.Printf("\n============= SYSTEM MESSAGE =============\n%s\n=========================================\n\n", systemMsg)

				// Save the new message as the last shown
				if err := git.SaveSystemMessage(systemMsg); err != nil {
					log.Printf("Warning: failed to save system message: %v", err)
				}
			}

			// Record that we checked the message, regardless of whether we display it
			if err := git.RecordSystemMessageDisplayed(); err != nil {
				log.Printf("Warning: failed to record system message display time: %v", err)
			}
		} else if err != nil {
			// Log the error but don't show it to the user
			log.Printf("Failed to fetch system message: %v", err)
		}
	}

	config, err := utils.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
	remoteName := fs.String("remote", "origin", "Name of the remote repository")
	forceFlag := fs.Bool("force", false, "Skip confirmation prompt and proceed with the operation.")
	verboseFlag := fs.Bool("verbose", false, "Print verbose output including timing information.")
	// Updated cost flag description
	costFlag := fs.Bool("cost", false, "Estimate token cost before proceeding with the sync.")
	// Added cost-only flag
	costOnlyFlag := fs.Bool("cost-only", false, "Estimate token cost and exit without performing the sync.")
    amplifyFlag := fs.String("amplify", "off", "Amplification level (off, low, mid, high), default is off when flag is present")
    depthFlag := fs.Int("depth", 10000, "Depth of commit history to fetch (integer, default 10000)") // Added depth flag

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
	projectName := remoteURL

	var apiKey *string = utils.GetCodeHostAPIKey(config)

	// Check if no command is provided
	if len(os.Args) < 2 {
		printHelp()
		return // Exit after printing help
	}

	command := os.Args[1]
	switch command {
	case "status":
		handleStatus(&config, remoteURL)
		return // Exit after handling status

	case "git-sync":

		// Use the new validation function
		err := utils.ValidateArgFormat(fs, os.Args[2:])
		if err != nil {
			fmt.Printf("Error in command arguments: %v\n", err)
			os.Exit(1)
		}

		// Validate amplify flag value
		if err := utils.ValidateAmplifyFlag(*amplifyFlag); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Validate depth flag value
		if err := utils.ValidateDepthFlag(*depthFlag); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		headCommitHash, err := git.GetHeadCommitHash()
		if err != nil {
			log.Printf("Error getting HEAD commit hash: %v", err) // Log error but continue
		}

		// Use validated parameters
		if err := handleGitSync(remoteURL, apiKey, *forceFlag, *verboseFlag, *costFlag, *costOnlyFlag, config, headCommitHash, *amplifyFlag, *depthFlag); err != nil {
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
		vcsType := "git" // Set the VCS type as needed
		// Call the handleGitDelete function
		handleGitDelete(remoteURL, projectName, vcsType, apiKey, *forceFlag, config)
		return
	case "help":
		printHelp()
		return // Exit after printing help
	default:
		startTime := time.Now() // Start the timer here
		args := os.Args[1:]
		headCommitHash, err := git.GetHeadCommitHash()
		if err != nil {
			log.Printf("Error getting HEAD commit hash: %v", err) // Log error but continue
		}
		handlePrompt(args, &config, &remoteURL, apiKey, headCommitHash)
		duration := time.Since(startTime)
		fmt.Printf("Total response handling took %s\n", duration) // Print total duration
		return
	}
}
