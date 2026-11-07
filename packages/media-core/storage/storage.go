// Package storage reads and writes media files in S3-compatible object
// storage (RustFS locally, any S3 provider in production). It only uses the
// S3 API, so switching providers is a configuration change (docs/adr/0006).
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/transfermanager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// Config holds the storage settings, usually read with ConfigFromEnv.
type Config struct {
	Endpoint  string // S3 API URL, e.g. http://localhost:9000
	Region    string
	AccessKey string
	SecretKey string
	Bucket    string
	// PublicURL is the base URL browsers use to read public files (the HLS
	// output). Defaults to Endpoint.
	PublicURL string
}

// ConfigFromEnv reads S3_ENDPOINT, S3_REGION, S3_ACCESS_KEY, S3_SECRET_KEY,
// S3_BUCKET and S3_PUBLIC_URL.
func ConfigFromEnv() (Config, error) {
	cfg := Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Region:    os.Getenv("S3_REGION"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		Bucket:    os.Getenv("S3_BUCKET"),
		PublicURL: os.Getenv("S3_PUBLIC_URL"),
	}
	if cfg.Region == "" {
		cfg.Region = "us-east-1"
	}
	if cfg.PublicURL == "" {
		cfg.PublicURL = cfg.Endpoint
	}
	var missing []string
	for name, v := range map[string]string{"S3_ENDPOINT": cfg.Endpoint, "S3_ACCESS_KEY": cfg.AccessKey, "S3_SECRET_KEY": cfg.SecretKey, "S3_BUCKET": cfg.Bucket} {
		if v == "" {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return cfg, fmt.Errorf("missing environment variables: %s", strings.Join(missing, ", "))
	}
	return cfg, nil
}

// Client is a bucket-scoped storage client. Safe for concurrent use.
type Client struct {
	s3      *s3.Client
	presign *s3.PresignClient
	// browserPresign signs links for browsers: same credentials, but the
	// public address (S3_PUBLIC_URL), which differs from the internal one in
	// Docker (http://storage:9000 inside, http://localhost:9000 outside).
	browserPresign *s3.PresignClient
	transfer       *transfermanager.Client
	bucket         string
	publicURL      string
}

// New creates a client. It does not contact the server; call Check for that.
func New(cfg Config) *Client {
	s3Client := s3.NewFromConfig(aws.Config{
		Region:      cfg.Region,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, "")),
	}, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = true // bucket in the path; works with every S3-compatible server
		// Only send/verify checksums when the API requires them: newer SDK
		// defaults (CRC32 on every request) are not supported everywhere.
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.ResponseChecksumValidation = aws.ResponseChecksumValidationWhenRequired
	})
	transfer := transfermanager.New(s3Client, func(o *transfermanager.Options) {
		o.PartSizeBytes = 16 << 20 // 16MB parts: a 2GB upload uses 128 parts
		o.Concurrency = 4
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
	})
	publicURL := strings.TrimRight(cfg.PublicURL, "/")
	if publicURL == "" {
		publicURL = strings.TrimRight(cfg.Endpoint, "/")
	}
	return &Client{
		s3:      s3Client,
		presign: s3.NewPresignClient(s3Client),
		browserPresign: s3.NewPresignClient(s3Client, func(po *s3.PresignOptions) {
			po.ClientOptions = append(po.ClientOptions, func(o *s3.Options) { o.BaseEndpoint = aws.String(publicURL) })
		}),
		transfer:  transfer,
		bucket:    cfg.Bucket,
		publicURL: publicURL,
	}
}

// Check verifies that the bucket exists and the credentials work.
func (c *Client) Check(ctx context.Context) error {
	if _, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &c.bucket}); err != nil {
		return fmt.Errorf("bucket %q not reachable: %w", c.bucket, err)
	}
	return nil
}

// Upload streams body to key. Large bodies are sent as a multipart upload, so
// the size does not need to be known in advance.
func (c *Client) Upload(ctx context.Context, key string, body io.Reader, contentType string) error {
	_, err := c.transfer.UploadObject(ctx, &transfermanager.UploadObjectInput{
		Bucket:      &c.bucket,
		Key:         &key,
		Body:        body,
		ContentType: &contentType,
	})
	if err != nil {
		return fmt.Errorf("upload %s: %w", key, err)
	}
	return nil
}

// UploadFile uploads a local file to key.
func (c *Client) UploadFile(ctx context.Context, key, path, contentType string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return c.Upload(ctx, key, f, contentType)
}

// DownloadFile writes the object at key to a local file at path.
func (c *Client) DownloadFile(ctx context.Context, key, path string) error {
	out, err := c.s3.GetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key})
	if err != nil {
		return fmt.Errorf("download %s: %w", key, err)
	}
	defer out.Body.Close()

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, out.Body); err != nil {
		f.Close()
		return fmt.Errorf("download %s: %w", key, err)
	}
	return f.Close()
}

// Delete removes the object at key. Deleting a missing object is not an error.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &c.bucket, Key: &key})
	if err != nil {
		return fmt.Errorf("delete %s: %w", key, err)
	}
	return nil
}

// Exists reports whether an object exists at key.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	_, err := c.s3.HeadObject(ctx, &s3.HeadObjectInput{Bucket: &c.bucket, Key: &key})
	if err == nil {
		return true, nil
	}
	var notFound interface{ ErrorCode() string }
	if errors.As(err, &notFound) && (notFound.ErrorCode() == "NotFound" || notFound.ErrorCode() == "NoSuchKey") {
		return false, nil
	}
	return false, err
}

// PresignGet returns a temporary URL that reads the object at key without
// credentials. FFmpeg uses it to read only the parts of a large file it needs
// (HTTP range requests) instead of downloading it. Treat the URL as a secret:
// never log it.
func (c *Client) PresignGet(ctx context.Context, key string, expires time.Duration) (string, error) {
	req, err := c.presign.PresignGetObject(ctx, &s3.GetObjectInput{Bucket: &c.bucket, Key: &key},
		s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return req.URL, nil
}

// PresignForBrowser returns a temporary link to a private object for a
// browser, on the public address. With a filename (plain ASCII), the browser
// saves the object under that name; without one, it shows or plays it.
// PresignGet is for the workers, which use the internal address.
func (c *Client) PresignForBrowser(ctx context.Context, key, filename string, expires time.Duration) (string, error) {
	in := &s3.GetObjectInput{Bucket: &c.bucket, Key: &key}
	if filename != "" {
		in.ResponseContentDisposition = aws.String(`attachment; filename="` + filename + `"`)
	}
	req, err := c.browserPresign.PresignGetObject(ctx, in, s3.WithPresignExpires(expires))
	if err != nil {
		return "", fmt.Errorf("presign %s: %w", key, err)
	}
	return req.URL, nil
}

// PublicURL is the browser-facing URL of a public object (path-style).
func (c *Client) PublicURL(key string) string {
	return c.publicURL + "/" + c.bucket + "/" + key
}
