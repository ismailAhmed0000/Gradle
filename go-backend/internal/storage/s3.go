package storage

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Storage struct {
	client       *minio.Client
	publicClient *minio.Client
	bucket       string
}

// NewS3Storage sets up an S3 client for the backend's own reads/writes
// (endpointURL) and a second client for signing URLs handed to clients
// (publicURL) — they can differ, e.g. the backend reaches MinIO via
// "localhost:9000" while an Android emulator needs "10.0.2.2:9000" for the
// same server. If publicURL is empty, it falls back to endpointURL.
func NewS3Storage(endpointURL, publicURL, accessKey, secretKey, bucket, region string, virtualHost bool) (*S3Storage, error) {
	client, err := newMinioClient(endpointURL, accessKey, secretKey, region, virtualHost)
	if err != nil {
		return nil, fmt.Errorf("creating s3 client: %w", err)
	}

	if publicURL == "" {
		publicURL = endpointURL
	}
	publicClient, err := newMinioClient(publicURL, accessKey, secretKey, region, virtualHost)
	if err != nil {
		return nil, fmt.Errorf("creating public s3 client: %w", err)
	}

	return &S3Storage{client: client, publicClient: publicClient, bucket: bucket}, nil
}

func newMinioClient(endpointURL, accessKey, secretKey, region string, virtualHost bool) (*minio.Client, error) {
	secure := strings.HasPrefix(endpointURL, "https://")
	endpoint := strings.TrimPrefix(strings.TrimPrefix(endpointURL, "https://"), "http://")

	// MinIO's own auto-detection only recognizes a handful of well-known
	// providers (AWS, DigitalOcean, ...) and falls back to path-style for
	// anything else — some managed S3-compatible providers (e.g. Railway's
	// bucket storage) only support virtual-host-style addressing, so that
	// has to be requested explicitly rather than relying on auto-detection.
	bucketLookup := minio.BucketLookupAuto
	if virtualHost {
		bucketLookup = minio.BucketLookupDNS
	}

	return minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure:       secure,
		Region:       region,
		BucketLookup: bucketLookup,
	})
}

func (s *S3Storage) PutObject(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error {
	_, err := s.client.PutObject(ctx, s.bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return fmt.Errorf("uploading %q: %w", key, err)
	}
	return nil
}

func (s *S3Storage) PresignedGetURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
	url, err := s.publicClient.PresignedGetObject(ctx, s.bucket, key, expiry, nil)
	if err != nil {
		return "", fmt.Errorf("presigning %q: %w", key, err)
	}
	return url.String(), nil
}
