package utils

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/turboSource-marialis/machtiani/internal/git"
	"gopkg.in/yaml.v2"
)

// Constants for Environment Variable Names
const (
	EnvPrefix            = "MACHTIANI_"
	EnvModelAPIKey       = "MODEL_API_KEY"
	EnvModelAPIKeyOther  = "MODEL_API_KEY_OTHER"
	EnvModelBaseURL      = "MODEL_BASE_URL"
	EnvModelBaseURLOther = "MODEL_BASE_URL_OTHER"

	// **Prefixed names kept only for backward compatibility**
	EnvModelAPIKeyPrefixed       = "MACHTIANI_MODEL_API_KEY"
	EnvModelAPIKeyOtherPrefixed  = "MACHTIANI_MODEL_API_KEY_OTHER"
	EnvModelBaseURLPrefixed      = "MACHTIANI_MODEL_BASE_URL"
	EnvModelBaseURLOtherPrefixed = "MACHTIANI_MODEL_BASE_URL_OTHER"

	EnvMachtianiURL        = EnvPrefix + "URL" // Note: Adjusted name for consistency if MACHTIANI_URL is intended
	EnvRepoManagerURL      = EnvPrefix + "REPO_MANAGER_URL"
	EnvCodeHostURL         = "CODE_HOST_URL"
	EnvCodeHostAPIKey      = "CODE_HOST_API_KEY"
	EnvAPIGatewayHostKey   = EnvPrefix + "API_GATEWAY_HOST_KEY"
	EnvAPIGatewayHostValue = EnvPrefix + "API_GATEWAY_HOST_VALUE"
	EnvContentTypeKey      = EnvPrefix + "CONTENT_TYPE_KEY"
	EnvContentTypeValue    = EnvPrefix + "CONTENT_TYPE_VALUE"
	// Add other env var names here if needed
)

// EnsureDirExists creates a directory if it doesn't already exist.
func EnsureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory does not exist, create it
		err = os.MkdirAll(dirPath, 0755) // Use MkdirAll to create parent dirs if needed
		if err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	} else if err != nil {
		// Another error occurred when checking the directory
		return fmt.Errorf("failed to check directory status %s: %w", dirPath, err)
	}
	// Directory exists or was created successfully
	return nil
}

func CreateTempMarkdownFile(content string, filename string) (string, error) {
	// Define the directory where files will be saved
	chatDir := ".machtiani/chat"

	// Check if the directory exists, create if it doesn't
	if _, err := os.Stat(chatDir); os.IsNotExist(err) {
		if err := os.MkdirAll(chatDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
	}

	// Create a markdown file with the provided filename
	tempFile := fmt.Sprintf("%s/%s.md", chatDir, filename)
	if err := ioutil.WriteFile(tempFile, []byte(content), 0644); err != nil {
		return "", err
	}

	return tempFile, nil
}

var dryRun bool

// SetDryRun sets the dry-run state.
func SetDryRun(state bool) {
	dryRun = state
}

// IsDryRunEnabled returns true if dry-run mode is enabled.
func IsDryRunEnabled() bool {
	return dryRun
}

type Config struct {
	Environment struct {
		ModelAPIKey         string `yaml:"MODEL_API_KEY"`
		ModelAPIKeyOther    string `yaml:"MODEL_API_KEY_OTHER"`
		ModelBaseURL        string `yaml:"MODEL_BASE_URL"`
		ModelBaseURLOther   string `yaml:"MODEL_BASE_URL_OTHER"`
		MachtianiURL        string `yaml:"MACHTIANI_URL"`
		RepoManagerURL      string `yaml:"MACHTIANI_REPO_MANAGER_URL"`
		CodeHostURL         string `yaml:"CODE_HOST_URL"`
		CodeHostAPIKey      string `yaml:"CODE_HOST_API_KEY"`
		APIGatewayHostKey   string `yaml:"API_GATEWAY_HOST_KEY"`
		APIGatewayHostValue string `yaml:"API_GATEWAY_HOST_VALUE"`
		ContentTypeKey      string `yaml:"CONTENT_TYPE_KEY"`
		ContentTypeValue    string `yaml:"CONTENT_TYPE_VALUE"`
	} `yaml:"environment"`
}

// loadConfigFromFile attempts to load configuration from a specific file path.
// It returns the loaded config and a boolean indicating if the file was found and loaded.
func loadConfigFromFile(filePath string) (Config, bool, error) {
	var config Config
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, false, nil // File not found is not an error here, just return false
		}
		return config, false, fmt.Errorf("failed to read config file %s: %w", filePath, err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, false, fmt.Errorf("failed to unmarshal config from %s: %w", filePath, err)
	}
	return config, true, nil
}

// overrideConfig merges the 'override' config onto the 'base' config.
// Non-empty string values in 'override.Environment' will replace corresponding values in 'base.Environment'.
func overrideConfig(base *Config, override Config) {
	// Environment overrides
	if override.Environment.ModelAPIKey != "" {
		base.Environment.ModelAPIKey = override.Environment.ModelAPIKey
	}
	if override.Environment.ModelAPIKeyOther != "" {
		base.Environment.ModelAPIKeyOther = override.Environment.ModelAPIKeyOther
	}
	if override.Environment.ModelBaseURL != "" {
		base.Environment.ModelBaseURL = override.Environment.ModelBaseURL
	}
	if override.Environment.ModelBaseURLOther != "" {
		base.Environment.ModelBaseURLOther = override.Environment.ModelBaseURLOther
	}
	if override.Environment.MachtianiURL != "" {
		base.Environment.MachtianiURL = override.Environment.MachtianiURL
	}
	if override.Environment.RepoManagerURL != "" {
		base.Environment.RepoManagerURL = override.Environment.RepoManagerURL
	}
	if override.Environment.CodeHostURL != "" {
		base.Environment.CodeHostURL = override.Environment.CodeHostURL
	}
	if override.Environment.CodeHostAPIKey != "" {
		base.Environment.CodeHostAPIKey = override.Environment.CodeHostAPIKey
	}
	if override.Environment.APIGatewayHostKey != "" {
		base.Environment.APIGatewayHostKey = override.Environment.APIGatewayHostKey
	}
	if override.Environment.APIGatewayHostValue != "" {
		base.Environment.APIGatewayHostValue = override.Environment.APIGatewayHostValue
	}
	if override.Environment.ContentTypeKey != "" {
		base.Environment.ContentTypeKey = override.Environment.ContentTypeKey
	}
	if override.Environment.ContentTypeValue != "" {
		base.Environment.ContentTypeValue = override.Environment.ContentTypeValue
	}
	// Add more fields here if the Config struct grows
}

// loadConfigFromEnv overrides the given config struct with values from environment variables.
func loadConfigFromEnv(config *Config) {
	// 1) Primary API key: if set, apply it and clear the "Other" slot
	if v := firstNonEmpty(os.Getenv(EnvModelAPIKey), os.Getenv(EnvModelAPIKeyPrefixed)); v != "" {
		config.Environment.ModelAPIKey = v
		config.Environment.ModelAPIKeyOther = ""
	}
	// 2) Primary Base URL: if set, apply it and clear the "Other" slot
	if v := firstNonEmpty(os.Getenv(EnvModelBaseURL), os.Getenv(EnvModelBaseURLPrefixed)); v != "" {
		config.Environment.ModelBaseURL = v
		config.Environment.ModelBaseURLOther = ""
	}

	// 3) Now pick up any explicit "OTHER" overrides
	if v := firstNonEmpty(os.Getenv(EnvModelAPIKeyOther), os.Getenv(EnvModelAPIKeyOtherPrefixed)); v != "" {
		config.Environment.ModelAPIKeyOther = v
	}
	if v := firstNonEmpty(os.Getenv(EnvModelBaseURLOther), os.Getenv(EnvModelBaseURLOtherPrefixed)); v != "" {
		config.Environment.ModelBaseURLOther = v
	}

	// 4) Everything else, including CodeHost, picks up both prefixed and unprefixed
	if v := os.Getenv(EnvMachtianiURL); v != "" {
		config.Environment.MachtianiURL = v
	}
	if v := os.Getenv(EnvRepoManagerURL); v != "" {
		config.Environment.RepoManagerURL = v
	}
	if v := os.Getenv(EnvCodeHostURL); v != "" {
		config.Environment.CodeHostURL = v
	}
	// <-- here’s the key change: look for unprefixed first, then prefixed
	if v := os.Getenv(EnvCodeHostAPIKey); v != "" {
		config.Environment.CodeHostAPIKey = v
	}
	if v := os.Getenv(EnvAPIGatewayHostKey); v != "" {
		config.Environment.APIGatewayHostKey = v
	}
	if v := os.Getenv(EnvAPIGatewayHostValue); v != "" {
		config.Environment.APIGatewayHostValue = v
	}
	if v := os.Getenv(EnvContentTypeKey); v != "" {
		config.Environment.ContentTypeKey = v
	}
	if v := os.Getenv(EnvContentTypeValue); v != "" {
		config.Environment.ContentTypeValue = v
	}
}

// LoadConfig reads the configuration using the priority:
// 1. Environment Variables (prefixed with MACHTIANI_)
// 2. Local config file (.machtiani-config.yml in the current directory)
// 3. Global config file (~/.machtiani-config.yml)
func LoadConfig() (Config, error) {
	var finalConfig Config // Start with an empty config

	// 1. Load Global Config (Lowest Priority)
	homeDir, homeErr := os.UserHomeDir()
	if homeErr != nil {
		// Log warning but continue, global config is optional
		log.Printf("Warning: failed to get home directory, cannot load global config: %v", homeErr)
	} else {
		globalConfigPath := filepath.Join(homeDir, ".machtiani-config.yml")
		globalConfig, found, err := loadConfigFromFile(globalConfigPath)
		if err != nil {
			return finalConfig, fmt.Errorf("error processing global config file %s: %w", globalConfigPath, err)
		}
		if found {
			finalConfig = globalConfig // Use global as the base
		}
	}

	// 2. Load Local Config (Middle Priority)
	localConfigPath := ".machtiani-config.yml"
	localConfig, found, err := loadConfigFromFile(localConfigPath)
	if err != nil {
		return finalConfig, fmt.Errorf("error processing local config file %s: %w", localConfigPath, err)
	}
	if found {
		// Override global config values with local ones
		overrideConfig(&finalConfig, localConfig)
	}

	// 3. Load Environment Variables (Highest Priority)
	loadConfigFromEnv(&finalConfig)

	// Apply Defaults if values are still empty
	// Determine the LLM Model Base URL, defaulting if not set after all overrides
	if finalConfig.Environment.ModelBaseURL == "" {
		finalConfig.Environment.ModelBaseURL = "https://api.openai.com/v1"
	}
	// Add other defaults here if needed

	// Final check - ensure essential variables are present if required, or return error
	// Example: if finalConfig.Environment.MachtianiURL == "" {
	//	 return finalConfig, errors.New("MACHTIANI_URL or MachtianiURL in config is required but not set")
	// }

	return finalConfig, nil
}

func LoadConfigAndIgnoreFiles() (Config, []string, error) {
	config, err := LoadConfig()
	if err != nil {
		return config, nil, fmt.Errorf("error loading config: %w", err)
	}

	ignoreFilePath := ".machtiani.ignore"
	ignoreFiles, err := ReadIgnoreFile(ignoreFilePath)
	if err != nil {
		return config, nil, fmt.Errorf("error reading ignore file: %w", err)
	}
	if ignoreFiles == nil {
		ignoreFiles = []string{} // Default to empty list if nil
	}

	return config, ignoreFiles, nil
}

// ReadIgnoreFile reads a `machtiani.ignore》 file and returns a list of file paths
func ReadIgnoreFile(fileName string) ([]string, error) {
	var filePaths []string

	// Open the file
	file, err := os.Open(fileName)
	if os.IsNotExist(err) {
		// If the file does not exist, return an empty slice
		return filePaths, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", fileName, err)
	}
	defer file.Close()

	// Read the file line-by-line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Ignore empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Append valid file paths to the list
		filePaths = append(filePaths, line)
	}

	// Check for scanning errors
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", fileName, err)
	}

	return filePaths, nil
}

func GetCodeHostAPIKey(config Config) *string {
	if config.Environment.CodeHostAPIKey != "" {
		return &config.Environment.CodeHostAPIKey
	}
	return nil
}

func ParseFlags(fs *flag.FlagSet, args []string) {
	err := fs.Parse(args)
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}
}

func GetProjectOrDefault(projectFlag *string) (string, error) {
	if *projectFlag == "" {

		return git.GetProjectName()
	}
	return *projectFlag, nil
}

func ValidateFlags(modelFlag, matchStrengthFlag, modeFlag *string) {
	//model := *modelFlag
	//if model != "gpt-4o" && model != "gpt-4o-mini" {
	//    log.Fatalf("Error: Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")
	//}

	matchStrength := *matchStrengthFlag
	if matchStrength != "high" && matchStrength != "mid" && matchStrength != "low" {
		log.Fatalf("Error: Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")
	}

	mode := *modeFlag
	if mode != "chat" && mode != "pure-chat" && mode != "default" && mode != "answer-only" {
		log.Fatalf("Error: Invalid mode selected. Choose either chat, pure-chat, answer-only, or default.")
	}
}

// ValidateArgFormat checks arguments for common CLI usage errors like missing -- prefix
func ValidateArgFormat(fs *flag.FlagSet, args []string) error {
	// Before parsing, check for arguments that might be missing the -- prefix
	for i := 0; i < len(args); i++ {
		arg := args[i]
		// Skip if argument already has - or -- prefix
		if strings.HasPrefix(arg, "-") {
			continue
		}

		// Check if this matches a defined flag name
		var flagExists bool
		fs.VisitAll(func(f *flag.Flag) {
			if f.Name == arg {
				flagExists = true
			}
		})

		if flagExists {
			return fmt.Errorf("invalid flag format: '%s'. Did you mean '--%s'?", arg, arg)
		}
	}

	// If validation passes, continue with normal parsing
	return fs.Parse(args)
}

// ValidateAmplifyFlag validates the amplification level
func ValidateAmplifyFlag(value string) error {
	validValues := map[string]bool{
		"off":  true,
		"low":  true,
		"mid":  true,
		"high": true,
	}

	if !validValues[value] {
		return fmt.Errorf("invalid value for --amplify: '%s'. Must be one of: off, low, mid, high", value)
	}

	return nil
}

// ValidateDepthFlag validates the depth parameter
func ValidateDepthFlag(value int) error {
	if value <= 0 {
		return fmt.Errorf("invalid value for --depth: %d. Must be a positive integer", value)
	}

	return nil
}

// ValidateHeadCommitExistsOnRemote checks if the HEAD commit exists on any branch of the origin remote.
// Returns nil if the commit is found on at least one remote branch,
// error if it's not found on any remote branch or if validation fails.
func ValidateHeadCommitExistsOnRemote(headCommitHash string) error {
	// Fetch from origin to ensure we have up-to-date information about its branches.
	// This is necessary to accurately check if the commit exists on any branch of origin.
	fetchCmd := exec.Command("git", "fetch", "origin", "--quiet")
	if fetchOutput, fetchErr := fetchCmd.CombinedOutput(); fetchErr != nil {
		// It's important to include the command output in the error message for debugging
		return fmt.Errorf("failed to fetch from origin: %w, output: %s",
			fetchErr, strings.TrimSpace(string(fetchOutput)))
	}

	// Check if the headCommitHash exists on any branch of origin.
	// `git branch -r --contains <commit>` lists all remote-tracking branches that contain the specified commit.
	// We filter for only origin branches using grep
	checkCmd := exec.Command("sh", "-c", fmt.Sprintf("git branch -r --contains %s | grep '^  origin/'", headCommitHash))
	output, err := checkCmd.CombinedOutput() // Use CombinedOutput to capture both stdout and stderr

	// Trim whitespace from the output
	outputStr := strings.TrimSpace(string(output))

	// git branch --contains will exit with status 0 if the commit is found
	// on at least one branch, and non-zero (typically 1) if it's not found on any.
	// If there's an error *and* the output string is empty, it likely means
	// the commit was not found, which is the condition we're testing for.
	// If there's an error *and* there IS output, something else went wrong.
	if err != nil && outputStr != "" {
		// An error occurred that isn't just the commit not being found.
		return fmt.Errorf("failed to check origin branches for commit %s: %w, output: %s",
			headCommitHash, err, outputStr)
	}
	// If err is not nil but outputStr is empty, the next check will handle it.
	// If err is nil, outputStr contains the list of branches (or is empty).

	// If the output string is empty after trimming, it means no origin branch contains the commit.
	if outputStr == "" {
		// This is the validation failure case: the commit does not exist on any origin branch.
		return fmt.Errorf("local commit %s does not exist on any origin branch", headCommitHash)
	}

	// If we reach here, outputStr is not empty, meaning at least one origin branch contains the commit.
	// The validation passes.
	// Optional: Log the branches found for debugging/information.
	LogIfNotAnswerOnly(false, "Commit %s found on the following origin branches:\n%s", headCommitHash, outputStr) // Assuming !answerOnly for this info log

	return nil // Success: commit found on at least one origin branch
}

// ParseFlagsWithValidation combines argument format validation with flag parsing
func ParseFlagsWithValidation(fs *flag.FlagSet, args []string) error {
	return ValidateArgFormat(fs, args)
}

func Spinner(done chan bool) {
	symbols := []rune{'⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷'}
	i := 0
	hotPink := "\033[38;5;205m" // Updated to hot pink (256-color mode)
	reset := "\033[0m"

	fmt.Println()

	for {
		select {
		case <-done:
			// Clear the spinner by overwriting with a space and carriage return
			fmt.Print("\r \r")
			return
		default:
			fmt.Printf("\r%s%c%s", hotPink, symbols[i], reset)
			i = (i + 1) % len(symbols)
			time.Sleep(100 * time.Millisecond) // Adjust the spinner speed here
		}
	}
}

// GetCodehostURLFromCurrentRepository retrieves the codehost URL from the current Git repository.
func GetCodehostURLFromCurrentRepository() (string, error) {
	// Run the `git remote get-url origin` command to get the URL of the origin remote.
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("error getting git remote URL: %w", err)
	}

	// Convert output to string and trim whitespace
	codehostURL := strings.TrimSpace(string(output))
	return codehostURL, nil
}

// confirmProceed prompts the user for confirmation to proceed
func ConfirmProceed() bool {
	var response string
	fmt.Print("Do you wish to proceed? (y/n): ")
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y"
}

// FormatIntWithCommas returns an int as a string with commas, e.g. 12345 -> "12,345"
func FormatIntWithCommas(n int) string {
	s := strconv.Itoa(n)
	if len(s) <= 3 {
		return s
	}
	neg := false
	if n < 0 {
		neg = true
		s = s[1:]
	}
	var out []byte
	pre := len(s) % 3
	if pre > 0 {
		out = append(out, s[:pre]...)
	}
	for i := pre; i < len(s); i += 3 {
		if len(out) > 0 {
			out = append(out, ',')
		}
		out = append(out, s[i:i+3]...)
	}
	if neg {
		return "-" + string(out)
	}
	return string(out)
}

// IsAnswerOnlyMode checks if the application is running in answer-only mode
// by examining command line arguments
func IsAnswerOnlyMode() bool {
	for i, arg := range os.Args {
		if arg == "--mode" && i+1 < len(os.Args) && os.Args[i+1] == "answer-only" {
			return true
		}
		if strings.HasPrefix(arg, "--mode=answer-only") {
			return true
		}
	}
	return false
}

// PrintIfNotAnswerOnly prints the formatted message only if not in answer-only mode
//
// Example:
//
//	isAnswerOnlyMode := IsAnswerOnlyMode()
//	PrintIfNotAnswerOnly(isAnswerOnlyMode, "Using remote URL: %s\n", remoteURL)
func PrintIfNotAnswerOnly(isAnswerOnly bool, format string, args ...interface{}) {
	if !isAnswerOnly {
		fmt.Printf(format, args...)
	}
}

// LogIfNotAnswerOnly logs the formatted message only if not in answer-only mode
//
// Example:
//
//	LogIfNotAnswerOnly(isAnswerOnlyMode, "Warning: failed to parse system message frequency: %v", err)
func LogIfNotAnswerOnly(isAnswerOnly bool, format string, args ...interface{}) {
	if !isAnswerOnly {
		log.Printf(format, args...)
	}
}

// LogErrorIfNotAnswerOnly logs an error message only if the error is not nil and not in answer-only mode
//
// Example:
//
//	if err := git.SaveSystemMessage(systemMsg); err != nil {
//	    LogErrorIfNotAnswerOnly(isAnswerOnlyMode, err, "Failed to save system message")
//	}
func LogErrorIfNotAnswerOnly(isAnswerOnly bool, err error, message string) {
	if err != nil && !isAnswerOnly {
		log.Printf("%s: %v", message, err)
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
