package main

import (
    "concurrent-data-pipeline/cmd"
    "concurrent-data-pipeline/internal/logger"
    "os"
)

func main() {
    // Initialize logger
    logger.Init()
    
    // Execute CLI
    if err := cmd.Execute(); err != nil {
        logger.GetLogger().WithError(err).Fatal("Failed to execute command")
        os.Exit(1)
    }
}
