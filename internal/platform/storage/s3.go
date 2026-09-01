package storage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	appconfig "github.com/jefersonMarques/apresentacao-viagate/internal/config"
)

type S3 struct {
	bucket     string
	stage      string
	client     *s3.Client
	presigner  *s3.PresignClient
	encryption string
	kmsKeyID   string
}

func NewS3(ctx context.Context, cfg appconfig.S3Config) (*S3, error) {
	stage := strings.ToLower(strings.TrimSpace(cfg.Stage))
	if stage != "dev" && stage != "prod" {
		return nil, fmt.Errorf("invalid S3 stage %q", cfg.Stage)
	}

	loadOptions := []func(*config.LoadOptions) error{
		config.WithRegion(cfg.Region),
	}
	if cfg.AccessKeyID != "" || cfg.SecretAccessKey != "" {
		loadOptions = append(loadOptions, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		))
	}

	awsConfig, err := config.LoadDefaultConfig(ctx, loadOptions...)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	// A partir do service/s3 v1.74.1 o SDK passou a adicionar CRC32 por padrão
	// em uploads. Alguns provedores S3 compatíveis não suportam corretamente o
	// checksum/trailer automático e respondem XAmzContentSHA256Mismatch.
	// Para endpoints customizados mantemos a assinatura SigV4 normal e calculamos
	// checksums adicionais somente quando uma operação realmente exigir.
	if cfg.Endpoint != "" {
		awsConfig.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		awsConfig.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	}

	client := s3.NewFromConfig(awsConfig, func(options *s3.Options) {
		options.UsePathStyle = cfg.UsePathStyle
		if cfg.Endpoint != "" {
			options.BaseEndpoint = aws.String(strings.TrimRight(cfg.Endpoint, "/"))
		}
	})

	return &S3{
		bucket:     cfg.Bucket,
		stage:      stage,
		client:     client,
		presigner:  s3.NewPresignClient(client),
		encryption: cfg.ServerSideEncryption,
		kmsKeyID:   cfg.KMSKeyID,
	}, nil
}

// StageKey returns the physical object key used in the bucket. All runtime
// objects live under exactly one environment namespace: dev/ or prod/.
// Supplying a key from the opposite namespace is rejected to prevent accidental
// cross-environment reads or writes.
func (s *S3) StageKey(key string) (string, error) {
	key = strings.Trim(strings.TrimSpace(key), "/")
	if key == "" {
		return "", fmt.Errorf("S3 object key is required")
	}
	if strings.Contains(key, `\`) {
		return "", fmt.Errorf("S3 object key cannot contain backslashes")
	}

	segments := strings.Split(key, "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "", fmt.Errorf("invalid S3 object key %q", key)
		}
	}

	if segments[0] == "dev" || segments[0] == "prod" {
		if segments[0] != s.stage {
			return "", fmt.Errorf("S3 object key stage %q does not match configured stage %q", segments[0], s.stage)
		}
		return key, nil
	}
	return s.stage + "/" + key, nil
}

func (s *S3) Put(ctx context.Context, key, contentType string, body io.Reader, size int64) error {
	objectKey, err := s.StageKey(key)
	if err != nil {
		return err
	}
	input := &s3.PutObjectInput{
		Bucket:        aws.String(s.bucket),
		Key:           aws.String(objectKey),
		Body:          body,
		ContentType:   aws.String(contentType),
		ContentLength: aws.Int64(size),
	}

	switch s.encryption {
	case "none":
		// Endpoints S3 compatíveis podem aplicar criptografia no provedor sem
		// aceitar o header AWS ServerSideEncryption.
	case "aws:kms":
		input.ServerSideEncryption = types.ServerSideEncryptionAwsKms
		input.SSEKMSKeyId = aws.String(s.kmsKeyID)
	default:
		input.ServerSideEncryption = types.ServerSideEncryptionAes256
	}

	_, err = s.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("put s3 object %q: %w", objectKey, err)
	}
	return nil
}

func (s *S3) Get(ctx context.Context, key string) (io.ReadCloser, int64, string, error) {
	objectKey, err := s.StageKey(key)
	if err != nil {
		return nil, 0, "", err
	}
	result, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return nil, 0, "", fmt.Errorf("get s3 object %q: %w", objectKey, err)
	}
	contentType := "application/octet-stream"
	if result.ContentType != nil {
		contentType = *result.ContentType
	}
	return result.Body, aws.ToInt64(result.ContentLength), contentType, nil
}

func (s *S3) SignedDownloadURL(ctx context.Context, key string, ttl time.Duration) (*url.URL, error) {
	return s.signedGetURL(ctx, key, "", ttl)
}

func (s *S3) SignedAttachmentURL(ctx context.Context, key, filename string, ttl time.Duration) (*url.URL, error) {
	filename = cleanDownloadFilename(filename)
	disposition := mime.FormatMediaType("attachment", map[string]string{"filename": filename})
	return s.signedGetURL(ctx, key, disposition, ttl)
}

func (s *S3) signedGetURL(ctx context.Context, key, contentDisposition string, ttl time.Duration) (*url.URL, error) {
	objectKey, err := s.StageKey(key)
	if err != nil {
		return nil, err
	}
	input := &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	}
	if contentDisposition != "" {
		input.ResponseContentDisposition = aws.String(contentDisposition)
	}
	result, err := s.presigner.PresignGetObject(ctx, input, s3.WithPresignExpires(ttl))
	if err != nil {
		return nil, fmt.Errorf("presign s3 object %q: %w", objectKey, err)
	}
	return url.Parse(result.URL)
}

func cleanDownloadFilename(value string) string {
	value = filepath.Base(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if r < 32 || r == 127 {
			return -1
		}
		return r
	}, value)
	if value == "" || value == "." {
		return "download"
	}
	return value
}

func (s *S3) Delete(ctx context.Context, key string) error {
	objectKey, err := s.StageKey(key)
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("delete s3 object %q: %w", objectKey, err)
	}
	return nil
}
