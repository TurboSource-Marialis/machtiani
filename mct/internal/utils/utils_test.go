// utils_test.go
package utils

import (
    "io/ioutil"
    "os"
    "testing"
    "strings"
)

func createTempConfigFile(content string) (string, error) {
    tempFile, err := ioutil.TempFile("", "machtiani-config-*.yml")
    if err != nil {
        return "", err
    }

    if _, err := tempFile.Write([]byte(content)); err != nil {
        return "", err
    }

    if err := tempFile.Close(); err != nil {
        return "", err
    }

    return tempFile.Name(), nil
}

func TestLoadConfig_BareMinimumValidConfig(t *testing.T) {
    validConfig := `
environment:
  MODEL_API_KEY: ""
  MACHTIANI_URL: "http://localhost:5071"
  MACHTIANI_REPO_MANAGER_URL: "http://localhost:5070"
  CODE_HOST_URL: "http://localhost:8080"
  CODE_HOST_API_KEY: ""
  API_GATEWAY_HOST_KEY: ""
  API_GATEWAY_HOST_VALUE: ""
  CONTENT_TYPE_KEY: "Content-Type"
  CONTENT_TYPE_VALUE: "application/json"
`
    tempFile, err := createTempConfigFile(validConfig)
    if err != nil {
        t.Fatalf("Failed to create temp config file: %v", err)
    }
    defer os.Remove(tempFile)

    // Backup original config if it exists
    originalConfigPath := "machtiani-config.yml"
    if _, err := os.Stat(originalConfigPath); !os.IsNotExist(err) {
        os.Rename(originalConfigPath, originalConfigPath+".bak") // Backup original
        defer os.Rename(originalConfigPath+".bak", originalConfigPath) // Restore original
    }

    // Copy the contents of tempFile to the original config path
    content, err := ioutil.ReadFile(tempFile)
    if err != nil {
        t.Fatalf("Failed to read temp config file: %v", err)
    }

    if err := ioutil.WriteFile(originalConfigPath, content, 0644); err != nil {
        t.Fatalf("Failed to set temp config file: %v", err)
    }

    config, err := LoadConfig()
    if err != nil {
        t.Fatalf("LoadConfig() failed: %v", err)
    }

    if config.Environment.MachtianiURL != "http://localhost:5071" {
        t.Errorf("Expected MACHTIANI_URL to be 'http://localhost:5071', got: %s", config.Environment.MachtianiURL)
    }
}

func TestLoadConfig_ValidConfig(t *testing.T) {
    validConfig := `
environment:
  MODEL_API_KEY: "sk-proj-6a0d4..."
  MACHTIANI_URL: "http://localhost:5071"
  MACHTIANI_REPO_MANAGER_URL: "http://localhost:5070"
  CODE_HOST_URL: "http://localhost:8080"
  CODE_HOST_API_KEY: "ghp_3eZ4c..."
  API_GATEWAY_HOST_KEY: ""
  API_GATEWAY_HOST_VALUE: ""
  CONTENT_TYPE_KEY: "Content-Type"
  CONTENT_TYPE_VALUE: "application/json"
`
    tempFile, err := createTempConfigFile(validConfig)
    if err != nil {
        t.Fatalf("Failed to create temp config file: %v", err)
    }
    defer os.Remove(tempFile)

    // Backup original config if it exists
    originalConfigPath := "machtiani-config.yml"
    if _, err := os.Stat(originalConfigPath); !os.IsNotExist(err) {
        os.Rename(originalConfigPath, originalConfigPath+".bak") // Backup original
        defer os.Rename(originalConfigPath+".bak", originalConfigPath) // Restore original
    }

    // Copy the contents of tempFile to the original config path
    content, err := ioutil.ReadFile(tempFile)
    if err != nil {
        t.Fatalf("Failed to read temp config file: %v", err)
    }

    if err := ioutil.WriteFile(originalConfigPath, content, 0644); err != nil {
        t.Fatalf("Failed to set temp config file: %v", err)
    }

    config, err := LoadConfig()
    if err != nil {
        t.Fatalf("LoadConfig() failed: %v", err)
    }

    if config.Environment.MachtianiURL != "http://localhost:5071" {
        t.Errorf("Expected MACHTIANI_URL to be 'http://localhost:5071', got: %s", config.Environment.MachtianiURL)
    }
}


func TestLoadConfig_InvalidConfigMachtianiURL(t *testing.T) {
    invalidConfig := `
environment:
  MODEL_API_KEY: ""
  MACHTIANI_URL: ""
  MACHTIANI_REPO_MANAGER_URL: "http://localhost:5070"
  CODE_HOST_URL: "http://localhost:8080"
  CODE_HOST_API_KEY: "ghp_3eZ4c..."
  API_GATEWAY_HOST_KEY: ""
  API_GATEWAY_HOST_VALUE: ""
  CONTENT_TYPE_KEY: "Content-Type"
  CONTENT_TYPE_VALUE: "application/json"
`
    tempFile, err := createTempConfigFile(invalidConfig)
    if err != nil {
        t.Fatalf("Failed to create temp config file: %v", err)
    }
    defer os.Remove(tempFile)

    // Backup original config if it exists
    originalConfigPath := "machtiani-config.yml"
    if _, err := os.Stat(originalConfigPath); !os.IsNotExist(err) {
        os.Rename(originalConfigPath, originalConfigPath+".bak") // Backup original
        defer os.Rename(originalConfigPath+".bak", originalConfigPath) // Restore original
    }

    // Copy the contents of tempFile to the original config path
    content, err := ioutil.ReadFile(tempFile)
    if err != nil {
        t.Fatalf("Failed to read temp config file: %v", err)
    }

    if err := ioutil.WriteFile(originalConfigPath, content, 0644); err != nil {
        t.Fatalf("Failed to set temp config file: %v", err)
    }

    _, err = LoadConfig()
    if err == nil {
        t.Fatalf("Expected LoadConfig() to fail for invalid config, but it succeeded")
    }

    expectedErrorMessages := []string{
        "MACHTIANI_URL must be set",
    }

    for _, msg := range expectedErrorMessages {
        if !strings.Contains(err.Error(), msg) {
            t.Errorf("Expected error message to contain '%s', got: %s", msg, err.Error())
        }
    }
}

func TestLoadConfig_InvalidConfigCodeHostURL(t *testing.T) {
    invalidConfig := `
environment:
  MODEL_API_KEY: "sk-proj-6a0d4..."
  MACHTIANI_URL: "http://localhost:5071"
  MACHTIANI_REPO_MANAGER_URL: "http://localhost:5070"
  CODE_HOST_URL: ""
  CODE_HOST_API_KEY: "ghp_3eZ4c..."
  API_GATEWAY_HOST_KEY: ""
  API_GATEWAY_HOST_VALUE: ""
  CONTENT_TYPE_KEY: "Content-Type"
  CONTENT_TYPE_VALUE: "application/json"
`
    tempFile, err := createTempConfigFile(invalidConfig)
    if err != nil {
        t.Fatalf("Failed to create temp config file: %v", err)
    }
    defer os.Remove(tempFile)

    // Backup original config if it exists
    originalConfigPath := "machtiani-config.yml"
    if _, err := os.Stat(originalConfigPath); !os.IsNotExist(err) {
        os.Rename(originalConfigPath, originalConfigPath+".bak") // Backup original
        defer os.Rename(originalConfigPath+".bak", originalConfigPath) // Restore original
    }

    // Copy the contents of tempFile to the original config path
    content, err := ioutil.ReadFile(tempFile)
    if err != nil {
        t.Fatalf("Failed to read temp config file: %v", err)
    }

    if err := ioutil.WriteFile(originalConfigPath, content, 0644); err != nil {
        t.Fatalf("Failed to set temp config file: %v", err)
    }

    _, err = LoadConfig()
    if err == nil {
        t.Fatalf("Expected LoadConfig() to fail for invalid config, but it succeeded")
    }

    expectedErrorMessages := []string{
        "CODE_HOST_URL must be set",
    }

    for _, msg := range expectedErrorMessages {
        if !strings.Contains(err.Error(), msg) {
            t.Errorf("Expected error message to contain '%s', got: %s", msg, err.Error())
        }
    }
}

