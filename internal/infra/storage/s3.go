package storage

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"pkl-golang-be/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Client struct {
	client        *s3.Client
	presignClient *s3.PresignClient
	bucket        string
	publicPrefix  string
}

func NewS3Client(cfg *config.Config) (*S3Client, error) {
	customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...any) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               cfg.S3Endpoint,
			HostnameImmutable: true,
			SigningRegion:     cfg.S3Region,
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			cfg.S3AccessKey,
			cfg.S3SecretKey,
			"",
		)),
		awsconfig.WithEndpointResolverWithOptions(customResolver),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = cfg.S3UsePathStyle
	})

	presignClient := s3.NewPresignClient(client)

	slog.Info("Successfully initialized S3 Storage Client",
		"endpoint", cfg.S3Endpoint,
		"bucket", cfg.S3Bucket,
		"region", cfg.S3Region,
	)

	return &S3Client{
		client:        client,
		presignClient: presignClient,
		bucket:        cfg.S3Bucket,
		publicPrefix:  strings.TrimRight(cfg.S3PublicURLPrefix, "/"),
	}, nil
}

func (s *S3Client) Bucket() string {
	return s.bucket
}

func (s *S3Client) PresignPutObject(ctx context.Context, key, contentType string, contentLength int64, lifetime time.Duration) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(key),
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(contentLength),
	}

	req, err := s.presignClient.PresignPutObject(ctx, input, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", fmt.Errorf("failed to presign PUT object: %w", err)
	}

	return req.URL, nil
}

func (s *S3Client) PresignGetObject(ctx context.Context, key string, lifetime time.Duration) (string, error) {
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	req, err := s.presignClient.PresignGetObject(ctx, input, s3.WithPresignExpires(lifetime))
	if err != nil {
		return "", fmt.Errorf("failed to presign GET object: %w", err)
	}

	return req.URL, nil
}

func (s *S3Client) DeleteObject(ctx context.Context, key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object '%s' from S3: %w", key, err)
	}

	return nil
}

func (s *S3Client) GetPublicURL(key string) string {
	cleanKey := strings.TrimLeft(key, "/")
	return fmt.Sprintf("%s/%s", s.publicPrefix, cleanKey)
}

// EnsureBucket verifies if the target bucket exists, and creates it if missing.
func (s *S3Client) EnsureBucket(ctx context.Context) error {
	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if err == nil {
		slog.Debug("S3 storage bucket already exists", "bucket", s.bucket)
		return nil
	}

	slog.Info("S3 storage bucket not found or unreachable, attempting to create...", "bucket", s.bucket)
	_, createErr := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})
	if createErr != nil {
		errStr := createErr.Error()
		if strings.Contains(errStr, "BucketAlreadyOwnedByYou") ||
			strings.Contains(errStr, "BucketAlreadyExists") ||
			strings.Contains(errStr, "Conflict") {
			slog.Debug("S3 storage bucket already exists (reported during creation)", "bucket", s.bucket)
			return nil
		}
		return fmt.Errorf("failed to ensure S3 bucket '%s': %w", s.bucket, createErr)
	}

	slog.Info("✅ Successfully created S3 storage bucket", "bucket", s.bucket)
	return nil
}
