package internxtclient_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/StarHack/go-internxt-drive/internxtclient"
)

func TestFilesIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	//creating something we can test with
	fileMeta := createFile(t, "files_file", testFolderUUID)
	filesFolder := createFolder(t, "files_folder", testFolderUUID)

	var file *internxtclient.File

	time.Sleep(1 * time.Second)

	t.Run("GetFileMeta", func(t *testing.T) {
		file = getFileMeta(t, fileMeta.UUID)
	})

	t.Run("UpdateFileMeta", func(t *testing.T) {
		file = updateFileMefa(t, fileMeta.UUID, "newname")
	})

	time.Sleep(1 * time.Second)

	t.Run("MoveFile", func(t *testing.T) {
		file = moveFile(t, file.UUID, filesFolder.UUID)
	})

	t.Run("GetRecentFiles", func(t *testing.T) {
		recentFiles := getRecentFiles(t, 3)
		found := false
		for _, f := range recentFiles {
			if f.UUID == file.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("can't find file in recent files")
		}
	})

	t.Run("DeleteFile", func(t *testing.T) {
		deleteFile(t, file.UUID)
		time.Sleep(1 * time.Second)
		deletedFile := getFileMeta(t, file.UUID)
		if deletedFile.Status != "DELETED" {
			t.Fatalf("can't delete file")
		}
	})

	deleteFolder(t, filesFolder.UUID)
	time.Sleep(1 * time.Second)
}

func getRecentFiles(t *testing.T, limit int) []internxtclient.File {
	files, err := c.Files.GetRecentFiles(testCtx, limit)
	if err != nil {
		t.Fatalf("can't get recent files: %v", err)
	}
	if files == nil {
		t.Fatal("files is nil")
	}
	return files
}

func deleteFile(t *testing.T, uuid string) {
	err := c.Files.DeleteFile(testCtx, uuid)
	if err != nil {
		t.Fatalf("can't delete file %s: %v", uuid, err)
	}
}

func moveFile(t *testing.T, uuid string, targetFolderUUID string) *internxtclient.File {
	movedFile, err := c.Files.MoveFile(testCtx, uuid, targetFolderUUID)
	if err != nil {
		t.Fatalf("can't move file: %v", err)
	}

	if movedFile == nil {
		t.Fatal("movedFile is nil")
	}
	if movedFile.UUID != uuid {
		t.Errorf("expected file UUID %s, got %s", uuid, movedFile.UUID)
	}
	if movedFile.FolderUUID != targetFolderUUID {
		t.Errorf("expected file parent %s, got %s", targetFolderUUID, movedFile.FolderID)
	}

	return movedFile
}

func updateFileMefa(t *testing.T, uuid string, newName string) *internxtclient.File {
	newType := newName + "ext"
	newValues := internxtclient.UpdateFileMetaRequest{PlainName: &newName, Type: &newType}
	updatedFile, err := c.Files.UpdateFileMeta(testCtx, uuid, &newValues)
	if err != nil {
		t.Fatalf("can't update file meta: %v", err)
	}

	if updatedFile == nil {
		t.Fatal("updated file metadata is nil")
	}
	if updatedFile.UUID != uuid {
		t.Errorf("expected file UUID %s, got %s", uuid, updatedFile.UUID)
	}
	if updatedFile.PlainName != newName {
		t.Errorf("expected file name %s, got %s", newName, updatedFile.Name)
	}
	if updatedFile.Type != newName+"ext" {
		t.Errorf("expected file Type %s, got %s", newName+"ext", updatedFile.Type)
	}

	return updatedFile
}

func getFileMeta(t *testing.T, uuid string) *internxtclient.File {
	file, err := c.Files.GetFileMeta(testCtx, uuid)
	if err != nil {
		t.Fatalf("can't get file meta: %v", err)
	}

	// Add assertions for the retrieved file metadata
	if file == nil {
		t.Fatal("retrieved file metadata is nil")
	}
	if file.UUID != uuid {
		t.Errorf("expected file UUID %s, got %s", uuid, file.UUID)
	}
	if file.Name == "" {
		t.Error("file name is empty")
	}
	return file
}

func TestFileListingEndpointsIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	fileMeta := createFile(t, "listing_file", testFolderUUID)

	time.Sleep(1 * time.Second)

	t.Run("GetFiles", func(t *testing.T) {
		files := getFiles(t, internxtclient.GetFilesOptions{Limit: 100, Offset: 0, Status: "EXISTS"})
		found := false
		for _, f := range files {
			if f.UUID == fileMeta.UUID {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("can't find uploaded file in GET /files listing")
		}
	})

	t.Run("GetFileCount", func(t *testing.T) {
		count := getFileCount(t)
		if count < 1 {
			t.Fatalf("file count = %d, want >= 1", count)
		}
	})

	t.Run("GetFileMetaByPath", func(t *testing.T) {
		byPath := getFileMetaByPath(t, "/"+TESTFOLDER+"/listing_file")
		if byPath.UUID != fileMeta.UUID {
			t.Fatalf("meta by path returned uuid %s, want %s", byPath.UUID, fileMeta.UUID)
		}
	})

	deleteFile(t, fileMeta.UUID)
}

func TestFileReplaceIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	target := createFile(t, "replace_target", testFolderUUID)
	source := createFile(t, "replace_source", testFolderUUID)

	time.Sleep(1 * time.Second)

	t.Run("ReplaceFile", func(t *testing.T) {
		req := internxtclient.ReplaceFileRequest{FileID: source.FileID, Size: int64(len(testBytes))}
		replaced := replaceFile(t, target.UUID, req)
		if replaced.FileID != source.FileID {
			t.Fatalf("replaced file has fileId %s, want %s", replaced.FileID, source.FileID)
		}

		// the replacement must be persisted, not just echoed
		byPath := getFileMetaByPath(t, "/"+TESTFOLDER+"/replace_target")
		if byPath.FileID != source.FileID {
			t.Errorf("after replace, stored fileId is %s, want %s", byPath.FileID, source.FileID)
		}
	})

	deleteFile(t, target.UUID)
	deleteFile(t, source.UUID)
}

func TestThumbnailIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	fileMeta := createFile(t, "thumbnail_file", testFolderUUID)

	time.Sleep(1 * time.Second)

	t.Run("CreateThumbnail", func(t *testing.T) {
		thumb := createThumbnail(t, fileMeta)
		if thumb.FileUUID != fileMeta.UUID {
			t.Errorf("thumbnail fileUuid = %s, want %s", thumb.FileUUID, fileMeta.UUID)
		}
		if thumb.ID == 0 {
			t.Errorf("thumbnail id is 0")
		}
		if thumb.BucketFile != fileMeta.FileID {
			t.Errorf("thumbnail bucketFile = %s, want %s", thumb.BucketFile, fileMeta.FileID)
		}
	})

	deleteFile(t, fileMeta.UUID)
}

func TestCreateFileEntryIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	source := createFile(t, "entry_source", testFolderUUID)

	time.Sleep(1 * time.Second)

	// A second entry for the same stored content (dedup) is the direct-use case of CreateFileEntry.
	// Empty files are rejected client-side, so they are covered by the offline validation test.
	t.Run("CreateFileEntry registers entry for stored content", func(t *testing.T) {
		entry := createFileEntry(t, "entry_dup", source.FileID, int64(len(testBytes)))

		dup := waitForFileMeta(t, entry.UUID)
		if dup.FileID != source.FileID {
			t.Errorf("dup entry fileId = %s, want %s", dup.FileID, source.FileID)
		}
		if dup.Status != "EXISTS" {
			t.Errorf("dup entry status = %s, want EXISTS", dup.Status)
		}
		deleteFile(t, entry.UUID)
	})

	deleteFile(t, source.UUID)
}

func TestGetFilesAccountScopeIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	rootUUID := c.UserData.AccessData.User.RootFolderUUID
	rootFile := createFile(t, "scope_root_file", rootUUID)
	folderFile := createFile(t, "scope_folder_file", testFolderUUID)
	nested := createFolder(t, "scope_nested", testFolderUUID)
	nestedFile := createFile(t, "scope_nested_file", nested.UUID)

	t.Logf("manual UI check for 10s: /scope_root_file | /%s/scope_folder_file | /%s/scope_nested/scope_nested_file", TESTFOLDER, TESTFOLDER)
	time.Sleep(10 * time.Second)

	opts := internxtclient.GetFilesOptions{Limit: 50, Offset: 0, Status: "EXISTS", Sort: "updatedAt", Order: "DESC"}

	t.Run("AccountScope", func(t *testing.T) {
		files := waitForFilesContaining(t, opts, rootFile.UUID, folderFile.UUID, nestedFile.UUID)
		for _, want := range []string{rootFile.UUID, folderFile.UUID, nestedFile.UUID} {
			if !containsFileUUID(files, want) {
				t.Errorf("account listing missing %s", want)
			}
		}
	})

	t.Run("FolderMembership", func(t *testing.T) {
		files := listFolderFiles(t, testFolderUUID)
		if !containsFileUUID(files, folderFile.UUID) {
			t.Errorf("folder listing missing its own file %s", folderFile.UUID)
		}
		if containsFileUUID(files, rootFile.UUID) {
			t.Errorf("folder listing must not contain root file %s", rootFile.UUID)
		}
		if containsFileUUID(files, nestedFile.UUID) {
			t.Errorf("folder listing must not contain nested file %s", nestedFile.UUID)
		}
	})

	t.Run("RootAsFolder", func(t *testing.T) {
		files := listFolderFiles(t, rootUUID)
		if !containsFileUUID(files, rootFile.UUID) {
			t.Errorf("root listing missing root file %s", rootFile.UUID)
		}
		if containsFileUUID(files, folderFile.UUID) || containsFileUUID(files, nestedFile.UUID) {
			t.Errorf("root listing must not contain nested files")
		}
	})

	t.Run("StatusFilterExcludesDeleted", func(t *testing.T) {
		deleteFile(t, folderFile.UUID)
		files := waitForFilesAbsent(t, opts, folderFile.UUID)
		if containsFileUUID(files, folderFile.UUID) {
			t.Errorf("EXISTS listing still contains deleted file %s", folderFile.UUID)
		}
	})

	deleteFile(t, rootFile.UUID)
	deleteFile(t, nestedFile.UUID)
	deleteFolder(t, nested.UUID)
}

func containsFileUUID(files []internxtclient.File, uuid string) bool {
	for _, f := range files {
		if f.UUID == uuid {
			return true
		}
	}
	return false
}

func waitForFilesContaining(t *testing.T, opts internxtclient.GetFilesOptions, uuids ...string) []internxtclient.File {
	deadline := time.Now().Add(10 * time.Second)
	var files []internxtclient.File
	for {
		files = getFiles(t, opts)
		all := true
		for _, u := range uuids {
			if !containsFileUUID(files, u) {
				all = false
				break
			}
		}
		if all || time.Now().After(deadline) {
			return files
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func waitForFilesAbsent(t *testing.T, opts internxtclient.GetFilesOptions, uuid string) []internxtclient.File {
	deadline := time.Now().Add(10 * time.Second)
	var files []internxtclient.File
	for {
		files = getFiles(t, opts)
		if !containsFileUUID(files, uuid) || time.Now().After(deadline) {
			return files
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func listFolderFiles(t *testing.T, folderUUID string) []internxtclient.File {
	files, err := c.Folders.ListFiles(testCtx, folderUUID, &internxtclient.ListOptions{Limit: 50, Offset: 0})
	if err != nil {
		t.Fatalf("can't list files of folder %s: %v", folderUUID, err)
	}
	return files
}

func TestMoveFileWithRequestIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	file := createFile(t, "mvrename_src", testFolderUUID)
	dest := createFolder(t, "mvrename_dest", testFolderUUID)

	newName := "mvrenamed"
	newType := "dat"

	t.Run("MoveAndRename", func(t *testing.T) {
		// A freshly uploaded file needs a moment to materialize everywhere it is looked up: moves briefly
		// answer 404 (entry not visible yet) or 422 "can not be moved" (propagation window), observed 2026-10-08.
		waitForFileMeta(t, file.UUID)
		deadline := time.Now().Add(10 * time.Second)
		var moved *internxtclient.File
		for {
			var err error
			moved, err = c.Files.MoveFileWithRequest(testCtx, file.UUID, &internxtclient.MoveFileRequest{
				DestinationFolder: dest.UUID,
				Name:              &newName,
				Type:              &newType,
			})
			if err == nil {
				break
			}
			apiErr, isAPI := internxtclient.AsAPIError(err)
			maturing := isAPI && (apiErr.StatusCode == http.StatusUnprocessableEntity || apiErr.StatusCode == http.StatusNotFound)
			if !maturing || time.Now().After(deadline) {
				t.Fatalf("can't move file with rename: %v", err)
			}
			time.Sleep(1 * time.Second)
		}
		if moved == nil || moved.UUID != file.UUID {
			t.Fatalf("move response is nil or wrong file: %+v", moved)
		}
		if moved.FolderUUID != dest.UUID {
			t.Errorf("moved file folderUuid = %s, want %s", moved.FolderUUID, dest.UUID)
		}
		if moved.Type != newType {
			t.Errorf("moved file type = %s, want %s", moved.Type, newType)
		}

		// The meta read path can lag the mutation and return the pre-move state (observed 2026-10-08),
		// so poll until the change is visible.
		reread := waitForFileState(t, file.UUID, func(f *internxtclient.File) bool {
			return f.FolderUUID == dest.UUID && f.Type == newType && f.PlainName == newName
		})
		// Live behavior (observed 2026-10-08): the server sets both name and plainName from the plaintext MoveFileRequest.Name.
		if reread.PlainName != newName {
			t.Errorf("persisted plainName = %s, want %s", reread.PlainName, newName)
		}
	})

	// Cleanup is tolerant: a rejected move can leave the entry in a state where deletion 404s (observed 2026-10-08);
	// the TESTFOLDER teardown purges the rest.
	if err := c.Files.DeleteFile(testCtx, file.UUID); err != nil {
		t.Logf("cleanup: delete moved file: %v", err)
	}
	deleteFolder(t, dest.UUID)
}

// waitForFileMeta polls GetFileMeta until the entry is readable. Freshly created entries can 404 for a few hundred ms
// after creation (read-after-write lag, observed 2026-10-08), like the logout revocation lag in the auth test.
func waitForFileMeta(t *testing.T, uuid string) *internxtclient.File {
	deadline := time.Now().Add(6 * time.Second)
	var file *internxtclient.File
	var err error
	for {
		file, err = c.Files.GetFileMeta(testCtx, uuid)
		if err == nil || time.Now().After(deadline) {
			break
		}
		time.Sleep(300 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("file meta %s never became readable: %v", uuid, err)
	}
	if file.UUID != uuid {
		t.Errorf("expected file UUID %s, got %s", uuid, file.UUID)
	}
	return file
}

// waitForFileState polls GetFileMeta until the predicate holds. Update responses
// are authoritative but the read path can serve the pre-update state for a short
// while (same eventual-consistency class as waitForFileMeta, observed 2026-10-08).
func waitForFileState(t *testing.T, uuid string, pred func(*internxtclient.File) bool) *internxtclient.File {
	deadline := time.Now().Add(8 * time.Second)
	var file *internxtclient.File
	for {
		file = getFileMeta(t, uuid)
		if pred(file) || time.Now().After(deadline) {
			return file
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func getFiles(t *testing.T, opts internxtclient.GetFilesOptions) []internxtclient.File {
	files, err := c.Files.GetFiles(testCtx, opts)
	if err != nil {
		t.Fatalf("can't get files: %v", err)
	}
	return files
}

func getFileCount(t *testing.T) int64 {
	count, err := c.Files.GetFileCount(testCtx)
	if err != nil {
		t.Fatalf("can't get file count: %v", err)
	}
	return count
}

func getFileMetaByPath(t *testing.T, filePath string) *internxtclient.File {
	file, err := c.Files.GetFileMetaByPath(testCtx, filePath)
	if err != nil {
		t.Fatalf("can't get file meta by path %s: %v", filePath, err)
	}
	if file == nil {
		t.Fatal("retrieved file metadata is nil")
	}
	return file
}

func replaceFile(t *testing.T, fileUUID string, req internxtclient.ReplaceFileRequest) *internxtclient.File {
	replaced, err := c.Files.ReplaceFile(testCtx, fileUUID, &req)
	if err != nil {
		t.Fatalf("can't replace file %s: %v", fileUUID, err)
	}
	if replaced == nil {
		t.Fatal("replaced file is nil")
	}
	if replaced.UUID != fileUUID {
		t.Errorf("expected replaced file UUID %s, got %s", fileUUID, replaced.UUID)
	}
	return replaced
}

func createThumbnail(t *testing.T, fileMeta *internxtclient.CreateMetaResponse) *internxtclient.Thumbnail {
	thumb, err := c.Files.CreateThumbnail(testCtx, &internxtclient.CreateThumbnailRequest{
		FileUUID:       fileMeta.UUID,
		Type:           "png",
		Size:           int64(len(testBytes)),
		MaxWidth:       20,
		MaxHeight:      20,
		BucketID:       fileMeta.Bucket,
		BucketFile:     fileMeta.FileID,
		EncryptVersion: fileMeta.EncryptVersion,
	})
	if err != nil {
		t.Fatalf("can't create thumbnail: %v", err)
	}
	if thumb == nil {
		t.Fatal("thumbnail is nil")
	}
	if thumb.ID == 0 {
		t.Error("thumbnail id is zero")
	}
	return thumb
}

func createFileEntry(t *testing.T, plainName, fileID string, size int64) *internxtclient.CreateMetaResponse {
	entry, err := c.Files.CreateFileEntry(testCtx, &internxtclient.CreateMetaRequest{
		Name:           plainName,
		PlainName:      plainName,
		Type:           "txt",
		EncryptVersion: "03-aes",
		FolderUuid:     testFolderUUID,
		FileID:         fileID,
		Size:           size,
	})
	if err != nil {
		t.Fatalf("can't create file entry: %v", err)
	}
	if entry == nil || entry.UUID == "" {
		t.Fatal("created file entry is nil or has no uuid")
	}
	if entry.Bucket != c.UserData.AccessData.User.Bucket {
		t.Errorf("created entry bucket = %s, want %s", entry.Bucket, c.UserData.AccessData.User.Bucket)
	}
	return entry
}
