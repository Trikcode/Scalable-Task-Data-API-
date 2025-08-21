package pipeline

import (
    "concurrent-data-pipeline/internal/logger"
    "context"
    "encoding/json"
    "fmt"
    "strings"
    "time"
)

// Worker represents a pipeline worker
type Worker struct {
    id       int
    manager  *Manager
    jobChan  <-chan *Job
    quitChan chan bool
}

// Job represents a unit of work for the pipeline
type Job struct {
    ID       string
    Message  *Message
    Attempt  int
    Context  context.Context
}

// Message represents a data message to be processed
type Message struct {
    ID      string
    Source  string
    Data    []byte
    Headers map[string]string
}

// ProcessingResult represents the result of processing a message
type ProcessingResult struct {
    JobID        string
    MessageID    string
    Success      bool
    Error        string
    ProcessedAt  time.Time
    ProcessingDuration time.Duration
    OutputData   []byte
    Metadata     map[string]string
}

// NewWorker creates a new worker
func NewWorker(id int, manager *Manager, jobChan <-chan *Job) *Worker {
    return &Worker{
        id:       id,
        manager:  manager,
        jobChan:  jobChan,
        quitChan: make(chan bool),
    }
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
    log := logger.GetLogger()
    log.WithField("worker_id", w.id).Info("Starting worker")
    
    go func() {
        for {
            select {
            case job := <-w.jobChan:
                w.processJob(job)
            case <-w.quitChan:
                log.WithField("worker_id", w.id).Info("Worker stopped")
                return
            case <-ctx.Done():
                log.WithField("worker_id", w.id).Info("Worker stopped due to context cancellation")
                return
            }
        }
    }()
}

// Stop stops the worker
func (w *Worker) Stop() {
    log := logger.GetLogger()
    log.WithField("worker_id", w.id).Info("Stopping worker")
    w.quitChan <- true
}

// processJob processes a single job
func (w *Worker) processJob(job *Job) {
    log := logger.GetLogger()
    startTime := time.Now()
    
    log.WithFields(map[string]interface{}{
        "worker_id":  w.id,
        "job_id":     job.ID,
        "message_id": job.Message.ID,
        "source":     job.Message.Source,
        "attempt":    job.Attempt,
    }).Debug("Processing job")
    
    result := &ProcessingResult{
        JobID:       job.ID,
        MessageID:   job.Message.ID,
        ProcessedAt: startTime,
        Metadata:    make(map[string]string),
    }
    
    // Process the message
    err := w.processMessage(job.Message, result)
    
    result.ProcessingDuration = time.Since(startTime)
    result.Success = err == nil
    
    if err != nil {
        result.Error = err.Error()
        log.WithError(err).WithFields(map[string]interface{}{
            "worker_id":  w.id,
            "job_id":     job.ID,
            "message_id": job.Message.ID,
            "attempt":    job.Attempt,
        }).Error("Job processing failed")
    } else {
        log.WithFields(map[string]interface{}{
            "worker_id":  w.id,
            "job_id":     job.ID,
            "message_id": job.Message.ID,
            "duration":   result.ProcessingDuration,
        }).Debug("Job processed successfully")
    }
    
    // Handle the result
    w.handleResult(result, job)
}

// processMessage processes a single message and generates output
func (w *Worker) processMessage(msg *Message, result *ProcessingResult) error {
    _ = logger.GetLogger()
    
    // Add metadata
    result.Metadata["source"] = msg.Source
    result.Metadata["worker_id"] = fmt.Sprintf("%d", w.id)
    result.Metadata["original_size"] = fmt.Sprintf("%d", len(msg.Data))
    
    // Copy headers to metadata
    for key, value := range msg.Headers {
        result.Metadata["header_"+key] = value
    }
    
    // Process based on source type
    switch msg.Source {
    case "kafka":
        return w.processKafkaMessage(msg, result)
    case "file":
        return w.processFileMessage(msg, result)
    case "api":
        return w.processAPIMessage(msg, result)
    default:
        return w.processGenericMessage(msg, result)
    }
}

// processKafkaMessage processes a Kafka message
func (w *Worker) processKafkaMessage(msg *Message, result *ProcessingResult) error {
    log := logger.GetLogger()
    
    // Validate JSON if it appears to be JSON
    if strings.HasPrefix(strings.TrimSpace(string(msg.Data)), "{") {
        var jsonData map[string]interface{}
        if err := json.Unmarshal(msg.Data, &jsonData); err != nil {
            return fmt.Errorf("invalid JSON in Kafka message: %w", err)
        }
        result.Metadata["format"] = "json"
        result.Metadata["json_keys"] = fmt.Sprintf("%d", len(jsonData))
    } else {
        result.Metadata["format"] = "text"
    }
    
    // Transform the data (example: add processing timestamp)
    processedData := map[string]interface{}{
        "original_data": json.RawMessage(msg.Data),
        "processed_at":  time.Now().Format(time.RFC3339),
        "processed_by":  fmt.Sprintf("worker-%d", w.id),
        "message_id":    msg.ID,
        "source":        "kafka",
    }
    
    outputData, err := json.Marshal(processedData)
    if err != nil {
        return fmt.Errorf("failed to marshal processed data: %w", err)
    }
    
    result.OutputData = outputData
    result.Metadata["output_size"] = fmt.Sprintf("%d", len(outputData))
    
    log.WithField("message_id", msg.ID).Debug("Processed Kafka message")
    return nil
}

// processFileMessage processes a file-based message
func (w *Worker) processFileMessage(msg *Message, result *ProcessingResult) error {
    log := logger.GetLogger()
    
    filePath := string(msg.Data)
    result.Metadata["file_path"] = filePath
    result.Metadata["format"] = "file"
    
    // Create processed file metadata
    processedData := map[string]interface{}{
        "file_path":    filePath,
        "processed_at": time.Now().Format(time.RFC3339),
        "processed_by": fmt.Sprintf("worker-%d", w.id),
        "message_id":   msg.ID,
        "source":       "file",
        "status":       "processed",
    }
    
    outputData, err := json.Marshal(processedData)
    if err != nil {
        return fmt.Errorf("failed to marshal file processing result: %w", err)
    }
    
    result.OutputData = outputData
    result.Metadata["output_size"] = fmt.Sprintf("%d", len(outputData))
    
    log.WithFields(map[string]interface{}{
        "message_id": msg.ID,
        "file_path":  filePath,
    }).Debug("Processed file message")
    
    return nil
}

// processAPIMessage processes an API-based message
func (w *Worker) processAPIMessage(msg *Message, result *ProcessingResult) error {
    log := logger.GetLogger()
    
    result.Metadata["format"] = "api"
    
    // Parse API data if it's JSON
    var apiData map[string]interface{}
    if err := json.Unmarshal(msg.Data, &apiData); err != nil {
        return fmt.Errorf("invalid JSON in API message: %w", err)
    }
    
    // Enrich the API data
    processedData := map[string]interface{}{
        "api_data":     apiData,
        "processed_at": time.Now().Format(time.RFC3339),
        "processed_by": fmt.Sprintf("worker-%d", w.id),
        "message_id":   msg.ID,
        "source":       "api",
        "enrichment": map[string]interface{}{
            "processing_timestamp": time.Now().Unix(),
            "worker_id":            w.id,
            "data_size":            len(msg.Data),
        },
    }
    
    outputData, err := json.Marshal(processedData)
    if err != nil {
        return fmt.Errorf("failed to marshal API processing result: %w", err)
    }
    
    result.OutputData = outputData
    result.Metadata["output_size"] = fmt.Sprintf("%d", len(outputData))
    
    log.WithField("message_id", msg.ID).Debug("Processed API message")
    return nil
}

// processGenericMessage processes a generic message
func (w *Worker) processGenericMessage(msg *Message, result *ProcessingResult) error {
    log := logger.GetLogger()
    
    result.Metadata["format"] = "generic"
    
    // Create a generic processed message
    processedData := map[string]interface{}{
        "data":         string(msg.Data),
        "processed_at": time.Now().Format(time.RFC3339),
        "processed_by": fmt.Sprintf("worker-%d", w.id),
        "message_id":   msg.ID,
        "source":       msg.Source,
        "data_length":  len(msg.Data),
    }
    
    outputData, err := json.Marshal(processedData)
    if err != nil {
        return fmt.Errorf("failed to marshal generic processing result: %w", err)
    }
    
    result.OutputData = outputData
    result.Metadata["output_size"] = fmt.Sprintf("%d", len(outputData))
    
    log.WithField("message_id", msg.ID).Debug("Processed generic message")
    return nil
}

// handleResult handles the processing result
func (w *Worker) handleResult(result *ProcessingResult, job *Job) {
    log := logger.GetLogger()
    
    if result.Success {
        // Store successful result
        w.manager.handleSuccessfulResult(result)
    } else {
        // Handle failure with retry logic
        if job.Attempt < w.manager.config.MaxRetries {
            log.WithFields(map[string]interface{}{
                "job_id":     job.ID,
                "message_id": job.Message.ID,
                "attempt":    job.Attempt,
                "max_retries": w.manager.config.MaxRetries,
            }).Info("Retrying failed job")
            
            // Retry the job
            retryJob := &Job{
                ID:      job.ID,
                Message: job.Message,
                Attempt: job.Attempt + 1,
                Context: job.Context,
            }
            
            // Add delay before retry
            go func() {
                time.Sleep(time.Duration(job.Attempt) * time.Second)
                w.manager.SubmitJob(retryJob)
            }()
        } else {
            log.WithFields(map[string]interface{}{
                "job_id":      job.ID,
                "message_id":  job.Message.ID,
                "attempt":     job.Attempt,
                "max_retries": w.manager.config.MaxRetries,
            }).Error("Job failed permanently after max retries")
            
            // Store failed result
            w.manager.handleFailedResult(result)
        }
    }
    
    // Update metrics
    w.manager.updateMetrics(result)
}
