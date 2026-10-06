package internxtclient_test

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/StarHack/go-internxt-drive/internxtclient"
)

func TestBucketsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	var bucketsFile *internxtclient.CreateMetaResponse

	t.Run("CreateFile", func(t *testing.T) {
		bucketsFile = createFile(t, "buckets_file", testFolderUUID)
	})

	time.Sleep(1 * time.Second)
	deleteFile(t, bucketsFile.UUID)
	time.Sleep(1 * time.Second)
}

func TestBucketsDownloadIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	content := []byte("go-internxt-drive integration roundtrip 0123456789 abcdefghijklmnop")

	meta, err := c.Buckets.UploadFileStream(testCtx, testFolderUUID, "roundtrip", bytes.NewReader(content), int64(len(content)), time.Now())
	if err != nil {
		t.Fatalf("couldn't upload roundtrip file: %v", err)
	}
	time.Sleep(1 * time.Second)
	defer func() {
		deleteFile(t, meta.UUID)
		time.Sleep(1 * time.Second)
	}()

	rc, err := c.Buckets.DownloadFileStream(testCtx, meta.FileID)
	if err != nil {
		t.Fatalf("couldn't open download stream: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("couldn't read download stream: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("roundtrip mismatch: got %d bytes, want %d bytes", len(got), len(content))
	}

	rcRange, err := c.Buckets.DownloadFileStream(testCtx, meta.FileID, "bytes=5-20")
	if err != nil {
		t.Fatalf("couldn't open ranged download: %v", err)
	}
	gotRange, err := io.ReadAll(rcRange)
	rcRange.Close()
	if err != nil {
		t.Fatalf("couldn't read ranged download: %v", err)
	}
	if !bytes.Equal(gotRange, content[5:21]) {
		t.Fatalf("ranged download mismatch: got %q, want %q", gotRange, content[5:21])
	}

	rcVerified, err := c.Buckets.DownloadFileStreamVerified(testCtx, meta.FileID)
	if err != nil {
		t.Fatalf("couldn't open verified download: %v", err)
	}
	gotVerified, err := io.ReadAll(rcVerified)
	if err != nil {
		t.Fatalf("verified download failed hash check: %v", err)
	}
	if err := rcVerified.Close(); err != nil {
		t.Fatalf("closing verified stream: %v", err)
	}
	if !bytes.Equal(gotVerified, content) {
		t.Fatalf("verified roundtrip mismatch")
	}
}

func createFile(t *testing.T, filename, destFolderUUID string) *internxtclient.CreateMetaResponse {
	createMetaResponse, err := c.Buckets.UploadFileStream(testCtx, destFolderUUID, filename, bytes.NewReader(testBytes), int64(len(testBytes)), time.Now())
	if err != nil {
		t.Fatalf("couldn't upload filestream: %v", err)
	}
	if createMetaResponse == nil {
		t.Fatalf("createMetaResponse is nil")
	}
	if createMetaResponse.Bucket != c.UserData.AccessData.User.Bucket {
		t.Fatalf("createMetaResponse.Bucket is not the same as user's bucket. User's bucket is %s, but got %s", createMetaResponse.Bucket, c.UserData.AccessData.User.Bucket)
	}

	return createMetaResponse
}
