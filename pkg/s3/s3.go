package s3

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"lick-scroll/pkg/config"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

type Client struct {
	s3Client *s3.S3
	bucket   string
	publicURL string
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

	if cfg.AWSEndpoint != "" {
		awsConfig.Endpoint = aws.String(cfg.AWSEndpoint)
		awsConfig.S3ForcePathStyle = aws.Bool(true)
		awsConfig.DisableSSL = aws.Bool(cfg.S3UseSSL != "true")
	}

	sess, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create AWS session: %w", err)
	}

	client := &Client{
		s3Client: s3.New(sess),
		bucket:   cfg.S3BucketName,
		publicURL: cfg.S3PublicURL,
	}

	if err := client.ensureBucketExists(); err != nil {
		return nil, fmt.Errorf("failed to ensure bucket exists: %w", err)
	}

	return client, nil
}

func (c *Client) ensureBucketExists() error {
	_, err := c.s3Client.HeadBucket(&s3.HeadBucketInput{
		Bucket: aws.String(c.bucket),
	})
	if err == nil {
		return nil
	}

	_, err = c.s3Client.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(c.bucket),
		ACL:    aws.String("public-read"),
	})
	if err != nil {
		awsErr, ok := err.(interface {
			Code() string
			Message() string
		})
		if ok && (awsErr.Code() == "BucketAlreadyOwnedByYou" || awsErr.Code() == "BucketAlreadyExists") {
			return nil
		}
		return fmt.Errorf("failed to create bucket: %w", err)
	}

	_, err = c.s3Client.PutBucketPolicy(&s3.PutBucketPolicyInput{
		Bucket: aws.String(c.bucket),
		Policy: aws.String(fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Effect": "Allow",
				"Principal": {"AWS": ["*"]},
				"Action": ["s3:GetObject"],
				"Resource": ["arn:aws:s3:::%s/*"]
			}]
		}`, c.bucket)),
	})
	if err != nil {
		return fmt.Errorf("failed to set bucket policy: %w", err)
	}

	return nil
}

func (c *Client) UploadFile(key string, reader io.Reader, contentType string) (string, error) {
	var body io.ReadSeeker
	if seeker, ok := reader.(io.ReadSeeker); ok {
		body = seeker
	} else {
		data, err := io.ReadAll(reader)
		if err != nil {
			return "", fmt.Errorf("failed to read file data: %w", err)
		}
		body = bytes.NewReader(data)
	}

	_, err := c.s3Client.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(c.bucket),
		Key:         aws.String(key),
		Body:        body,
		ContentType: aws.String(contentType),
		ACL:         aws.String("public-read"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload file to S3: %w", err)
	}

	if c.publicURL != "" {
		return fmt.Sprintf("%s/%s/%s", c.publicURL, c.bucket, key), nil
	}

	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", c.bucket, "us-east-1", key), nil
}

func (c *Client) GetPresignedURL(key string, duration time.Duration) (string, error) {
	req, _ := c.s3Client.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(c.bucket),
		Key:    aws.String(key),
	})

	url, err := req.Presign(duration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned URL: %w", err)
	}

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
