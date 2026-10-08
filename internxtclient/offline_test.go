package internxtclient_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/StarHack/go-internxt-drive/internxtclient"
)

const testMnemonic = "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

// testBucketID is hex-encoded, as real bucket IDs are (key derivation decodes it).
const testBucketID = "0123456789abcdef0123456789abcdef"

// newOfflineClient returns a client whose API URLs all point at the given
// test server URL. No network access is required.
func newOfflineClient(t *testing.T, baseURL string) *internxtclient.Client {
	t.Helper()

	c := internxtclient.NewWithDefaults()
	c.Config.APIURLs = map[internxtclient.APIType]string{
		internxtclient.APITypeDrive:  baseURL + "/drive",
		internxtclient.APITypeAuth:   baseURL + "/drive/auth",
		internxtclient.APITypeUsers:  baseURL + "/users",
		internxtclient.APITypeBucket: baseURL + "/network/buckets",
		internxtclient.APITypeBase:   baseURL,
	}
	c.UserData = &internxtclient.UserData{
		AccessData: &internxtclient.AccessResponse{
			User: &internxtclient.User{
				Bucket:     testBucketID,
				Mnemonic:   testMnemonic,
				UUID:       "user-uuid",
				BridgeUser: "bridgeuser",
				UserID:     "userid",
			},
			NewToken: "test-token",
		},
		BasicAuthHeader: "Basic dGVzdA==",
	}
	return c
}

func TestOfflineUpdateFileMetaPayload(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/drive/files/file-uuid/meta" {
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"file-uuid"}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)
	plain := "newname"
	typ := "txt"
	if _, err := c.Files.UpdateFileMeta(context.Background(), "file-uuid", &internxtclient.UpdateFileMetaRequest{PlainName: &plain, Type: &typ}); err != nil {
		t.Fatalf("UpdateFileMeta: %v", err)
	}
	want := `{"plainName":"newname","type":"txt"}`
	if string(gotBody) != want {
		t.Errorf("request body = %s, want %s", gotBody, want)
	}

	// Partial update: only the provided fields are sent.
	gotBody = nil
	if _, err := c.Files.UpdateFileMeta(context.Background(), "file-uuid", &internxtclient.UpdateFileMetaRequest{PlainName: &plain}); err != nil {
		t.Fatalf("UpdateFileMeta partial: %v", err)
	}
	want = `{"plainName":"newname"}`
	if string(gotBody) != want {
		t.Errorf("partial request body = %s, want %s", gotBody, want)
	}
}

func TestOfflineMoveFilePayload(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && r.URL.Path == "/drive/files/file-uuid" {
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"file-uuid"}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	if _, err := c.Files.MoveFile(context.Background(), "file-uuid", "dest-uuid"); err != nil {
		t.Fatalf("MoveFile: %v", err)
	}
	if string(gotBody) != `{"destinationFolder":"dest-uuid"}` {
		t.Errorf("move body = %s", gotBody)
	}

	empty := ""
	gotBody = nil
	_, err := c.Files.MoveFileWithRequest(context.Background(), "file-uuid", &internxtclient.MoveFileRequest{
		DestinationFolder: "dest-uuid",
		Name:              &empty,
	})
	if err != nil {
		t.Fatalf("MoveFileWithRequest: %v", err)
	}
	if string(gotBody) != `{"destinationFolder":"dest-uuid","name":""}` {
		t.Errorf("move-with-rename body = %s", gotBody)
	}
}

func TestOfflineGetFilesQuery(t *testing.T) {
	var gotRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/drive/files" {
			gotRequest = r.URL.String()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"uuid":"file-uuid","status":"EXISTS"}]`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	files, err := c.Files.GetFiles(context.Background(), internxtclient.GetFilesOptions{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("GetFiles: %v", err)
	}
	if len(files) != 1 || files[0].UUID != "file-uuid" {
		t.Fatalf("GetFiles decoded = %+v", files)
	}
	if want := "/drive/files?limit=50&offset=0"; gotRequest != want {
		t.Errorf("GetFiles request = %s, want %s", gotRequest, want)
	}

	_, err = c.Files.GetFiles(context.Background(), internxtclient.GetFilesOptions{
		Limit:     25,
		Offset:    100,
		Status:    "TRASHED",
		Sort:      "updatedAt",
		Order:     "DESC",
		UpdatedAt: "2026-01-01T00:00:00.000Z",
	})
	if err != nil {
		t.Fatalf("GetFiles filtered: %v", err)
	}
	want := "/drive/files?limit=25&offset=100&order=DESC&sort=updatedAt&status=TRASHED&updatedAt=2026-01-01T00:00:00.000Z"
	if gotRequest != want {
		t.Errorf("GetFiles filtered request = %s, want %s", gotRequest, want)
	}
}

func TestOfflineGetFileCount(t *testing.T) {
	var gotRequest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/drive/files/count" {
			gotRequest = r.URL.String()
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"count":7}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	count, err := c.Files.GetFileCount(context.Background())
	if err != nil {
		t.Fatalf("GetFileCount: %v", err)
	}
	if count != 7 {
		t.Errorf("GetFileCount = %d, want 7", count)
	}
	if want := "/drive/files/count"; gotRequest != want {
		t.Errorf("GetFileCount request = %s, want %s (no query params)", gotRequest, want)
	}
}

func TestOfflineGetFileMetaByPathEncoding(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/drive/files/meta" {
			gotPath = r.URL.Query().Get("path")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"file-uuid"}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	wantPath := "/my folder/файл 1.txt"
	file, err := c.Files.GetFileMetaByPath(context.Background(), wantPath)
	if err != nil {
		t.Fatalf("GetFileMetaByPath: %v", err)
	}
	if file.UUID != "file-uuid" {
		t.Errorf("decoded file = %+v", file)
	}
	if gotPath != wantPath {
		t.Errorf("path query param = %q, want %q", gotPath, wantPath)
	}
}

func TestOfflineReplaceFilePayload(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPut && r.URL.Path == "/drive/files/file-uuid" {
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"file-uuid","fileId":"new-file-id","size":"123"}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	replaced, err := c.Files.ReplaceFile(context.Background(), "file-uuid", &internxtclient.ReplaceFileRequest{FileID: "new-file-id", Size: 123})
	if err != nil {
		t.Fatalf("ReplaceFile: %v", err)
	}
	if replaced.UUID != "file-uuid" || replaced.FileID != "new-file-id" || replaced.Size.String() != "123" {
		t.Errorf("ReplaceFile decoded = %+v", replaced)
	}
	if want := `{"fileId":"new-file-id","size":123}`; string(gotBody) != want {
		t.Errorf("request body = %s, want %s", gotBody, want)
	}
}

func TestOfflineCreateThumbnailPayload(t *testing.T) {
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/drive/files/thumbnail" {
			gotBody, _ = io.ReadAll(r.Body)
			w.Header().Set("Content-Type", "application/json")
			// Verbatim live response captured 2026-10-08.
			_, _ = w.Write([]byte(`{"id":440764818,"fileId":1752890934,"fileUuid":"01a11c3c-0ed3-74fe-b651-e6068919fd2a","type":"png","size":"100","bucketId":"6ac661bc7dab71b3654a47f5","bucketFile":"6ac7bd733f8817d45492fe4a","encryptVersion":"03-aes","createdAt":"2026-10-08T15:57:42.745Z","updatedAt":"2026-10-08T15:57:42.746Z","maxWidth":20,"maxHeight":20}`))
			return
		}
		http.Error(w, "unexpected request "+r.URL.String(), http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)

	thumb, err := c.Files.CreateThumbnail(context.Background(), &internxtclient.CreateThumbnailRequest{
		FileUUID:       "01a11c3c-0ed3-74fe-b651-e6068919fd2a",
		Type:           "png",
		Size:           100,
		MaxWidth:       20,
		MaxHeight:      20,
		BucketID:       "6ac661bc7dab71b3654a47f5",
		BucketFile:     "6ac7bd733f8817d45492fe4a",
		EncryptVersion: "03-aes",
	})
	if err != nil {
		t.Fatalf("CreateThumbnail: %v", err)
	}
	if want := `{"fileUuid":"01a11c3c-0ed3-74fe-b651-e6068919fd2a","type":"png","size":100,"maxWidth":20,"maxHeight":20,"bucketId":"6ac661bc7dab71b3654a47f5","bucketFile":"6ac7bd733f8817d45492fe4a","encryptVersion":"03-aes"}`; string(gotBody) != want {
		t.Errorf("request body = %s, want %s", gotBody, want)
	}
	if thumb.ID != 440764818 || thumb.FileID.String() != "1752890934" || thumb.Size.String() != "100" {
		t.Errorf("decoded thumbnail = %+v", thumb)
	}
	if thumb.MaxWidth != 20 || thumb.MaxHeight != 20 || thumb.EncryptVersion != "03-aes" {
		t.Errorf("thumbnail fields mismatch: %+v", thumb)
	}
}

func TestOfflineInvalidUploadSize(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)
	_, err := c.Buckets.UploadFileStream(context.Background(), "folder-uuid", "f", bytes.NewReader(nil), 0, time.Now())
	if !errors.Is(err, internxtclient.ErrInvalidUploadSize) {
		t.Fatalf("got %v, want ErrInvalidUploadSize", err)
	}
}

// mockBridge stores the uploaded ciphertext and serves it back through the
// bucket info endpoint, allowing a full offline encrypt→upload→download→
// decrypt roundtrip through the client's own crypto.
type mockBridge struct {
	srv        *httptest.Server
	ciphertext []byte
	index      string
	shardHash  string
	fileID     string
	startQuery string

	infoVersion  int
	extraShard   bool
	hashOverride string
}

func newMockBridge(t *testing.T) *mockBridge {
	m := &mockBridge{infoVersion: 2, fileID: "net-file-1"}
	mux := http.NewServeMux()

	mux.HandleFunc("/network/v2/buckets/"+testBucketID+"/files/start", func(w http.ResponseWriter, r *http.Request) {
		m.startQuery = r.URL.Query().Get("multiparts")
		writeJSON(w, map[string]any{"uploads": []map[string]any{{"index": 0, "uuid": "shard-uuid-1", "url": m.srv.URL + "/shard"}}})
	})
	mux.HandleFunc("/shard", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPut:
			body, _ := io.ReadAll(r.Body)
			m.ciphertext = body
			w.WriteHeader(http.StatusOK)
		case http.MethodGet:
			data := m.ciphertext
			if rng := r.Header.Get("Range"); rng != "" {
				start, end, err := parseTestRange(rng, len(data))
				if err != nil {
					http.Error(w, err.Error(), http.StatusRequestedRangeNotSatisfiable)
					return
				}
				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(data[start : end+1])
				return
			}
			_, _ = w.Write(data)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.HandleFunc("/network/v2/buckets/"+testBucketID+"/files/finish", func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Index  string `json:"index"`
			Shards []struct {
				Hash string `json:"hash"`
				UUID string `json:"uuid"`
			} `json:"shards"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		m.index = payload.Index
		m.shardHash = payload.Shards[0].Hash
		writeJSON(w, map[string]any{"id": m.fileID, "bucket": testBucketID, "index": payload.Index})
	})
	mux.HandleFunc("/drive/files", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"uuid": "drive-uuid-1", "fileId": m.fileID})
	})
	mux.HandleFunc("/network/buckets/"+testBucketID+"/files/net-file-1/info", func(w http.ResponseWriter, r *http.Request) {
		hash := m.shardHash
		if m.hashOverride != "" {
			hash = m.hashOverride
		}
		shards := []map[string]any{{"index": 0, "hash": hash, "url": m.srv.URL + "/shard"}}
		if m.extraShard {
			shards = append(shards, map[string]any{"index": 1, "hash": hash, "url": m.srv.URL + "/shard"})
		}
		writeJSON(w, map[string]any{
			"bucket":  testBucketID,
			"index":   m.index,
			"size":    len(m.ciphertext),
			"version": m.infoVersion,
			"shards":  shards,
		})
	})

	m.srv = httptest.NewServer(mux)
	t.Cleanup(m.srv.Close)
	return m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func parseTestRange(rng string, max int) (int, int, error) {
	if !strings.HasPrefix(rng, "bytes=") {
		return 0, 0, errors.New("bad range")
	}
	parts := strings.Split(strings.TrimPrefix(rng, "bytes="), "-")
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	end := max - 1
	if parts[1] != "" {
		end, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, 0, err
		}
	}
	if end >= max {
		end = max - 1
	}
	return start, end, nil
}

func uploadTestFile(t *testing.T, c *internxtclient.Client, data []byte) {
	t.Helper()
	meta, err := c.Buckets.UploadFileStream(context.Background(), "folder-uuid", "testfile", bytes.NewReader(data), int64(len(data)), time.Now())
	if err != nil {
		t.Fatalf("UploadFileStream: %v", err)
	}
	if meta.UUID != "drive-uuid-1" {
		t.Fatalf("unexpected meta uuid %q", meta.UUID)
	}
}

func TestOfflineUploadDownloadRoundtrip(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	plain := make([]byte, 100)
	if _, err := rand.Read(plain); err != nil {
		t.Fatal(err)
	}
	uploadTestFile(t, c, plain)

	if m.startQuery != "1" {
		t.Errorf("start multiparts query = %q, want 1", m.startQuery)
	}
	// Ciphertext must not equal plaintext (encryption happened on the fly).
	if bytes.Equal(m.ciphertext, plain) {
		t.Fatal("shard stored unencrypted")
	}
	// Stored hash matches the stored bytes.
	sum := sha1.Sum(m.ciphertext)
	if m.shardHash != hex.EncodeToString(sum[:]) {
		t.Fatalf("finish hash %s != sha1 of stored ciphertext", m.shardHash)
	}

	rc, err := c.Buckets.DownloadFileStream(context.Background(), m.fileID)
	if err != nil {
		t.Fatalf("DownloadFileStream: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read all: %v", err)
	}
	if !bytes.Equal(data, plain) {
		t.Fatalf("roundtrip mismatch: got %x, want %x", data, plain)
	}
}

func TestOfflineDownloadVerified(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	plain := make([]byte, 64)
	if _, err := rand.Read(plain); err != nil {
		t.Fatal(err)
	}
	uploadTestFile(t, c, plain)

	rc, err := c.Buckets.DownloadFileStreamVerified(context.Background(), m.fileID)
	if err != nil {
		t.Fatalf("verified download: %v", err)
	}
	data, err := io.ReadAll(rc)
	rc.Close()
	if err != nil || !bytes.Equal(data, plain) {
		t.Fatalf("verified roundtrip failed: %v", err)
	}

	// Corrupt the expected hash → mismatch surfaces on Read at EOF.
	m.hashOverride = strings.Repeat("0", 40)
	rc2, err := c.Buckets.DownloadFileStreamVerified(context.Background(), m.fileID)
	if err != nil {
		t.Fatalf("verified download (override): %v", err)
	}
	_, err = io.ReadAll(rc2)
	rc2.Close()
	if !errors.Is(err, internxtclient.ErrShardHashMismatch) {
		t.Fatalf("got %v, want ErrShardHashMismatch", err)
	}
}

func TestOfflineDownloadUnalignedRange(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	plain := make([]byte, 40)
	if _, err := rand.Read(plain); err != nil {
		t.Fatal(err)
	}
	uploadTestFile(t, c, plain)

	rc, err := c.Buckets.DownloadFileStream(context.Background(), m.fileID, "bytes=7-19")
	if err != nil {
		t.Fatalf("ranged download: %v", err)
	}
	defer rc.Close()
	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read ranged: %v", err)
	}
	if !bytes.Equal(data, plain[7:20]) {
		t.Fatalf("ranged data mismatch: got %x want %x", data, plain[7:20])
	}
}

func TestOfflineDownloadAlignedRange(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	plain := make([]byte, 40)
	if _, err := rand.Read(plain); err != nil {
		t.Fatal(err)
	}
	uploadTestFile(t, c, plain)

	rc, err := c.Buckets.DownloadFileStream(context.Background(), m.fileID, "bytes=16-31")
	if err != nil {
		t.Fatalf("aligned range: %v", err)
	}
	defer rc.Close()
	data, _ := io.ReadAll(rc)
	if !bytes.Equal(data, plain[16:32]) {
		t.Fatalf("aligned range mismatch")
	}
}

func TestOfflineDownloadVersionOneRejected(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	m.infoVersion = 1
	_, err := c.Buckets.DownloadFileStream(context.Background(), m.fileID)
	if !errors.Is(err, internxtclient.ErrFileVersionOne) {
		t.Fatalf("got %v, want ErrFileVersionOne", err)
	}
}

func TestOfflineDownloadMultiShardRejected(t *testing.T) {
	m := newMockBridge(t)
	c := newOfflineClient(t, m.srv.URL)

	plain := make([]byte, 32)
	if _, err := rand.Read(plain); err != nil {
		t.Fatal(err)
	}
	uploadTestFile(t, c, plain)

	m.extraShard = true
	_, err := c.Buckets.DownloadFileStream(context.Background(), m.fileID)
	if !errors.Is(err, internxtclient.ErrMultiShardUnsupported) {
		t.Fatalf("got %v, want ErrMultiShardUnsupported", err)
	}
}

func TestOfflineRetryOn429WithRetryAfter(t *testing.T) {
	var attempts int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			w.Header().Set("Retry-After", "1")
			http.Error(w, `{"retryable":true,"retry_after":1}`, http.StatusTooManyRequests)
			return
		}
		writeJSON(w, map[string]any{"uuid": "file-uuid"})
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)
	file, err := c.Files.GetFileMeta(context.Background(), "file-uuid")
	if err != nil {
		t.Fatalf("GetFileMeta after retry: %v", err)
	}
	if file.UUID != "file-uuid" {
		t.Fatalf("unexpected file %+v", file)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestOfflineAPIErrorHelpers(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"missing"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)
	_, err := c.Files.GetFileMeta(context.Background(), "nope")
	if !internxtclient.IsNotFound(err) {
		t.Fatalf("IsNotFound = false for %v", err)
	}
	apiErr, ok := internxtclient.AsAPIError(err)
	if !ok || apiErr.StatusCode != http.StatusNotFound {
		t.Fatalf("AsAPIError failed for %v", err)
	}
	if internxtclient.IsRateLimited(err) || internxtclient.IsPaymentRequired(err) || internxtclient.IsServerError(err) {
		t.Fatal("wrong status classification")
	}
}

func TestOfflineGetUserCredentialsUnmarshal(t *testing.T) {
	// Wire shape observed from the live API (matches SDK UserResponseDto):
	// mnemonic is a plain hex string; root_folder_id (legacy number) and
	// rootFolderId (root folder UUID) are distinct fields.
	const mnemonicHex = "53616c7465645f5fa54605e4910c043b"
	payload := `{"user":{"email":"user@example.com","userId":"$2a$08$abcdefghijklmnopqrstuv",` +
		`"mnemonic":"` + mnemonicHex + `","root_folder_id":42,` +
		`"rootFolderId":"01a11389-c83d-7578-aed5-422bd298f99f","uuid":"user-uuid"},` +
		`"oldToken":"old","newToken":"new"}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, payload)
	}))
	defer srv.Close()

	c := newOfflineClient(t, srv.URL)
	creds, err := c.Users.GetUserCredentials(context.Background())
	if err != nil {
		t.Fatalf("GetUserCredentials: %v", err)
	}
	if creds.User.Mnemonic != mnemonicHex {
		t.Fatalf("Mnemonic = %q, want %q", creds.User.Mnemonic, mnemonicHex)
	}
	if creds.User.RootFolderID != 42 {
		t.Fatalf("RootFolderID = %d, want 42", creds.User.RootFolderID)
	}
	if creds.User.RootFolderUUID != "01a11389-c83d-7578-aed5-422bd298f99f" {
		t.Fatalf("RootFolderUUID = %q", creds.User.RootFolderUUID)
	}
	if creds.NewToken != "new" {
		t.Fatalf("NewToken = %q, want %q", creds.NewToken, "new")
	}
}

func TestOfflineCanceledContext(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("request must not be made after cancellation")
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	c := newOfflineClient(t, srv.URL)
	_, err := c.Files.GetFileMeta(ctx, "file-uuid")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}
