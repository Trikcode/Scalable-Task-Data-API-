package cmd

import (
    "concurrent-data-pipeline/internal/config"
    "concurrent-data-pipeline/internal/logger"
    "fmt"
    "os"

    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    cfg     *config.Config
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
    Use:   "pipeline",
    Short: "Concurrent Data Pipeline Tool",
    Long: `A Go-based CLI tool for concurrent data pipeline processing 
with Kafka and AWS S3 integration. Process multiple data sources 
concurrently using goroutines and channels.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
    return rootCmd.Execute()
}

func init() {
    cobra.OnInitialize(initConfig)

    // Global flags
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config.yaml)")
    rootCmd.PersistentFlags().String("log-level", "info", "log level (debug, info, warn, error)")
    rootCmd.PersistentFlags().Int("workers", 5, "number of worker goroutines")
    rootCmd.PersistentFlags().Int("batch-size", 100, "batch size for processing")
    
    // Bind flags to viper
    viper.BindPFlag("log_level", rootCmd.PersistentFlags().Lookup("log-level"))
    viper.BindPFlag("workers", rootCmd.PersistentFlags().Lookup("workers"))
    viper.BindPFlag("batch_size", rootCmd.PersistentFlags().Lookup("batch-size"))
}

// initConfig reads in config file and ENV variables.
func initConfig() {
    if cfgFile != "" {
        // Use config file from the flag.
        viper.SetConfigFile(cfgFile)
    } else {
        // Search for config in current directory
        viper.AddConfigPath(".")
        viper.SetConfigType("yaml")
        viper.SetConfigName("config")
    }

    // Read environment variables
    viper.AutomaticEnv()

    // If a config file is found, read it in.
    if err := viper.ReadInConfig(); err == nil {
        logger.GetLogger().WithField("config_file", viper.ConfigFileUsed()).Info("Using config file")
    } else {
        logger.GetLogger().WithError(err).Warn("Config file not found, using defaults and environment variables")
    }

    // Load configuration
    var err error
    cfg, err = config.Load()
    if err != nil {
        logger.GetLogger().WithError(err).Fatal("Failed to load configuration")
        os.Exit(1)
    }

    // Set log level
    logLevel := viper.GetString("log_level")
    if err := logger.SetLevel(logLevel); err != nil {
        logger.GetLogger().WithError(err).Warn("Invalid log level, using default")
    }

    fmt.Printf("Concurrent Data Pipeline Tool initialized\n")
    fmt.Printf("Workers: %d\n", cfg.Workers)
    fmt.Printf("Batch Size: %d\n", cfg.BatchSize)
    fmt.Printf("Log Level: %s\n", logLevel)
}
