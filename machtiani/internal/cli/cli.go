package cli

import (
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "strings"
    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/git"
)

const (
    defaultModel        = "gpt-4o-mini"
    defaultMatchStrength = "mid"
    defaultMode         = "commit"
)

func Execute() {
    fs := flag.NewFlagSet("machtiani", flag.ContinueOnError)
    markdownFlag := fs.String("markdown", "", "Path to the markdown file")
    projectFlag := fs.String("project", "", "Name of the project (if not set, it will be fetched from git)")
    modelFlag := fs.String("model", defaultModel, "Model to use (options: gpt-4o, gpt-4o-mini)")
    matchStrengthFlag := fs.String("match-strength", defaultMatchStrength, "Match strength (options: high, mid, low)")
    modeFlag := fs.String("mode", defaultMode, "Search mode: content, commit, or super")
    verboseFlag := fs.Bool("verbose", false, "Enable verbose output.")

    args := os.Args[1:]
    var promptParts []string
    var flagArgs []string

    for i := 0; i < len(args); i++ {
        if strings.HasPrefix(args[i], "-") {
            flagArgs = append(flagArgs, args[i])
            if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
                flagArgs = append(flagArgs, args[i+1])
                i++
            }
        } else {
            promptParts = append(promptParts, args[i])
        }
    }

    if err := fs.Parse(flagArgs); err != nil {
        log.Fatalf("Error parsing flags: %v", err)
    }

    prompt := strings.Join(promptParts, " ")
    project, err := getProjectOrDefault(projectFlag)
    if err != nil {
        log.Fatalf("Error getting project name: %v", err)
    }

    validateFlags(modelFlag, matchStrengthFlag, modeFlag)

    if *markdownFlag != "" {
        content, err := ioutil.ReadFile(*markdownFlag)
        if err != nil {
            log.Fatalf("Error reading markdown file: %v", err)
        }
        prompt = string(content)
    } else if prompt == "" {
        log.Fatal("Error: No prompt provided. Please provide either a prompt or a markdown file.")
    }

    if *verboseFlag {
        printVerboseInfo(*markdownFlag, *projectFlag, *modelFlag, *matchStrengthFlag, *modeFlag, prompt)
    }

    openAIAPIKey := os.Getenv("OPENAI_API_KEY")
    if openAIAPIKey == "" {
        log.Fatal("Error: OPENAI_API_KEY environment variable is not set.")
    }

    apiResponse, err := api.CallOpenAIAPI(openAIAPIKey, prompt, project, *modeFlag, *modelFlag, *matchStrengthFlag)
    if err != nil {
        log.Fatalf("Error making API call: %v", err)
    }

    handleAPIResponse(apiResponse, *markdownFlag)
}

func getProjectOrDefault(projectFlag *string) (string, error) {
    if *projectFlag == "" {
        return git.GetProjectName()
    }
    return *projectFlag, nil
}

func validateFlags(modelFlag, matchStrengthFlag, modeFlag *string) {
    // Validate model argument
    model := *modelFlag
    if model != "gpt-4o" && model != "gpt-4o-mini" {
        log.Fatalf("Error: Invalid model selected. Choose either 'gpt-4o' or 'gpt-4o-mini'.")
    }

    // Validate match strength argument
    matchStrength := *matchStrengthFlag
    if matchStrength != "high" && matchStrength != "mid" && matchStrength != "low" {
        log.Fatalf("Error: Invalid match strength selected. Choose either 'high', 'mid', or 'low'.")
    }

    // Validate mode argument
    mode := *modeFlag
    if mode != "content" && mode != "commit" && mode != "super" {
        log.Fatalf("Error: Invalid mode selected. Choose either 'content', 'commit', or 'super'.")
    }
}

func printVerboseInfo(markdown, project, model, matchStrength, mode, prompt string) {
    fmt.Println("Arguments passed:")
    fmt.Printf("Markdown file: %s\n", markdown)
    fmt.Printf("Project name: %s\n", project)
    fmt.Printf("Model: %s\n", model)
    fmt.Printf("Match strength: %s\n", matchStrength)
    fmt.Printf("Mode: %s\n", mode)
    fmt.Printf("Prompt: %s\n", prompt)
}

func handleAPIResponse(apiResponse map[string]interface{}, markdownFlag string) {
    openAIResponse, ok := apiResponse["openai_response"].(string)
    if !ok {
        log.Fatalf("Error: openai_response key missing")
    }

    // Additional logic to handle the response and save to file, etc.
}
