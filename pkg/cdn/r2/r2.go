package r2

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

// UploadFile uploads a file to Cloudflare R2 with strict structure and returns the public URL, authenticity hash, and R2 key.
func UploadFile(ctx context.Context, fileBytes []byte, fileName, mimeType, folderPrefix string) (publicURL, authenticityHash, key string, err error) {
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	endpoint := os.Getenv("R2_ENDPOINT")
	region := os.Getenv("R2_REGION")
	if region == "" {
		region = "auto"
	}

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return "", "", "", fmt.Errorf("failed to create AWS session: %w", err)
	}

	// Generate unique filename
	uniqueName := fmt.Sprintf("%s_%s", uuid.New().String(), fileName)
	key = fmt.Sprintf("%s/%s", folderPrefix, uniqueName)

	// Calculate authenticity hash (SHA256)
	hash := sha256.Sum256(fileBytes)
	authenticityHash = base64.StdEncoding.EncodeToString(hash[:])

	uploader := s3manager.NewUploader(sess)
	upParams := &s3manager.UploadInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(fileBytes),
		ContentType: aws.String(mimeType),
		ACL:         aws.String("public-read"),
	}

	_, err = uploader.UploadWithContext(ctx, upParams)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to upload to R2: %w", err)
	}

	publicURL = fmt.Sprintf("%s/%s/%s", endpoint, bucket, key)
	return publicURL, authenticityHash, key, nil
}

// GenerateSignedURL creates a signed URL for secure, time-limited access.
func GenerateSignedURL(key string, expiresIn time.Duration) (string, error) {
	accessKey := os.Getenv("R2_ACCESS_KEY_ID")
	secretKey := os.Getenv("R2_SECRET_ACCESS_KEY")
	bucket := os.Getenv("R2_BUCKET")
	endpoint := os.Getenv("R2_ENDPOINT")
	region := os.Getenv("R2_REGION")
	if region == "" {
		region = "auto"
	}

	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		S3ForcePathStyle: aws.Bool(true),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create AWS session: %w", err)
	}

	svc := s3.New(sess)
	req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	urlStr, err := req.Presign(expiresIn)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return urlStr, nil
}
