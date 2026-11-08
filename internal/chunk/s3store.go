package chunk

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config configures an S3-compatible chunk store.
type S3Config struct {
	Bucket   string
	Region   string
	Endpoint string // optional; set for MinIO/R2/other S3-compatible services (path-style addressing)
	Prefix   string // optional; prepended to every key
}

// S3Store stores chunks as objects in an S3-compatible bucket. It satisfies
// the same contract as LocalStore: Delete is idempotent, and Get returns an
// error satisfying errors.Is(err, os.ErrNotExist) for a missing key.
type S3Store struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3Store builds an S3Store using the standard AWS credential chain (env
// vars, shared config/credentials files, or an IAM role). Credentials are not
// configured here; set them the way any AWS SDK expects.
func NewS3Store(ctx context.Context, cfg S3Config) (*S3Store, error) {
	if cfg.Bucket == "" {
		return nil, errors.New("chunk: S3 bucket is required")
	}
	var opts []func(*awsconfig.LoadOptions) error
	if cfg.Region != "" {
		opts = append(opts, awsconfig.WithRegion(cfg.Region))
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("chunk: load AWS config: %w", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		if cfg.Endpoint != "" {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		}
	})
	return &S3Store{client: client, bucket: cfg.Bucket, prefix: strings.TrimSuffix(cfg.Prefix, "/")}, nil
}

func (s *S3Store) objectKey(key string) (string, error) {
	if err := validateKey(key); err != nil {
		return "", err
	}
	if s.prefix == "" {
		return key, nil
	}
	return s.prefix + "/" + key, nil
}

// Put writes data to the bucket, overwriting any existing object at key.
func (s *S3Store) Put(key string, data []byte) error {
	ok, err := s.objectKey(key)
	if err != nil {
		return err
	}
	_, err = s.client.PutObject(context.Background(), &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(ok),
		Body:   bytes.NewReader(data),
	})
	if err != nil {
		return fmt.Errorf("chunk: s3 put %q: %w", key, err)
	}
	return nil
}

// Get reads a whole object.
func (s *S3Store) Get(key string) ([]byte, error) {
	ok, err := s.objectKey(key)
	if err != nil {
		return nil, err
	}
	out, err := s.client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(ok),
	})
	if err != nil {
		if isNotFound(err) {
			return nil, fmt.Errorf("chunk: s3 get %q: %w", key, os.ErrNotExist)
		}
		return nil, fmt.Errorf("chunk: s3 get %q: %w", key, err)
	}
	defer out.Body.Close()
	return io.ReadAll(out.Body)
}

// Delete removes the object; deleting a missing key is not an error (S3's
// DeleteObject is already idempotent, but a prior 404 is tolerated too).
func (s *S3Store) Delete(key string) error {
	ok, err := s.objectKey(key)
	if err != nil {
		return err
	}
	_, err = s.client.DeleteObject(context.Background(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(ok),
	})
	if err != nil && !isNotFound(err) {
		return fmt.Errorf("chunk: s3 delete %q: %w", key, err)
	}
	return nil
}

func isNotFound(err error) bool {
	if _, ok := errors.AsType[*types.NoSuchKey](err); ok {
		return true
	}
	if apiErr, ok := errors.AsType[smithy.APIError](err); ok {
		switch apiErr.ErrorCode() {
		case "NoSuchKey", "NotFound":
			return true
		}
	}
	return false
}
