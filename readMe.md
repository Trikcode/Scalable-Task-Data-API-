# Concurrent Data Pipeline Tool

## Overview

This is a Go-based CLI tool for concurrent data pipeline processing with Kafka and AWS S3 integration. The tool processes multiple data sources concurrently using goroutines and channels, implementing error handling, retries, and monitoring for scalable data processing in real-time applications.

## User Preferences

Preferred communication style: Simple, everyday language.

## System Architecture

### Core Components
- **CLI Interface**: Built with Cobra for command-line interaction
- **Pipeline Manager**: Coordinates worker goroutines and job distribution
- **Worker Pool**: Concurrent workers processing data using goroutines and channels
- **Kafka Integration**: Producer and consumer for message streaming
- **S3 Storage**: AWS S3 client for storing processed results
- **Metrics Collection**: Performance monitoring and throughput tracking
- **Structured Logging**: JSON-formatted logging with logrus

### Data Processing Flow
1. **Data Ingestion**: From multiple sources (Kafka, files, APIs)
2. **Concurrent Processing**: Worker goroutines process messages in parallel
3. **Result Storage**: Processed data stored in AWS S3 with hierarchical organization
4. **Monitoring**: Real-time metrics and result notifications via Kafka

### Configuration Management
- YAML-based configuration with environment variable overrides
- Support for SASL authentication and TLS for Kafka
- AWS credentials from environment variables or IAM roles
- Configurable worker count, batch sizes, and retry policies

## External Dependencies

### Message Streaming
- **Kafka**: Apache Kafka for message streaming and result notifications
- **Sarama**: Go client library for Kafka integration
- Support for SASL/SCRAM authentication and TLS encryption

### Cloud Storage
- **AWS S3**: Object storage for processed results and error logs
- **AWS SDK v2**: Official AWS SDK for Go with credential chain support
- Custom endpoint support for S3-compatible storage (MinIO)

### CLI and Configuration
- **Cobra**: Command-line interface framework
- **Viper**: Configuration management with multiple source support
- **Logrus**: Structured logging with JSON output

### Concurrency and Sync
- **golang.org/x/sync**: Extended synchronization primitives
- Native Go goroutines and channels for concurrent processing

## Usage Examples

### Basic Kafka Processing
```bash
./pipeline process --source kafka --topic data-input --workers 10
```

### File Processing
```bash
./pipeline process --source file data1.json data2.json --batch-size 50
```

### API Data Processing
```bash
./pipeline process --source api --s3-bucket my-bucket --timeout 60s
```

## Current Implementation Status

✅ **Completed Features:**
- CLI interface with help system
- Configuration management
- Kafka producer/consumer implementation
- AWS S3 client with retry logic
- Worker pool with concurrent processing
- Metrics collection and monitoring
- Error handling and retry mechanisms
- File and API data source support

## Recent Changes (2025-08-21)

- Fixed Go version compatibility (downgraded from 1.21 to 1.19)
- Resolved compilation issues in logger, Kafka consumer, and S3 client
- Successfully built and tested CLI functionality
- Verified help system and command structure
- Implemented proper error handling for invalid configurations