package config

import (
    "fmt"
    "os"

    "github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
    Workers   int    `mapstructure:"workers"`
    BatchSize int    `mapstructure:"batch_size"`
    LogLevel  string `mapstructure:"log_level"`
    
    Kafka KafkaConfig `mapstructure:"kafka"`
    S3    S3Config    `mapstructure:"s3"`
}

// KafkaConfig holds Kafka-specific configuration
type KafkaConfig struct {
    Brokers       []string `mapstructure:"brokers"`
    ConsumerGroup string   `mapstructure:"consumer_group"`
    SecurityProtocol string `mapstructure:"security_protocol"`
    SASLMechanism    string `mapstructure:"sasl_mechanism"`
    SASLUsername     string `mapstructure:"sasl_username"`
    SASLPassword     string `mapstructure:"sasl_password"`
}

// S3Config holds AWS S3-specific configuration
type S3Config struct {
    Region          string `mapstructure:"region"`
    Bucket          string `mapstructure:"bucket"`
    AccessKeyID     string `mapstructure:"access_key_id"`
    SecretAccessKey string `mapstructure:"secret_access_key"`
    Endpoint        string `mapstructure:"endpoint"`
}

// Load reads configuration from viper and environment variables
func Load() (*Config, error) {
    // Set defaults
    viper.SetDefault("workers", 5)
    viper.SetDefault("batch_size", 100)
    viper.SetDefault("log_level", "info")
    
    viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
    viper.SetDefault("kafka.consumer_group", "data-pipeline-group")
    viper.SetDefault("kafka.security_protocol", "PLAINTEXT")
    
    viper.SetDefault("s3.region", "us-east-1")
    viper.SetDefault("s3.bucket", "data-pipeline-bucket")

    // Load from environment variables with fallbacks
    cfg := &Config{}
    
    // Basic configuration
    cfg.Workers = viper.GetInt("workers")
    cfg.BatchSize = viper.GetInt("batch_size")
    cfg.LogLevel = viper.GetString("log_level")
    
    // Kafka configuration
    cfg.Kafka = KafkaConfig{
        Brokers:       viper.GetStringSlice("kafka.brokers"),
        ConsumerGroup: viper.GetString("kafka.consumer_group"),
        SecurityProtocol: viper.GetString("kafka.security_protocol"),
        SASLMechanism: viper.GetString("kafka.sasl_mechanism"),
        SASLUsername:  getEnvOrConfig("KAFKA_SASL_USERNAME", "kafka.sasl_username"),
        SASLPassword:  getEnvOrConfig("KAFKA_SASL_PASSWORD", "kafka.sasl_password"),
    }
    
    // Override Kafka brokers from environment if set
    if envBrokers := os.Getenv("KAFKA_BROKERS"); envBrokers != "" {
        cfg.Kafka.Brokers = []string{envBrokers}
    }
    
    // S3 configuration
    cfg.S3 = S3Config{
        Region:          viper.GetString("s3.region"),
        Bucket:          getEnvOrConfig("S3_BUCKET", "s3.bucket"),
        AccessKeyID:     getEnvOrConfig("AWS_ACCESS_KEY_ID", "s3.access_key_id"),
        SecretAccessKey: getEnvOrConfig("AWS_SECRET_ACCESS_KEY", "s3.secret_access_key"),
        Endpoint:        getEnvOrConfig("S3_ENDPOINT", "s3.endpoint"),
    }
    
    // Override S3 region from environment if set
    if envRegion := os.Getenv("AWS_REGION"); envRegion != "" {
        cfg.S3.Region = envRegion
    }
    
    // Validate required configuration
    if err := cfg.validate(); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return cfg, nil
}

// getEnvOrConfig gets value from environment variable first, then from viper config
func getEnvOrConfig(envKey, configKey string) string {
    if value := os.Getenv(envKey); value != "" {
        return value
    }
    return viper.GetString(configKey)
}

// validate ensures all required configuration is present
func (c *Config) validate() error {
    if c.Workers <= 0 {
        return fmt.Errorf("workers must be greater than 0")
    }
    
    if c.BatchSize <= 0 {
        return fmt.Errorf("batch_size must be greater than 0")
    }
    
    if len(c.Kafka.Brokers) == 0 {
        return fmt.Errorf("kafka brokers must be specified")
    }
    
    if c.Kafka.ConsumerGroup == "" {
        return fmt.Errorf("kafka consumer group must be specified")
    }
    
    if c.S3.Region == "" {
        return fmt.Errorf("s3 region must be specified")
    }
    
    if c.S3.Bucket == "" {
        return fmt.Errorf("s3 bucket must be specified")
    }
    
    return nil
}

// GetKafkaBrokerString returns brokers as a comma-separated string
func (k *KafkaConfig) GetKafkaBrokerString() string {
    if len(k.Brokers) == 0 {
        return ""
    }
    
    result := k.Brokers[0]
    for i := 1; i < len(k.Brokers); i++ {
        result += "," + k.Brokers[i]
    }
    return result
}
