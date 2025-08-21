package kafka

import (
    "concurrent-data-pipeline/internal/config"
    "concurrent-data-pipeline/internal/logger"
    "fmt"
    "time"

    "github.com/Shopify/sarama"
)

// Producer wraps Kafka producer functionality
type Producer struct {
    producer sarama.SyncProducer
    config   *config.KafkaConfig
}

// Message represents a Kafka message to be produced
type ProduceMessage struct {
    Topic     string
    Key       string
    Value     []byte
    Headers   map[string]string
    Partition int32
}

// NewProducer creates a new Kafka producer
func NewProducer(cfg config.KafkaConfig) (*Producer, error) {
    log := logger.GetLogger()
    
    // Create Sarama configuration
    config := sarama.NewConfig()
    config.Producer.RequiredAcks = sarama.WaitForAll
    config.Producer.Retry.Max = 3
    config.Producer.Return.Successes = true
    config.Producer.Timeout = 30 * time.Second
    config.Producer.Compression = sarama.CompressionSnappy
    config.Producer.Flush.Frequency = 500 * time.Millisecond
    
    // Configure authentication if needed
    if cfg.SecurityProtocol == "SASL_SSL" || cfg.SecurityProtocol == "SASL_PLAINTEXT" {
        config.Net.SASL.Enable = true
        config.Net.SASL.User = cfg.SASLUsername
        config.Net.SASL.Password = cfg.SASLPassword
        
        switch cfg.SASLMechanism {
        case "PLAIN":
            config.Net.SASL.Mechanism = sarama.SASLTypePlaintext
        case "SCRAM-SHA-256":
            config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA256
        case "SCRAM-SHA-512":
            config.Net.SASL.Mechanism = sarama.SASLTypeSCRAMSHA512
        }
    }
    
    if cfg.SecurityProtocol == "SSL" || cfg.SecurityProtocol == "SASL_SSL" {
        config.Net.TLS.Enable = true
    }
    
    // Create producer
    producer, err := sarama.NewSyncProducer(cfg.Brokers, config)
    if err != nil {
        return nil, fmt.Errorf("failed to create Kafka producer: %w", err)
    }
    
    log.WithField("brokers", cfg.Brokers).Info("Kafka producer created successfully")
    
    return &Producer{
        producer: producer,
        config:   &cfg,
    }, nil
}

// SendMessage sends a single message to Kafka
func (p *Producer) SendMessage(msg *ProduceMessage) error {
    log := logger.GetLogger()
    
    // Create Sarama message
    kafkaMsg := &sarama.ProducerMessage{
        Topic: msg.Topic,
        Value: sarama.ByteEncoder(msg.Value),
    }
    
    // Add key if provided
    if msg.Key != "" {
        kafkaMsg.Key = sarama.StringEncoder(msg.Key)
    }
    
    // Add partition if specified
    if msg.Partition >= 0 {
        kafkaMsg.Partition = msg.Partition
    }
    
    // Add headers if provided
    if len(msg.Headers) > 0 {
        var headers []sarama.RecordHeader
        for key, value := range msg.Headers {
            headers = append(headers, sarama.RecordHeader{
                Key:   []byte(key),
                Value: []byte(value),
            })
        }
        kafkaMsg.Headers = headers
    }
    
    // Send message with retry logic
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        partition, offset, err := p.producer.SendMessage(kafkaMsg)
        if err == nil {
            log.WithFields(map[string]interface{}{
                "topic":     msg.Topic,
                "partition": partition,
                "offset":    offset,
                "key":       msg.Key,
            }).Debug("Message sent successfully")
            return nil
        }
        
        lastErr = err
        log.WithError(err).WithField("attempt", attempt+1).Warn("Failed to send message, retrying")
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    
    return fmt.Errorf("failed to send message after 3 attempts: %w", lastErr)
}

// SendBatch sends multiple messages in a batch
func (p *Producer) SendBatch(messages []*ProduceMessage) error {
    log := logger.GetLogger()
    
    if len(messages) == 0 {
        return nil
    }
    
    var successCount, failureCount int
    var lastErr error
    
    for _, msg := range messages {
        if err := p.SendMessage(msg); err != nil {
            failureCount++
            lastErr = err
            log.WithError(err).WithField("topic", msg.Topic).Error("Failed to send message in batch")
        } else {
            successCount++
        }
    }
    
    log.WithFields(map[string]interface{}{
        "total":     len(messages),
        "success":   successCount,
        "failures":  failureCount,
    }).Info("Batch send completed")
    
    if failureCount > 0 {
        return fmt.Errorf("batch send failed for %d/%d messages, last error: %w", 
            failureCount, len(messages), lastErr)
    }
    
    return nil
}

// SendProcessingResult sends a processing result message
func (p *Producer) SendProcessingResult(topic, messageID string, success bool, errorMsg string, data []byte) error {
    headers := map[string]string{
        "message_id": messageID,
        "timestamp":  time.Now().Format(time.RFC3339),
        "success":    fmt.Sprintf("%t", success),
    }
    
    if errorMsg != "" {
        headers["error"] = errorMsg
    }
    
    msg := &ProduceMessage{
        Topic:   topic,
        Key:     messageID,
        Value:   data,
        Headers: headers,
    }
    
    return p.SendMessage(msg)
}

// Close closes the producer
func (p *Producer) Close() error {
    log := logger.GetLogger()
    log.Info("Closing Kafka producer")
    
    return p.producer.Close()
}

// GetMetrics returns producer metrics
func (p *Producer) GetMetrics() map[string]interface{} {
    // In a real implementation, you would collect actual metrics
    return map[string]interface{}{
        "brokers": p.config.Brokers,
        "status":  "connected",
    }
}
