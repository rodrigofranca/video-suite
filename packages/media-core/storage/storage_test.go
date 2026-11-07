package storage

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testClient connects to the storage from `make up`, or skips the test.
func testClient(t *testing.T) *Client {
	t.Helper()
	cfg, err := ConfigFromEnv()
	if err != nil {
		cfg = Config{Endpoint: "http://localhost:9000", Region: "us-east-1", AccessKey: "ultramedia", SecretKey: "ultramedia", Bucket: "ultramedia", PublicURL: "http://localhost:9000"}
	}
	c := New(cfg)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := c.Check(ctx); err != nil {
		t.Skipf("storage not available: %v", err)
	}
	return c
}

func TestUploadDownloadDelete(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	key := "test/" + t.Name() + "/" + time.Now().Format("150405.000000")

	// 20MB of random data forces a multipart upload with 16MB parts.
	data := make([]byte, 20<<20)
	rand.Read(data)
	if err := c.Upload(ctx, key, io.NopCloser(bytes.NewReader(data)), "application/octet-stream"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Delete(context.Background(), key) })

	path := filepath.Join(t.TempDir(), "out")
	if err := c.DownloadFile(ctx, key, path); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(path)
	if !bytes.Equal(got, data) {
		t.Fatalf("downloaded %d bytes, want the %d uploaded", len(got), len(data))
	}

	if ok, err := c.Exists(ctx, key); !ok || err != nil {
		t.Fatalf("Exists = %v, %v", ok, err)
	}
	if err := c.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if ok, err := c.Exists(ctx, key); ok || err != nil {
		t.Fatalf("after delete: Exists = %v, %v", ok, err)
	}
}

func TestOnlyHLSIsPublic(t *testing.T) {
	c := testClient(t)
	ctx := context.Background()
	asset := "test-" + time.Now().Format("150405000000")
	hls, src := HLSManifestKey(asset), SourceKey(asset)
	for _, key := range []string{hls, src} {
		if err := c.Upload(ctx, key, bytes.NewReader([]byte("#EXTM3U\n")), "application/vnd.apple.mpegurl"); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { c.Delete(context.Background(), key) })
	}

	status := func(key string) int {
		resp, err := http.Get(c.PublicURL(key))
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	if got := status(hls); got != http.StatusOK {
		t.Errorf("anonymous GET of HLS playlist = %d, want 200", got)
	}
	if got := status(src); got != http.StatusForbidden {
		t.Errorf("anonymous GET of source = %d, want 403", got)
	}
}
