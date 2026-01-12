package s3

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"lick-scroll/pkg/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Client struct {
	s3Client *s3.S3
	bucket   string
}

func NewClient(cfg *config.Config) (*Client, error) {
	awsConfig := &aws.Config{
		Region: aws.String(cfg.AWSRegion),
		Credentials: credentials.NewStaticCredentials(
			cfg.AWSAccessKeyID,
			cfg.AWSSecretAccessKey,
			"",
		),
	}

	// Support MinIO for local development
	if cfg.AWSEndpoint != "" {
		awsConfig.Endpoint = aws.String(cfg.AWSEndpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
		if cfg.S3UseSSL == "false" {
			awsConfig.DisableSSL = aws.Bool(true)
		}
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := &Client{
		s3Client: s3.New(sess),
		bucket:   cfg.S3BucketName,
	}

	// Ensure bucket exists (for MinIO)
	_, err = client.s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(cfg.S3BucketName),
	})
	if err != nil {
		// Try to create bucket if it doesn't exist
		_, err = client.s3Client.CreateBucket(&s3.CreateBucketInput{
			Bucket: aws.String(cfg.S3BucketName),
		})
		if err != nil {
			// Ignore error if bucket already exists
		}
	}

	return client, nil
}

func (c *Client) UploadFile(key string, file multipart.File, contentType string) (string, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(buf.Bytes()),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	// Generate URL based on endpoint (MinIO or AWS S3)
	endpoint := aws.StringValue(c.s3Client.Config.Endpoint)
	if endpoint != "" && !strings.Contains(endpoint, "amazonaws.com") {
		// MinIO URL format
		protocol := "http"
		if c.s3Client.Config.DisableSSL != nil && !*c.s3Client.Config.DisableSSL {
			protocol = "https"
		}
		endpoint = strings.TrimPrefix(endpoint, "http://")
		endpoint = strings.TrimPrefix(endpoint, "https://")
		url := fmt.Sprintf("%s://%s/%s/%s", protocol, endpoint, c.bucket, key)
		return url, nil
	}

	// AWS S3 URL format
	region := aws.StringValue(c.s3Client.Config.Region)
	if region == "" {
		region = "us-east-1"
	}
	url := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, region, key)
	return url, nil
}

func (c *Client) DeleteFile(key string) error {
	_, err := c.s3Client.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file from S3: %w", err)
	}
	return nil
}

