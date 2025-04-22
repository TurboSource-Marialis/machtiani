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

	"github.com/7db9a/machtiani/internal/git"
	"gopkg.in/yaml.v2"
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

// LoadConfig reads the configuration from the YAML file and prioritizes the environment variable
func LoadConfig() (Config, error) {
	var config Config

	// First, try to load from the current directory
	configPath := ".machtiani-config.yml"
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		// If it doesn't exist, try to load from the home directory
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return config, fmt.Errorf("failed to get home directory: %w", homeErr)
		}
		configPath = filepath.Join(homeDir, ".machtiani-config.yml")
		data, err = ioutil.ReadFile(configPath)
		if err != nil {
			return config, fmt.Errorf("failed to read config from both locations: %w", err)
		}
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return config, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Check for the environment variable and prioritize it
	if envAPIKey := os.Getenv("MODEL_API_KEY"); envAPIKey != "" {
		config.Environment.ModelAPIKey = envAPIKey
	}

	// Determine the LLM Model Base URL, defaulting if not set
	llmModelBaseURL := config.Environment.ModelBaseURL
	if llmModelBaseURL == "" {
		config.Environment.ModelBaseURL = "https://api.openai.com/v1"
	}

	return config, nil
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
	if mode != "chat" && mode != "pure-chat" && mode != "default" {
		log.Fatalf("Error: Invalid mode selected. Choose either chat, pure-chat, or default.")
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
	log.Printf("Commit %s found on the following origin branches:\n%s", headCommitHash, outputStr)

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
