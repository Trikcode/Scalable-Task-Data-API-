package s3

import (
    "concurrent-data-pipeline/internal/config"
    "concurrent-data-pipeline/internal/logger"
    "context"
    "fmt"
    "io"
    "strings"
    "time"

    "github.com/aws/aws-sdk-go-v2/aws"
    awsConfig "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/s3"
    "github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Client wraps AWS S3 functionality
type Client struct {
    s3Client *s3.Client
    config   *config.S3Config
}

// Object represents an S3 object
type Object struct {
    Key          string
    Size         int64
    LastModified time.Time
    ETag         string
    StorageClass string
}

// UploadInput represents parameters for uploading to S3
type UploadInput struct {
    Bucket      string
    Key         string
    Body        io.Reader
    ContentType string
    Metadata    map[string]string
}

// DownloadInput represents parameters for downloading from S3
type DownloadInput struct {
    Bucket string
    Key    string
}

// NewClient creates a new S3 client
func NewClient(cfg config.S3Config) (*Client, error) {
    log := logger.GetLogger()
    
    // Load AWS configuration
    var awsCfg aws.Config
    var err error
    
    if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
        // Use static credentials
        awsCfg, err = awsConfig.LoadDefaultConfig(context.TODO(),
            awsConfig.WithRegion(cfg.Region),
            awsConfig.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
                return aws.Credentials{
                    AccessKeyID:     cfg.AccessKeyID,
                    SecretAccessKey: cfg.SecretAccessKey,
                }, nil
            })),
        )
    } else {
        // Use default credential chain (IAM roles, environment variables, etc.)
        awsCfg, err = awsConfig.LoadDefaultConfig(context.TODO(),
            awsConfig.WithRegion(cfg.Region),
        )
    }
    
    if err != nil {
        return nil, fmt.Errorf("failed to load AWS configuration: %w", err)
    }
    
    // Create S3 client
    s3Client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
        if cfg.Endpoint != "" {
            o.BaseEndpoint = aws.String(cfg.Endpoint)
            o.UsePathStyle = true // Required for custom endpoints like MinIO
        }
    })
    
    client := &Client{
        s3Client: s3Client,
        config:   &cfg,
    }
    
    // Test connection by listing buckets
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    _, err = s3Client.ListBuckets(ctx, &s3.ListBucketsInput{})
    if err != nil {
        log.WithError(err).Warn("Failed to test S3 connection, but continuing")
    } else {
        log.WithFields(map[string]interface{}{
            "region": cfg.Region,
            "bucket": cfg.Bucket,
        }).Info("S3 client created successfully")
    }
    
    return client, nil
}

// Upload uploads data to S3
func (c *Client) Upload(ctx context.Context, input *UploadInput) error {
    log := logger.GetLogger()
    
    putInput := &s3.PutObjectInput{
        Bucket: aws.String(input.Bucket),
        Key:    aws.String(input.Key),
        Body:   input.Body,
    }
    
    if input.ContentType != "" {
        putInput.ContentType = aws.String(input.ContentType)
    }
    
    if len(input.Metadata) > 0 {
        putInput.Metadata = input.Metadata
    }
    
    // Perform upload with retry logic
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        _, err := c.s3Client.PutObject(ctx, putInput)
        if err == nil {
            log.WithFields(map[string]interface{}{
                "bucket": input.Bucket,
                "key":    input.Key,
            }).Debug("Successfully uploaded object to S3")
            return nil
        }
        
        lastErr = err
        log.WithError(err).WithField("attempt", attempt+1).Warn("Failed to upload to S3, retrying")
        
        // Exponential backoff
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    
    return fmt.Errorf("failed to upload to S3 after 3 attempts: %w", lastErr)
}

// Download downloads data from S3
func (c *Client) Download(ctx context.Context, input *DownloadInput) ([]byte, error) {
    log := logger.GetLogger()
    
    getInput := &s3.GetObjectInput{
        Bucket: aws.String(input.Bucket),
        Key:    aws.String(input.Key),
    }
    
    // Perform download with retry logic
    var lastErr error
    for attempt := 0; attempt < 3; attempt++ {
        result, err := c.s3Client.GetObject(ctx, getInput)
        if err == nil {
            defer result.Body.Close()
            
            data, err := io.ReadAll(result.Body)
            if err != nil {
                return nil, fmt.Errorf("failed to read S3 object body: %w", err)
            }
            
            log.WithFields(map[string]interface{}{
                "bucket": input.Bucket,
                "key":    input.Key,
                "size":   len(data),
            }).Debug("Successfully downloaded object from S3")
            
            return data, nil
        }
        
        lastErr = err
        log.WithError(err).WithField("attempt", attempt+1).Warn("Failed to download from S3, retrying")
        
        // Exponential backoff
        time.Sleep(time.Duration(attempt+1) * time.Second)
    }
    
    return nil, fmt.Errorf("failed to download from S3 after 3 attempts: %w", lastErr)
}

// UploadString uploads a string to S3
func (c *Client) UploadString(ctx context.Context, bucket, key, content string) error {
    return c.Upload(ctx, &UploadInput{
        Bucket:      bucket,
        Key:         key,
        Body:        strings.NewReader(content),
        ContentType: "text/plain",
    })
}

// UploadJSON uploads JSON data to S3
func (c *Client) UploadJSON(ctx context.Context, bucket, key string, data []byte) error {
    return c.Upload(ctx, &UploadInput{
        Bucket:      bucket,
        Key:         key,
        Body:        strings.NewReader(string(data)),
        ContentType: "application/json",
    })
}

// List lists objects in an S3 bucket with optional prefix
func (c *Client) List(ctx context.Context, bucket, prefix string, maxKeys int) ([]*Object, error) {
    log := logger.GetLogger()
    
    listInput := &s3.ListObjectsV2Input{
        Bucket: aws.String(bucket),
    }
    
    if prefix != "" {
        listInput.Prefix = aws.String(prefix)
    }
    
    if maxKeys > 0 {
        listInput.MaxKeys = aws.Int32(int32(maxKeys))
    }
    
    result, err := c.s3Client.ListObjectsV2(ctx, listInput)
    if err != nil {
        return nil, fmt.Errorf("failed to list S3 objects: %w", err)
    }
    
    objects := make([]*Object, 0, len(result.Contents))
    for _, obj := range result.Contents {
        objects = append(objects, &Object{
            Key:          aws.ToString(obj.Key),
            Size:         aws.ToInt64(obj.Size),
            LastModified: aws.ToTime(obj.LastModified),
            ETag:         aws.ToString(obj.ETag),
            StorageClass: string(obj.StorageClass),
        })
    }
    
    log.WithFields(map[string]interface{}{
        "bucket": bucket,
        "prefix": prefix,
        "count":  len(objects),
    }).Debug("Listed S3 objects")
    
    return objects, nil
}

// Delete deletes an object from S3
func (c *Client) Delete(ctx context.Context, bucket, key string) error {
    log := logger.GetLogger()
    
    deleteInput := &s3.DeleteObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    }
    
    _, err := c.s3Client.DeleteObject(ctx, deleteInput)
    if err != nil {
        return fmt.Errorf("failed to delete S3 object: %w", err)
    }
    
    log.WithFields(map[string]interface{}{
        "bucket": bucket,
        "key":    key,
    }).Debug("Deleted S3 object")
    
    return nil
}

// Exists checks if an object exists in S3
func (c *Client) Exists(ctx context.Context, bucket, key string) (bool, error) {
    headInput := &s3.HeadObjectInput{
        Bucket: aws.String(bucket),
        Key:    aws.String(key),
    }
    
    _, err := c.s3Client.HeadObject(ctx, headInput)
    if err != nil {
        // Check if it's a "not found" error
        if strings.Contains(err.Error(), "NoSuchKey") {
            return false, nil
        }
        return false, fmt.Errorf("failed to check S3 object existence: %w", err)
    }
    
    return true, nil
}

// CreateBucket creates an S3 bucket if it doesn't exist
func (c *Client) CreateBucket(ctx context.Context, bucket string) error {
    log := logger.GetLogger()
    
    // Check if bucket already exists
    _, err := c.s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
        Bucket: aws.String(bucket),
    })
    if err == nil {
        log.WithField("bucket", bucket).Debug("Bucket already exists")
        return nil
    }
    
    // Create bucket
    createInput := &s3.CreateBucketInput{
        Bucket: aws.String(bucket),
    }
    
    // Add location constraint for regions other than us-east-1
    if c.config.Region != "us-east-1" {
        createInput.CreateBucketConfiguration = &types.CreateBucketConfiguration{
            LocationConstraint: types.BucketLocationConstraint(c.config.Region),
        }
    }
    
    _, err = c.s3Client.CreateBucket(ctx, createInput)
    if err != nil {
        return fmt.Errorf("failed to create S3 bucket: %w", err)
    }
    
    log.WithField("bucket", bucket).Info("Created S3 bucket")
    return nil
}

// GetMetrics returns S3 client metrics
func (c *Client) GetMetrics() map[string]interface{} {
    return map[string]interface{}{
        "region":   c.config.Region,
        "bucket":   c.config.Bucket,
        "endpoint": c.config.Endpoint,
        "status":   "connected",
    }
}
