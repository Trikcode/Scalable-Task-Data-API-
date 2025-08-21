package pipeline

import (
    "concurrent-data-pipeline/internal/kafka"
    "concurrent-data-pipeline/internal/logger"
    "concurrent-data-pipeline/internal/metrics"
    "concurrent-data-pipeline/internal/s3"
    "context"
    "encoding/json"
    "fmt"
    "sync"
    "time"
)

// Manager manages the pipeline workers and job distribution
type Manager struct {
    config      *Config
    workers     []*Worker
    jobQueue    chan *Job
    resultQueue chan *ProcessingResult
    metrics     *metrics.Collector
    wg          sync.WaitGroup
    ctx         context.Context
    cancel      context.CancelFunc
    started     bool
    mu          sync.RWMutex
}

// Config holds configuration for the pipeline manager
type Config struct {
    Workers     int
    BatchSize   int
    Timeout     time.Duration
    MaxRetries  int
    S3Client    *s3.Client
    S3Bucket    string
    Producer    *kafka.Producer
}

// NewManager creates a new pipeline manager
func NewManager(config *Config) *Manager {
    if config.MaxRetries == 0 {
        config.MaxRetries = 3
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    return &Manager{
        config:      config,
        jobQueue:    make(chan *Job, config.BatchSize*2), // Buffer for jobs
        resultQueue: make(chan *ProcessingResult, config.BatchSize),
        metrics:     metrics.NewCollector(),
        ctx:         ctx,
        cancel:      cancel,
    }
}

// Start starts the pipeline manager and workers
func (m *Manager) Start(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if m.started {
        return fmt.Errorf("manager already started")
    }
    
    log := logger.GetLogger()
    log.WithField("workers", m.config.Workers).Info("Starting pipeline manager")
    
    // Create and start workers
    m.workers = make([]*Worker, m.config.Workers)
    for i := 0; i < m.config.Workers; i++ {
        worker := NewWorker(i+1, m, m.jobQueue)
        m.workers[i] = worker
        worker.Start(ctx)
    }
    
    // Start result handler
    m.wg.Add(1)
    go m.resultHandler(ctx)
    
    // Start metrics collector
    m.wg.Add(1)
    go m.metricsHandler(ctx)
    
    m.started = true
    log.Info("Pipeline manager started successfully")
    return nil
}

// Stop stops the pipeline manager and all workers
func (m *Manager) Stop() error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if !m.started {
        return nil
    }
    
    log := logger.GetLogger()
    log.Info("Stopping pipeline manager")
    
    // Cancel context to signal shutdown
    m.cancel()
    
    // Stop all workers
    for _, worker := range m.workers {
        worker.Stop()
    }
    
    // Close channels
    close(m.jobQueue)
    close(m.resultQueue)
    
    // Wait for goroutines to finish
    m.wg.Wait()
    
    m.started = false
    log.Info("Pipeline manager stopped")
    return nil
}

// SubmitMessage submits a single message for processing
func (m *Manager) SubmitMessage(message *Message) error {
    job := &Job{
        ID:      fmt.Sprintf("job-%d", time.Now().UnixNano()),
        Message: message,
        Attempt: 1,
        Context: m.ctx,
    }
    
    return m.SubmitJob(job)
}

// SubmitBatch submits a batch of messages for processing
func (m *Manager) SubmitBatch(messages []*Message) error {
    log := logger.GetLogger()
    
    for _, message := range messages {
        if err := m.SubmitMessage(message); err != nil {
            log.WithError(err).WithField("message_id", message.ID).Error("Failed to submit message")
        }
    }
    
    log.WithField("batch_size", len(messages)).Debug("Submitted batch for processing")
    return nil
}

// SubmitJob submits a job to the job queue
func (m *Manager) SubmitJob(job *Job) error {
    select {
    case m.jobQueue <- job:
        m.metrics.IncrementJobsSubmitted()
        return nil
    case <-time.After(5 * time.Second):
        return fmt.Errorf("timeout submitting job to queue")
    case <-m.ctx.Done():
        return fmt.Errorf("manager is shutting down")
    }
}

// resultHandler handles processing results
func (m *Manager) resultHandler(ctx context.Context) {
    defer m.wg.Done()
    log := logger.GetLogger()
    
    log.Info("Started result handler")
    
    for {
        select {
        case result := <-m.resultQueue:
            if result != nil {
                m.handleResult(result)
            }
        case <-ctx.Done():
            log.Info("Result handler stopped")
            return
        }
    }
}

// metricsHandler periodically logs metrics
func (m *Manager) metricsHandler(ctx context.Context) {
    defer m.wg.Done()
    log := logger.GetLogger()
    
    ticker := time.NewTicker(30 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            m.logMetrics()
        case <-ctx.Done():
            log.Info("Metrics handler stopped")
            return
        }
    }
}

// handleResult processes a processing result
func (m *Manager) handleResult(result *ProcessingResult) {
    if result.Success {
        m.handleSuccessfulResult(result)
    } else {
        m.handleFailedResult(result)
    }
}

// handleSuccessfulResult handles a successful processing result
func (m *Manager) handleSuccessfulResult(result *ProcessingResult) {
    log := logger.GetLogger()
    
    // Store result in S3
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    key := fmt.Sprintf("processed/%s/%s.json", 
        time.Now().Format("2006/01/02/15"), 
        result.MessageID)
    
    err := m.config.S3Client.UploadJSON(ctx, m.config.S3Bucket, key, result.OutputData)
    if err != nil {
        log.WithError(err).WithFields(map[string]interface{}{
            "message_id": result.MessageID,
            "s3_key":     key,
        }).Error("Failed to store result in S3")
    } else {
        log.WithFields(map[string]interface{}{
            "message_id": result.MessageID,
            "s3_key":     key,
            "duration":   result.ProcessingDuration,
        }).Debug("Stored processing result in S3")
    }
    
    // Send success notification to Kafka
    if m.config.Producer != nil {
        kafkaMsg := &kafka.ProduceMessage{
            Topic:   "processing-results",
            Key:     result.MessageID,
            Value:   result.OutputData,
            Headers: map[string]string{
                "status":             "success",
                "processing_duration": result.ProcessingDuration.String(),
                "processed_at":        result.ProcessedAt.Format(time.RFC3339),
            },
        }
        
        if err := m.config.Producer.SendMessage(kafkaMsg); err != nil {
            log.WithError(err).WithField("message_id", result.MessageID).Error("Failed to send success notification")
        }
    }
    
    m.metrics.IncrementJobsSucceeded()
}

// handleFailedResult handles a failed processing result
func (m *Manager) handleFailedResult(result *ProcessingResult) {
    log := logger.GetLogger()
    
    log.WithFields(map[string]interface{}{
        "message_id": result.MessageID,
        "error":      result.Error,
        "duration":   result.ProcessingDuration,
    }).Error("Processing failed permanently")
    
    // Store error in S3
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    errorData := map[string]interface{}{
        "message_id":  result.MessageID,
        "error":       result.Error,
        "failed_at":   result.ProcessedAt.Format(time.RFC3339),
        "duration":    result.ProcessingDuration.String(),
        "metadata":    result.Metadata,
    }
    
    key := fmt.Sprintf("failed/%s/%s.json", 
        time.Now().Format("2006/01/02/15"), 
        result.MessageID)
    
    if errorJSON, err := json.Marshal(errorData); err == nil {
        if err := m.config.S3Client.UploadJSON(ctx, m.config.S3Bucket, key, errorJSON); err != nil {
            log.WithError(err).WithField("message_id", result.MessageID).Error("Failed to store error in S3")
        }
    }
    
    // Send failure notification to Kafka
    if m.config.Producer != nil {
        kafkaMsg := &kafka.ProduceMessage{
            Topic: "processing-results",
            Key:   result.MessageID,
            Value: []byte(result.Error),
            Headers: map[string]string{
                "status":             "failed",
                "error":              result.Error,
                "processing_duration": result.ProcessingDuration.String(),
                "failed_at":          result.ProcessedAt.Format(time.RFC3339),
            },
        }
        
        if err := m.config.Producer.SendMessage(kafkaMsg); err != nil {
            log.WithError(err).WithField("message_id", result.MessageID).Error("Failed to send failure notification")
        }
    }
    
    m.metrics.IncrementJobsFailed()
}

// updateMetrics updates pipeline metrics
func (m *Manager) updateMetrics(result *ProcessingResult) {
    m.metrics.RecordProcessingDuration(result.ProcessingDuration)
    
    if result.Success {
        m.metrics.IncrementJobsSucceeded()
    } else {
        m.metrics.IncrementJobsFailed()
    }
}

// logMetrics logs current pipeline metrics
func (m *Manager) logMetrics() {
    log := logger.GetLogger()
    metrics := m.metrics.GetMetrics()
    
    log.WithFields(map[string]interface{}{
        "jobs_submitted":       metrics["jobs_submitted"],
        "jobs_succeeded":       metrics["jobs_succeeded"],
        "jobs_failed":          metrics["jobs_failed"],
        "avg_processing_time":  metrics["avg_processing_duration"],
        "queue_size":           len(m.jobQueue),
        "active_workers":       len(m.workers),
    }).Info("Pipeline metrics")
}

// GetMetrics returns current pipeline metrics
func (m *Manager) GetMetrics() map[string]interface{} {
    metrics := m.metrics.GetMetrics()
    metrics["queue_size"] = len(m.jobQueue)
    metrics["active_workers"] = len(m.workers)
    metrics["started"] = m.started
    return metrics
}
