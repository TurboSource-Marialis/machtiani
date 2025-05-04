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
	ldflags := fmt.Sprintf("-X 'github.com/turboSource-marialis/machtiani/mct/internal/api.HeadOID=%s' -X 'github.com/turboSource-marialis/machtiani/mct/internal/api.BuildDate=%s' -X 'github.com/turboSource-marialis/machtiani/mct/internal/api.MachtianiGitRemoteURL=%s' -X 'github.com/turboSource-marialis/machtiani/mct/internal/cli.SystemMessageFrequencyHours=24'",
		headOID,
		buildDate,
		"https://github.com/turboSource-marialis/machtiani/mct")

	// If the release flag is provided, process it to include the new ldflags
	if *releaseFlag != "" {
		urls := strings.Fields(*releaseFlag)
		if len(urls) != 2 {
			fmt.Fprintf(os.Stderr, "Error: --release flag should have exactly two arguments: '<MachtianiURL> <RepoManagerURL>'\n")
			os.Exit(1)
		}
		machtianiURL := urls[0]
		repoManagerURL := urls[1]

		// Append the ldflags for MachtianiURL and RepoManagerURL for both api and cli packages
		ldflags += fmt.Sprintf(" -X 'github.com/turboSource-marialis/machtiani/mct/internal/api.MachtianiURL=%s' -X 'github.com/turboSource-marialis/machtiani/mct/internal/api.RepoManagerURL=%s'", machtianiURL, repoManagerURL)
		ldflags += fmt.Sprintf(" -X 'github.com/turboSource-marialis/machtiani/mct/internal/cli.MachtianiURL=%s' -X 'github.com/turboSource-marialis/machtiani/mct/internal/cli.RepoManagerURL=%s'", machtianiURL, repoManagerURL)
	}

	// Print the final ldflags
	fmt.Println(ldflags)
}
