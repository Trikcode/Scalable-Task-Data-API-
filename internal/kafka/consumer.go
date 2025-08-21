package kafka

import (
    "concurrent-data-pipeline/internal/config"
    "concurrent-data-pipeline/internal/logger"
    "context"
    "fmt"
    "sync"
    "time"

    "github.com/Shopify/sarama"
)

// Consumer wraps Kafka consumer functionality
type Consumer struct {
    consumerGroup sarama.ConsumerGroup
    config        *config.KafkaConfig
    topics        []string
    ready         chan bool
    ctx           context.Context
    cancel        context.CancelFunc
}

// Message represents a consumed Kafka message
type Message struct {
    Topic     string
    Partition int32
    Offset    int64
    Key       string
    Value     []byte
    Headers   map[string]string
    Timestamp time.Time
}

// NewConsumer creates a new Kafka consumer
func NewConsumer(cfg config.KafkaConfig) (*Consumer, error) {
    log := logger.GetLogger()
    
    // Create Sarama configuration
    config := sarama.NewConfig()
    config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
    config.Consumer.Offsets.Initial = sarama.OffsetNewest
    config.Consumer.Group.Session.Timeout = 10 * time.Second
    config.Consumer.Group.Heartbeat.Interval = 3 * time.Second
    config.Consumer.MaxProcessingTime = 30 * time.Second
    config.Consumer.Return.Errors = true
    
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
    
    // Create consumer group
    consumerGroup, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroup, config)
    if err != nil {
        return nil, fmt.Errorf("failed to create Kafka consumer group: %w", err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    
    consumer := &Consumer{
        consumerGroup: consumerGroup,
        config:        &cfg,
        ready:         make(chan bool),
        ctx:           ctx,
        cancel:        cancel,
    }
    
    log.WithFields(map[string]interface{}{
        "brokers":        cfg.Brokers,
        "consumer_group": cfg.ConsumerGroup,
    }).Info("Kafka consumer created successfully")
    
    return consumer, nil
}

// Subscribe subscribes to the specified topics
func (c *Consumer) Subscribe(topics ...string) error {
    log := logger.GetLogger()
    c.topics = topics
    
    log.WithField("topics", topics).Info("Subscribed to Kafka topics")
    return nil
}

// ConsumeMessages consumes messages from subscribed topics
func (c *Consumer) ConsumeMessages(ctx context.Context, maxMessages int) ([]*Message, error) {
    log := logger.GetLogger()
    
    if len(c.topics) == 0 {
        return nil, fmt.Errorf("no topics subscribed")
    }
    
    messages := make([]*Message, 0, maxMessages)
    messagesChan := make(chan *Message, maxMessages)
    errorsChan := make(chan error, 1)
    
    // Create consumer handler
    handler := &consumerGroupHandler{
        messagesChan: messagesChan,
        maxMessages:  maxMessages,
        ready:        c.ready,
    }
    
    // Start consuming in a goroutine
    var wg sync.WaitGroup
    wg.Add(1)
    go func() {
        defer wg.Done()
        for {
            if err := c.consumerGroup.Consume(ctx, c.topics, handler); err != nil {
                select {
                case errorsChan <- err:
                case <-ctx.Done():
                    return
                }
                return
            }
            
            if ctx.Err() != nil {
                return
            }
        }
    }()
    
    // Wait for consumer to be ready
    select {
    case <-c.ready:
        log.Debug("Kafka consumer is ready")
    case <-time.After(10 * time.Second):
        return nil, fmt.Errorf("timeout waiting for consumer to be ready")
    case <-ctx.Done():
        return nil, ctx.Err()
    }
    
    // Collect messages with timeout
    timeout := time.After(5 * time.Second)
    for len(messages) < maxMessages {
        select {
        case msg := <-messagesChan:
            messages = append(messages, msg)
        case err := <-errorsChan:
            log.WithError(err).Error("Error consuming messages")
            return messages, err
        case <-timeout:
            // Return whatever messages we have collected
            return messages, nil
        case <-ctx.Done():
            return messages, ctx.Err()
        }
    }
    
    return messages, nil
}

// Close closes the consumer
func (c *Consumer) Close() error {
    log := logger.GetLogger()
    log.Info("Closing Kafka consumer")
    
    c.cancel()
    
    return c.consumerGroup.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler
type consumerGroupHandler struct {
    messagesChan chan *Message
    maxMessages  int
    ready        chan bool
    messageCount int
}

func (h *consumerGroupHandler) Setup(sarama.ConsumerGroupSession) error {
    close(h.ready)
    return nil
}

func (h *consumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error {
    return nil
}

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
    log := logger.GetLogger()
    
    for {
        select {
        case kafkaMsg := <-claim.Messages():
            if kafkaMsg == nil {
                return nil
            }
            
            // Convert headers
            headers := make(map[string]string)
            for _, header := range kafkaMsg.Headers {
                headers[string(header.Key)] = string(header.Value)
            }
            
            // Create our message
            msg := &Message{
                Topic:     kafkaMsg.Topic,
                Partition: kafkaMsg.Partition,
                Offset:    kafkaMsg.Offset,
                Key:       string(kafkaMsg.Key),
                Value:     kafkaMsg.Value,
                Headers:   headers,
                Timestamp: kafkaMsg.Timestamp,
            }
            
            // Send to channel (non-blocking)
            select {
            case h.messagesChan <- msg:
                h.messageCount++
                log.WithFields(map[string]interface{}{
                    "topic":     msg.Topic,
                    "partition": msg.Partition,
                    "offset":    msg.Offset,
                }).Debug("Message consumed")
                
                // Mark message as processed
                session.MarkMessage(kafkaMsg, "")
                
                // Check if we've reached max messages
                if h.messageCount >= h.maxMessages {
                    return nil
                }
            default:
                // Channel is full, mark and continue
                session.MarkMessage(kafkaMsg, "")
            }
            
        case <-session.Context().Done():
            return nil
        }
    }
}

// GetMetrics returns consumer metrics
func (c *Consumer) GetMetrics() map[string]interface{} {
    return map[string]interface{}{
        "brokers":        c.config.Brokers,
        "consumer_group": c.config.ConsumerGroup,
        "topics":         c.topics,
        "status":         "connected",
    }
}
