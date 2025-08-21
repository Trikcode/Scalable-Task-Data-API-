package cmd

import (
    "concurrent-data-pipeline/internal/kafka"
    "concurrent-data-pipeline/internal/logger"
    "concurrent-data-pipeline/internal/pipeline"
    "concurrent-data-pipeline/internal/s3"
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/spf13/cobra"
)

var processCmd = &cobra.Command{
    Use:   "process",
    Short: "Start the data pipeline processing",
    Long: `Start processing data from multiple sources concurrently.
This command will consume messages from Kafka, process them using
worker goroutines, and store results in AWS S3.`,
    Run: runProcess,
}

func init() {
    rootCmd.AddCommand(processCmd)
    
    // Command-specific flags
    processCmd.Flags().String("source", "kafka", "data source (kafka, file, api)")
    processCmd.Flags().String("topic", "data-pipeline", "Kafka topic to consume from")
    processCmd.Flags().String("s3-bucket", "", "S3 bucket for output storage")
    processCmd.Flags().Duration("timeout", 30*time.Second, "processing timeout")
}

func runProcess(cmd *cobra.Command, args []string) {
    log := logger.GetLogger()
    log.Info("Starting data pipeline processing")

    // Create context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        sig := <-sigChan
        log.WithField("signal", sig).Info("Received shutdown signal")
        cancel()
    }()

    // Get command flags
    source, _ := cmd.Flags().GetString("source")
    topic, _ := cmd.Flags().GetString("topic")
    s3Bucket, _ := cmd.Flags().GetString("s3-bucket")
    timeout, _ := cmd.Flags().GetDuration("timeout")

    if s3Bucket == "" {
        s3Bucket = cfg.S3.Bucket
    }

    log.WithFields(map[string]interface{}{
        "source":     source,
        "topic":      topic,
        "s3_bucket":  s3Bucket,
        "timeout":    timeout,
        "workers":    cfg.Workers,
    }).Info("Pipeline configuration")

    // Initialize S3 client
    s3Client, err := s3.NewClient(cfg.S3)
    if err != nil {
        log.WithError(err).Fatal("Failed to initialize S3 client")
        return
    }

    // Initialize Kafka consumer
    kafkaConsumer, err := kafka.NewConsumer(cfg.Kafka)
    if err != nil {
        log.WithError(err).Fatal("Failed to initialize Kafka consumer")
        return
    }
    defer kafkaConsumer.Close()

    // Initialize Kafka producer for output/monitoring
    kafkaProducer, err := kafka.NewProducer(cfg.Kafka)
    if err != nil {
        log.WithError(err).Fatal("Failed to initialize Kafka producer")
        return
    }
    defer kafkaProducer.Close()

    // Create pipeline manager
    manager := pipeline.NewManager(&pipeline.Config{
        Workers:    cfg.Workers,
        BatchSize:  cfg.BatchSize,
        Timeout:    timeout,
        S3Client:   s3Client,
        S3Bucket:   s3Bucket,
        Producer:   kafkaProducer,
    })

    // Start processing based on source type
    switch source {
    case "kafka":
        err = processFromKafka(ctx, manager, kafkaConsumer, topic)
    case "file":
        err = processFromFile(ctx, manager, args)
    case "api":
        err = processFromAPI(ctx, manager)
    default:
        err = fmt.Errorf("unsupported source type: %s", source)
    }

    if err != nil {
        log.WithError(err).Error("Pipeline processing failed")
        os.Exit(1)
    }

    log.Info("Pipeline processing completed successfully")
}

func processFromKafka(ctx context.Context, manager *pipeline.Manager, consumer *kafka.Consumer, topic string) error {
    log := logger.GetLogger()
    log.WithField("topic", topic).Info("Starting Kafka consumption")

    // Subscribe to topic
    if err := consumer.Subscribe(topic); err != nil {
        return fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
    }

    // Start the pipeline manager
    if err := manager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start pipeline manager: %w", err)
    }

    // Consume messages
    for {
        select {
        case <-ctx.Done():
            log.Info("Shutting down Kafka consumer")
            return manager.Stop()
        default:
            messages, err := consumer.ConsumeMessages(ctx, 100) // Batch of 100 messages
            if err != nil {
                log.WithError(err).Error("Failed to consume messages")
                continue
            }

            if len(messages) > 0 {
                log.WithField("message_count", len(messages)).Debug("Received messages batch")
                
                // Convert kafka messages to pipeline messages
                pipelineMessages := make([]*pipeline.Message, len(messages))
                for i, kafkaMsg := range messages {
                    pipelineMessages[i] = &pipeline.Message{
                        ID:      fmt.Sprintf("kafka-%d-%d", kafkaMsg.Offset, time.Now().UnixNano()),
                        Source:  "kafka",
                        Data:    kafkaMsg.Value,
                        Headers: kafkaMsg.Headers,
                    }
                }
                
                // Submit batch for processing
                if err := manager.SubmitBatch(pipelineMessages); err != nil {
                    log.WithError(err).Error("Failed to submit batch for processing")
                }
            }
        }
    }
}

func processFromFile(ctx context.Context, manager *pipeline.Manager, files []string) error {
    log := logger.GetLogger()
    log.WithField("files", files).Info("Starting file processing")

    if len(files) == 0 {
        return fmt.Errorf("no files specified for processing")
    }

    // Start the pipeline manager
    if err := manager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start pipeline manager: %w", err)
    }

    // Process each file
    for _, file := range files {
        select {
        case <-ctx.Done():
            log.Info("Processing cancelled")
            return manager.Stop()
        default:
            log.WithField("file", file).Info("Processing file")
            
            // Create a mock message for the file
            message := &pipeline.Message{
                ID:      fmt.Sprintf("file-%d", time.Now().UnixNano()),
                Source:  "file",
                Data:    []byte(file), // In real implementation, read file content
                Headers: map[string]string{"file_path": file},
            }

            if err := manager.SubmitMessage(message); err != nil {
                log.WithError(err).WithField("file", file).Error("Failed to submit file for processing")
            }
        }
    }

    // Wait for completion
    time.Sleep(5 * time.Second) // Give time for processing to complete
    return manager.Stop()
}

func processFromAPI(ctx context.Context, manager *pipeline.Manager) error {
    log := logger.GetLogger()
    log.Info("Starting API data processing")

    // Start the pipeline manager
    if err := manager.Start(ctx); err != nil {
        return fmt.Errorf("failed to start pipeline manager: %w", err)
    }

    // Simulate API data fetching (in real implementation, this would call actual APIs)
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            log.Info("API processing cancelled")
            return manager.Stop()
        case <-ticker.C:
            log.Debug("Fetching data from API")
            
            // Create a mock message for API data
            message := &pipeline.Message{
                ID:      fmt.Sprintf("api-%d", time.Now().UnixNano()),
                Source:  "api",
                Data:    []byte(fmt.Sprintf(`{"timestamp": "%s", "data": "sample_api_data"}`, time.Now().Format(time.RFC3339))),
                Headers: map[string]string{"api_endpoint": "/data"},
            }

            if err := manager.SubmitMessage(message); err != nil {
                log.WithError(err).Error("Failed to submit API data for processing")
            }
        }
    }
}
