package metrics

import (
    "fmt"
    "sync"
    "time"
)

// Collector collects and manages pipeline metrics
type Collector struct {
    mu                     sync.RWMutex
    jobsSubmitted          int64
    jobsSucceeded          int64
    jobsFailed             int64
    totalProcessingTime    time.Duration
    processedJobs          int64
    startTime              time.Time
    lastResetTime          time.Time
}

// NewCollector creates a new metrics collector
func NewCollector() *Collector {
    now := time.Now()
    return &Collector{
        startTime:     now,
        lastResetTime: now,
    }
}

// IncrementJobsSubmitted increments the jobs submitted counter
func (c *Collector) IncrementJobsSubmitted() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jobsSubmitted++
}

// IncrementJobsSucceeded increments the jobs succeeded counter
func (c *Collector) IncrementJobsSucceeded() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jobsSucceeded++
}

// IncrementJobsFailed increments the jobs failed counter
func (c *Collector) IncrementJobsFailed() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.jobsFailed++
}

// RecordProcessingDuration records the processing duration for a job
func (c *Collector) RecordProcessingDuration(duration time.Duration) {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.totalProcessingTime += duration
    c.processedJobs++
}

// GetJobsSubmitted returns the number of jobs submitted
func (c *Collector) GetJobsSubmitted() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.jobsSubmitted
}

// GetJobsSucceeded returns the number of jobs succeeded
func (c *Collector) GetJobsSucceeded() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.jobsSucceeded
}

// GetJobsFailed returns the number of jobs failed
func (c *Collector) GetJobsFailed() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.jobsFailed
}

// GetAverageProcessingDuration returns the average processing duration
func (c *Collector) GetAverageProcessingDuration() time.Duration {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    if c.processedJobs == 0 {
        return 0
    }
    return c.totalProcessingTime / time.Duration(c.processedJobs)
}

// GetTotalProcessingTime returns the total processing time
func (c *Collector) GetTotalProcessingTime() time.Duration {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.totalProcessingTime
}

// GetProcessedJobs returns the number of processed jobs
func (c *Collector) GetProcessedJobs() int64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return c.processedJobs
}

// GetUptime returns the uptime since the collector was created
func (c *Collector) GetUptime() time.Duration {
    c.mu.RLock()
    defer c.mu.RUnlock()
    return time.Since(c.startTime)
}

// GetSuccessRate returns the success rate as a percentage
func (c *Collector) GetSuccessRate() float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    total := c.jobsSucceeded + c.jobsFailed
    if total == 0 {
        return 0.0
    }
    return float64(c.jobsSucceeded) / float64(total) * 100.0
}

// GetThroughput returns the jobs per second throughput
func (c *Collector) GetThroughput() float64 {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    uptime := time.Since(c.startTime).Seconds()
    if uptime == 0 {
        return 0.0
    }
    return float64(c.processedJobs) / uptime
}

// GetMetrics returns all metrics as a map
func (c *Collector) GetMetrics() map[string]interface{} {
    c.mu.RLock()
    defer c.mu.RUnlock()
    
    uptime := time.Since(c.startTime)
    totalJobs := c.jobsSucceeded + c.jobsFailed
    
    var successRate float64
    if totalJobs > 0 {
        successRate = float64(c.jobsSucceeded) / float64(totalJobs) * 100.0
    }
    
    var throughput float64
    if uptime.Seconds() > 0 {
        throughput = float64(c.processedJobs) / uptime.Seconds()
    }
    
    var avgDuration time.Duration
    if c.processedJobs > 0 {
        avgDuration = c.totalProcessingTime / time.Duration(c.processedJobs)
    }
    
    return map[string]interface{}{
        "jobs_submitted":          c.jobsSubmitted,
        "jobs_succeeded":          c.jobsSucceeded,
        "jobs_failed":             c.jobsFailed,
        "total_jobs":              totalJobs,
        "processed_jobs":          c.processedJobs,
        "success_rate_percent":    successRate,
        "throughput_jobs_per_sec": throughput,
        "avg_processing_duration": avgDuration.String(),
        "total_processing_time":   c.totalProcessingTime.String(),
        "uptime":                  uptime.String(),
        "start_time":              c.startTime.Format(time.RFC3339),
    }
}

// Reset resets all metrics
func (c *Collector) Reset() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    c.jobsSubmitted = 0
    c.jobsSucceeded = 0
    c.jobsFailed = 0
    c.totalProcessingTime = 0
    c.processedJobs = 0
    c.lastResetTime = time.Now()
}

// GetPeriodMetrics returns metrics since the last reset
func (c *Collector) GetPeriodMetrics() map[string]interface{} {
    metrics := c.GetMetrics()
    
    c.mu.RLock()
    periodDuration := time.Since(c.lastResetTime)
    c.mu.RUnlock()
    
    metrics["period_duration"] = periodDuration.String()
    metrics["period_start"] = c.lastResetTime.Format(time.RFC3339)
    
    return metrics
}

// PrintMetrics returns a formatted string of current metrics
func (c *Collector) PrintMetrics() string {
    metrics := c.GetMetrics()
    
    return fmt.Sprintf(`
Pipeline Metrics:
  Jobs Submitted: %d
  Jobs Succeeded: %d
  Jobs Failed: %d
  Success Rate: %.2f%%
  Throughput: %.2f jobs/sec
  Avg Processing Duration: %s
  Total Processing Time: %s
  Uptime: %s
`, 
        metrics["jobs_submitted"],
        metrics["jobs_succeeded"], 
        metrics["jobs_failed"],
        metrics["success_rate_percent"],
        metrics["throughput_jobs_per_sec"],
        metrics["avg_processing_duration"],
        metrics["total_processing_time"],
        metrics["uptime"])
}
