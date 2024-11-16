package cli

import (
    "fmt"
    "log"
    "time"

    "github.com/7db9a/machtiani/internal/api"
    "github.com/7db9a/machtiani/internal/utils"
)

func handleStatus(config *utils.Config, remoteURL string, apiKey *string) {
    // Call CheckStatus
    statusResponse, err := api.CheckStatus(remoteURL, apiKey)
    if err != nil {
        log.Fatalf("Error checking status: %v", err)
    }

    // Output the result
    if statusResponse.LockFilePresent {
        fmt.Println("Project is getting processed and not ready for chat.")
        // Convert the float64 seconds to a duration (in nanoseconds)
        duration := time.Duration(statusResponse.LockTimeDuration * float64(time.Second))
        // Format the duration to show hours, minutes, seconds
        fmt.Printf("Lock duration: %02d:%02d:%02d\n", int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
    } else {
        fmt.Println("Project is ready for chat!")
    }
}
