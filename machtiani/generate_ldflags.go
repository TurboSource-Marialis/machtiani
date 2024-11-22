package main

import (
    "flag"
    "fmt"
    "os"
    "os/exec"
    "strings"
    "time"
)

func main() {
    // Define the flags
    releaseFlag := flag.String("release", "", "Specify the MachtianiURL and RepoManagerURL in the format: '<MachtianiURL> <RepoManagerURL>'")
    flag.Parse()

    // Get Git commit hash
    cmd := exec.Command("git", "rev-parse", "HEAD")
    commitBytes, err := cmd.Output()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error getting commit hash: %v\n", err)
        os.Exit(1)
    }
    headOID := strings.TrimSpace(string(commitBytes))

    // Get build date
    buildDate := time.Now().Format(time.RFC3339)

    // Start constructing ldflags
    ldflags := fmt.Sprintf("-X 'github.com/7db9a/machtiani/internal/api.HeadOID=%s' -X 'github.com/7db9a/machtiani/internal/api.BuildDate=%s'", headOID, buildDate)

    // If the release flag is provided, process it to include the new ldflags
    if *releaseFlag != "" {
        urls := strings.Fields(*releaseFlag)
        if len(urls) != 2 {
            fmt.Fprintf(os.Stderr, "Error: --release flag should have exactly two arguments: '<MachtianiURL> <RepoManagerURL>'\n")
            os.Exit(1)
        }
        machtianiURL := urls[0]
        repoManagerURL := urls[1]

        // Append the ldflags for MachtianiURL and RepoManagerURL
        ldflags += fmt.Sprintf(" -X 'github.com/7db9a/machtiani/internal/api.MachtianiURL=%s' -X 'github.com/7db9a/machtiani/internal/api.RepoManagerURL=%s' -X 'github.com/7db9a/machtiani/internal/cli.MachtianiURL=%s'", machtianiURL, repoManagerURL, machtianiURL)
    }

    // Print the final ldflags
    fmt.Println(ldflags)
}
