package aws

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/adeel450/terraform-drift-detector/internal/tfstate"
)

func init() {
	// Make "s3://bucket/key" usable as a Terraform state source whenever the
	// AWS provider is compiled in.
	tfstate.RegisterBackend("s3", newS3Source)
}

// s3Source reads Terraform state from an S3 object.
type s3Source struct {
	bucket string
	key    string
	region string
}

// newS3Source parses an "s3://bucket/key" reference.
func newS3Source(ref, region string) (tfstate.StateSource, error) {
	rest := strings.TrimPrefix(ref, "s3://")
	bucket, key, ok := strings.Cut(rest, "/")
	if !ok || bucket == "" || key == "" {
		return nil, fmt.Errorf("invalid s3 state reference %q (want s3://bucket/key)", ref)
	}
	return &s3Source{bucket: bucket, key: key, region: region}, nil
}

// Read implements tfstate.StateSource.
func (s *s3Source) Read(ctx context.Context) ([]byte, string, error) {
	cfg, err := loadConfig(ctx, s.region)
	if err != nil {
		return nil, "", err
	}
	client := s3.NewFromConfig(cfg)
	out, err := client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(s.key),
	})
	if err != nil {
		return nil, "", fmt.Errorf("get s3://%s/%s: %w", s.bucket, s.key, err)
	}
	defer out.Body.Close()
	data, err := io.ReadAll(out.Body)
	if err != nil {
		return nil, "", fmt.Errorf("read s3://%s/%s: %w", s.bucket, s.key, err)
	}
	return data, fmt.Sprintf("s3://%s/%s", s.bucket, s.key), nil
}
