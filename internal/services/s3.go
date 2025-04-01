package services

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"kori/internal/utils/logger"

	"kori/internal/models"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

// Ensure S3Service implements FileURLGenerator
var _ models.FileURLGenerator = (*S3Service)(nil)

type S3Service struct {
	client     *s3.Client
	bucketName string
	endpoint   string
	region     string
	logger     *logger.Logger
	accessKey  string
	secretKey  string
}

func NewS3Service(bucketName, endpoint, region, accessKey, secretKey string) (*S3Service, error) {
	log := logger.New("s3_service")

	// Validate required credentials
	if accessKey == "" || secretKey == "" {
		return nil, log.Error("S3 credentials are empty ❌", fmt.Errorf("accessKey or secretKey is empty"))
	}

	// Create AWS config with explicit credentials
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("apac"),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"", // Session token (not needed for basic auth)
		)),
		config.WithRetryMode(aws.RetryModeStandard),
		config.WithRetryMaxAttempts(3),
	)
	if err != nil {
		return nil, log.Error("Unable to load SDK config ❌", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(fmt.Sprintf("https://%s.%s", region, endpoint))
	})

	// Verify credentials by making a test API call
	_, err = client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return nil, log.Error("Failed to verify S3 credentials ❌", err)
	}

	log.Success("S3 service initialized successfully ✅")

	return &S3Service{
		client:     client,
		bucketName: bucketName,
		endpoint:   endpoint,
		region:     region,
		accessKey:  accessKey,
		secretKey:  secretKey,
		logger:     log,
	}, nil
}

// UploadFile uploads a file to S3 or S3-compatible storage and returns the URL
func (s *S3Service) UploadFile(ctx context.Context, file []byte, filename string, acl types.ObjectCannedACL, contentType string) (string, error) {
	s.logger.Info("📤 Starting file upload: %s", filename)

	// Generate unique filename
	ext := filepath.Ext(filename)

	filename = fmt.Sprintf("%s%s", uuid.New().String(), ext)

	s.logger.Info("🔄 Processing upload for file: %s", filename)

	is_r2 := os.Getenv("STORAGE_PROVIDER") == "r2"

	ACL := acl
	if is_r2 {
		ACL = types.ObjectCannedACLPublicRead
	}

	// Upload to storage
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucketName),
		Key:         aws.String(filename),
		Body:        bytes.NewReader(file),
		ACL:         ACL,
		ContentType: aws.String(contentType),
	})
	if err != nil {
		return "", s.logger.Error("Failed to upload file to storage ❌", err)
	}

	// Generate URL based on endpoint configuration
	var url string
	if s.endpoint != "" {
		// Custom endpoint (e.g., MinIO)
		url = fmt.Sprintf("https://%s.%s/%s/%s", s.region, s.endpoint, s.bucketName, filename)
	} else {
		// AWS S3
		url = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.bucketName, s.region, filename)
	}

	s.logger.Success("✅ File uploaded successfully: %s", url)
	return url, nil
}

// GetSignedURL implements FileURLGenerator interface
func (s *S3Service) GetSignedURL(ctx context.Context, path string, duration time.Duration) (string, error) {
	presignClient := s3.NewPresignClient(s.client)

	s.logger.Info("🔄 Generating pre-signed URL for path: %s", path)

	presignedURL, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(path),
	}, s3.WithPresignExpires(duration))

	if err != nil {
		return "", s.logger.Error("Failed to generate pre-signed URL ❌", err)
	}

	s.logger.Success("✅ Generated pre-signed URL successfully")
	return presignedURL.URL, nil
}
