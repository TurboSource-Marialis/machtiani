package cli

import (
	"fmt"
	"log"
	"time"

	"github.com/turboSource-marialis/machtiani/mct/internal/api"
	"github.com/turboSource-marialis/machtiani/mct/internal/utils"
)

func handleStatus(config *utils.Config, remoteURL string) {
    // Call CheckStatus
    statusResponse, err := api.CheckStatus(remoteURL)
    if err != nil {
        log.Fatalf("Error checking status: %v", err)
    }

    // Output the result
    if statusResponse.LockFilePresent {
        // Project is still processing
        fmt.Println("Project is getting processed and not ready for chat.")
        // Convert the float64 seconds to a duration
        duration := time.Duration(statusResponse.LockTimeDuration * float64(time.Second))
        fmt.Printf("Lock duration: %02d:%02d:%02d\n", int(duration.Hours()), int(duration.Minutes())%60, int(duration.Seconds())%60)
        // Display error logs if present during processing
        if statusResponse.ErrorLogs != "" {
            fmt.Println("\nError logs:")
            fmt.Println(statusResponse.ErrorLogs)
        }
    } else {
        // Processing complete
        if statusResponse.ErrorLogs != "" {
            // Errors occurred during processing
            fmt.Println("Project encountered errors during processing and is not ready for chat.")
            fmt.Println("\nError logs:")
            fmt.Println(statusResponse.ErrorLogs)
        } else {
            // No errors, project ready
            fmt.Println("Project is ready for chat!")
        }
    }
}
