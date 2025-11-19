package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type R2Client struct {
	client     *s3.Client
	bucketName string
}

// NewR2Client creates a new R2 storage client
func NewR2Client(accountID, accessKeyID, secretAccessKey, bucketName string) (*R2Client, error) {
	r2Endpoint := fmt.Sprintf("https://%s.r2.cloudflarestorage.com", accountID)

	// Create custom credentials provider
	credProvider := credentials.NewStaticCredentialsProvider(
		accessKeyID,
		secretAccessKey,
		"", // R2 doesn't use session tokens
	)

	// Create custom configuration for R2
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("auto"), // R2 uses "auto" as region
		config.WithCredentialsProvider(credProvider),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Create S3 client with R2 endpoint
	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(r2Endpoint)
		o.UsePathStyle = true // R2 requires path-style addressing
	})

	return &R2Client{
		client:     client,
		bucketName: bucketName,
	}, nil
}

// UploadFile uploads a file to R2
func (r *R2Client) UploadFile(ctx context.Context, key string, file io.Reader, contentType string) error {
	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(r.bucketName),
		Key:         aws.String(key),
		Body:        file,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

// UploadMultipartFile uploads a multipart file to R2
func (r *R2Client) UploadMultipartFile(ctx context.Context, key string, file *multipart.FileHeader) error {
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	// Read file content
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, src); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine content type
	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	return r.UploadFile(ctx, key, buf, contentType)
}

// DownloadFile downloads a file from R2
func (r *R2Client) DownloadFile(ctx context.Context, key string) ([]byte, error) {
	result, err := r.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer result.Body.Close()

	return io.ReadAll(result.Body)
}

// DeleteFile deletes a file from R2
func (r *R2Client) DeleteFile(ctx context.Context, key string) error {
	_, err := r.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	return nil
}

// ListFiles lists files in the bucket with optional prefix
func (r *R2Client) ListFiles(ctx context.Context, prefix string) ([]string, error) {
	var files []string

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(r.bucketName),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	result, err := r.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	for _, object := range result.Contents {
		files = append(files, *object.Key)
	}

	return files, nil
}

// ListFilesPage lists files with pagination support using continuation tokens
func (r *R2Client) ListFilesPage(ctx context.Context, prefix string, limit int32, continuationToken string) ([]string, string, bool, error) {
	input := &s3.ListObjectsV2Input{
		Bucket:  aws.String(r.bucketName),
		MaxKeys: aws.Int32(limit),
	}
	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}
	if continuationToken != "" {
		input.ContinuationToken = aws.String(continuationToken)
	}

	result, err := r.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, "", false, fmt.Errorf("failed to list files: %w", err)
	}

	keys := make([]string, 0, len(result.Contents))
	for _, object := range result.Contents {
		keys = append(keys, *object.Key)
	}

	nextCursor := ""
	if result.NextContinuationToken != nil {
		nextCursor = *result.NextContinuationToken
	}

	return keys, nextCursor, *result.IsTruncated, nil
}

// GetPresignedURL generates a presigned URL for downloading
func (r *R2Client) GetPresignedURL(ctx context.Context, key string, expireMinutes int64) (string, error) {
	presignClient := s3.NewPresignClient(r.client)

	request, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(r.bucketName),
		Key:    aws.String(key),
	}, func(opts *s3.PresignOptions) {
		opts.Expires = time.Duration(expireMinutes) * time.Minute // Convert minutes to time.Duration
	})
	if err != nil {
		return "", fmt.Errorf("failed to create presigned URL: %w", err)
	}

	return request.URL, nil
}
