package utils

import (
    "fmt"
    "io/ioutil"
    "os"

    "gopkg.in/yaml.v2"
)

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
        OpenAIAPIKey         string `yaml:"OPENAI_MACHTIANI_API_KEY"`
        MachtianiURL         string `yaml:"MACHTIANI_URL"`
        RepoManagerURL       string `yaml:"MACHTIANI_REPO_MANAGER_URL"`
    } `yaml:"environment"`
}

// LoadConfig reads the configuration from the YAML file
func LoadConfig() (Config, error) {
    var config Config
    data, err := ioutil.ReadFile("machtiani-config.yml")
    if err != nil {
        return config, fmt.Errorf("failed to read config.yaml: %w", err)
    }
    err = yaml.Unmarshal(data, &config)
    if err != nil {
        return config, fmt.Errorf("failed to unmarshal config.yaml: %w", err)
    }
    return config, nil
}
