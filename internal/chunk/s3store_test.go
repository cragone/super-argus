package chunk

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/uuid"
)

// TestS3Store runs the shared Store contract against a real S3-compatible
// endpoint (e.g. local MinIO via docker-compose.yml / CI's minio service).
// Skipped unless ARGUS_TEST_S3_ENDPOINT is set:
//
//	docker compose up -d minio
//	ARGUS_TEST_S3_ENDPOINT=http://localhost:9000 ARGUS_TEST_S3_BUCKET=argus-test \
//	  AWS_ACCESS_KEY_ID=argus-test AWS_SECRET_ACCESS_KEY=argus-test-secret \
//	  go test ./internal/chunk -run TestS3Store -v
func TestS3Store(t *testing.T) {
	endpoint := os.Getenv("ARGUS_TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("set ARGUS_TEST_S3_ENDPOINT to run")
	}
	bucket := os.Getenv("ARGUS_TEST_S3_BUCKET")
	if bucket == "" {
		bucket = "argus-test"
	}
	// Unique prefix per run so repeated runs against a shared bucket don't
	// collide with leftover objects from a previous run.
	prefix := fmt.Sprintf("contract-test/%s", uuid.NewString())
	testStoreContract(t, func() Store {
		s, err := NewS3Store(context.Background(), S3Config{
			Bucket:   bucket,
			Endpoint: endpoint,
			Prefix:   prefix,
		})
		if err != nil {
			t.Fatalf("NewS3Store: %v", err)
		}
		return s
	})
}
