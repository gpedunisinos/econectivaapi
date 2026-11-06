// Package storagebackend stores private attachments without exposing AWS credentials.
package storagebackend

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials/stscreds"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

// Store is the narrow boundary used by attachment handlers and integration tests.
type Store interface {
	Put(context.Context, string, []byte, string, string) error
	URL(context.Context, string, string) (string, error)
	Delete(context.Context, string) error
	Bucket() string
}

type bucketStore struct {
	client *s3.Client
	bucket string
}

func New(ctx context.Context) (Store, error) {
	bucket := strings.TrimSpace(os.Getenv("S3_BUCKET"))
	if bucket == "" {
		return nil, errors.New("S3_BUCKET is required")
	}
	region := os.Getenv("AWS_REGION")
	if region == "" {
		region = "sa-east-1"
	}
	cfg, err := awsconfig.LoadDefaultConfig(ctx, awsconfig.WithRegion(region))
	if err != nil {
		return nil, err
	}
	if role := strings.TrimSpace(os.Getenv("S3_ROLE_ARN")); role != "" {
		// Assume the platform's attachment role; never fall back on failure.
		cfg.Credentials = aws.NewCredentialsCache(stscreds.NewAssumeRoleProvider(sts.NewFromConfig(cfg), role))
	}
	// The default credential chain uses the EC2 instance role in production.
	return &bucketStore{client: s3.NewFromConfig(cfg), bucket: bucket}, nil
}

func (s *bucketStore) Bucket() string { return s.bucket }
func validKey(key string) bool {
	return strings.HasPrefix(key, "files/") && !strings.Contains(key, "..") && !strings.ContainsAny(key, "\\\r\n")
}
func (s *bucketStore) Put(ctx context.Context, key string, data []byte, contentType, name string) error {
	if !validKey(key) {
		return errors.New("invalid object key")
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), Body: bytes.NewReader(data), ContentLength: aws.Int64(int64(len(data))),
		ContentType: aws.String(contentType), ContentDisposition: aws.String(mime.FormatMediaType("attachment", map[string]string{"filename": name})),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
	})
	return err
}
func (s *bucketStore) URL(ctx context.Context, key, name string) (string, error) {
	if !validKey(key) {
		return "", errors.New("invalid object key")
	}
	signed, err := s3.NewPresignClient(s.client).PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key),
		ResponseContentDisposition: aws.String(mime.FormatMediaType("attachment", map[string]string{"filename": name})),
		ResponseCacheControl:       aws.String("private, no-store"),
	}, func(o *s3.PresignOptions) { o.Expires = 5 * time.Minute })
	if err != nil {
		return "", err
	}
	return signed.URL, nil
}
func (s *bucketStore) Delete(ctx context.Context, key string) error {
	if !validKey(key) {
		return fmt.Errorf("invalid object key")
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.bucket), Key: aws.String(key)})
	return err
}
