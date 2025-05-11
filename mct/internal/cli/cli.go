
package cli

import (
	"flag"
	"os"
	"time"
	"strconv"

	"github.com/tursomari/machtiani/mct/internal/api"
	"github.com/tursomari/machtiani/mct/internal/git"
	"github.com/tursomari/machtiani/mct/internal/utils"
)

// Build-time variable for system message frequency
var SystemMessageFrequencyHours = "24" // Default 24 hours, will be set via ldflags



func Execute() {
	// First, check if we're in answer-only mode early
	isAnswerOnlyMode := utils.IsAnswerOnlyMode()


	// Parse the system message frequency
	frequencyHours, err := strconv.Atoi(SystemMessageFrequencyHours)
	if err != nil {
		frequencyHours = 24 // Default to 24 hours if parsing fails
		utils.LogIfNotAnswerOnly(isAnswerOnlyMode, "Warning: failed to parse system message frequency, using default 24 hours: %v", err)
	}


	// Check if we should display the system message
	shouldDisplay, err := git.ShouldDisplaySystemMessage(frequencyHours)
	if err != nil {
		utils.LogIfNotAnswerOnly(isAnswerOnlyMode, "Warning: error checking if system message should be displayed: %v", err)
		// Default to displaying the message if there's an error
		shouldDisplay = true
	}




	if shouldDisplay && !isAnswerOnlyMode {
		// Try to fetch and display the system message
		systemMsg, err := git.GetLatestMachtianiSystemMessage(api.MachtianiGitRemoteURL)
		if err == nil && systemMsg != "" {
			// Check if the message is different from the last one shown
			lastMsg, err := git.GetLastSystemMessage()
			if err != nil {
				utils.LogIfNotAnswerOnly(isAnswerOnlyMode, "Warning: failed to read last system message: %v", err)
				lastMsg = "" // Continue with displaying the message
			}

			// Only show if the message is different
			if systemMsg != lastMsg {
				utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "\n============= SYSTEM MESSAGE =============\n%s\n=========================================\n\n", systemMsg)

				// Save the new message as the last shown
				if err := git.SaveSystemMessage(systemMsg); err != nil {
					utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Warning: failed to save system message")
				}
			}

			// Record that we checked the message, regardless of whether we display it
			if err := git.RecordSystemMessageDisplayed(); err != nil {
				utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Warning: failed to record system message display time")
			}
		} else if err != nil {
			// Log the error but don't show it to the user
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Failed to fetch system message")
		}
	}




	config, err := utils.LoadConfig()
	if err != nil {
		utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error loading config")
		os.Exit(1)
	}

	// Check for missing API key and base URL
	missingRequired := false
	var missingFields []string

	// Check for missing ModelAPIKey
	if config.Environment.ModelAPIKey == "" {
		missingFields = append(missingFields, "MCT_MODEL_API_KEY")
		missingRequired = true
	}

	// Check for missing ModelBaseURL
	if config.Environment.ModelBaseURL == "" {
		missingFields = append(missingFields, "MCT_MODEL_BASE_URL")
		missingRequired = true
	}

	// Check for missing MachtianiURL
	if api.MachtianiURL == "" {
		missingFields = append(missingFields, "MACHTIANI_URL")
		missingRequired = true
	}

	// Check for missing RepoManagerURL
	if api.RepoManagerURL == "" {
		missingFields = append(missingFields, "MACHTIANI_REPO_MANAGER_URL")
		missingRequired = true
	}


	// If any required fields are missing, display an error and exit
	if missingRequired {
		utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "Error: Missing required configuration.\n")

		// Display specific guidance based on missing fields
		for _, field := range missingFields {
			if field == "MCT_MODEL_API_KEY" {
				utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, `
Please add MCT_MODEL_API_KEY using your API provider api key, such as:

$ export MCT_MODEL_API_KEY=sk...

`)
			} else if field == "MCT_MODEL_BASE_URL" {
				utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, `
Please add MCT_MODEL_BASE_URL using your API provider base url, such as

$ export MCT_MODEL_BASE_URL="https://openrouter.ai/api/v1"

or

$ export MCT_MODEL_BASE_URL="https://api.openai.com/v1"
`)
			} else {
				// For other missing fields, use the generic message
				utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "  - %s (environment variable or config file entry)\n", field)
			}
		}
		os.Exit(1)
	}

	fs := flag.NewFlagSet("mct", flag.ContinueOnError)
	remoteName := fs.String("remote", "origin", "Name of the remote repository")
	forceFlag := fs.Bool("force", false, "Skip confirmation prompt and proceed with the operation.")
	verboseFlag := fs.Bool("verbose", false, "Print verbose output including timing information.")
	// Updated cost flag description
	costFlag := fs.Bool("cost", false, "Estimate token cost before proceeding with the sync.")
	// Added cost-only flag
	costOnlyFlag := fs.Bool("cost-only", false, "Estimate token cost and exit without performing the sync.")


    amplifyFlag := fs.String("amplify", "off", "Amplification level (off, low, mid, high), default is off when flag is present")
    depthFlag := fs.Int("depth", 10000, "Depth of commit history to fetch (integer, default 10000)") // Added depth flag
    modelFlag := fs.String("model", "", "Specify the model to use for this operation")
    modelThreadsFlag := fs.Int("model-threads", 0, "Number of threads for model processing (0 means use default)")



	compatible, message, err := api.GetInstallInfo()
	if err != nil {
		utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error getting install info")
		os.Exit(1)
	}

	if !compatible {
		utils.LogIfNotAnswerOnly(isAnswerOnlyMode, "This CLI is no longer compatible with the current environment. Please update to the latest version by following the below instructions\n\n%v", message)
		os.Exit(1)
	}



	// Use the new remote URL function
	remoteURL, err := git.GetRemoteURL(remoteName)
	if err != nil {
		utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error getting remote url")
		os.Exit(1)
	}
	utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "Using remote URL: %s\n", remoteURL)
	projectName := remoteURL

	var apiKey *string = utils.GetCodeHostAPIKey(config)


	// Check if no command is provided
	if len(os.Args) < 2 {
		printHelp()
		return // Exit after printing help
	}

	command := os.Args[1]

	// Handle help variants upfront
	if command == "help" || command == "--help" || command == "-h" {
		printHelp()
		return
	}

	switch command {
	case "status":
		handleStatus(&config, remoteURL)
		return // Exit after handling status



	case "sync":
		// Use the new validation function

		err := utils.ValidateArgFormat(fs, os.Args[2:])
		if err != nil {
			utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "Error in command arguments: %v\n", err)
			os.Exit(1)
		}

		// Validate amplify flag value
		if err := utils.ValidateAmplifyFlag(*amplifyFlag); err != nil {
			utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%v\n", err)
			os.Exit(1)
		}

		// Validate depth flag value
		if err := utils.ValidateDepthFlag(*depthFlag); err != nil {
			utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "%v\n", err)
			os.Exit(1)
		}


		headCommitHash, err := git.GetHeadCommitHash()
		if err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error getting HEAD commit hash")
		}




		if err := handleSync(remoteURL, apiKey, *forceFlag, *verboseFlag, *costFlag, *costOnlyFlag, config, headCommitHash, *amplifyFlag, *depthFlag, *modelFlag, *modelThreadsFlag); err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error handling sync")
			os.Exit(1)
		}
		return


	case "remove":
		utils.ParseFlags(fs, os.Args[2:])

		if remoteURL == "" {
			utils.LogIfNotAnswerOnly(isAnswerOnlyMode, "Error: --remote must be provided.")
			os.Exit(1)
		}
		vcsType := "git"
		handleRemove(remoteURL, projectName, vcsType, apiKey, *forceFlag, config)
		return
	case "help":
		printHelp()
		return // Exit after printing help

	default:
		startTime := time.Now() // Start the timer here
		args := os.Args[1:]

		headCommitHash, err := git.GetHeadCommitHash()
		if err != nil {
			utils.LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Error getting HEAD commit hash") // Log error but continue
		}
		handlePrompt(args, &config, &remoteURL, apiKey, headCommitHash)


		// Print duration
		duration := time.Since(startTime)
		utils.PrintIfNotAnswerOnly(isAnswerOnlyMode, "Total response handling took %s\n", duration) // Print total duration
		return
	}
}

