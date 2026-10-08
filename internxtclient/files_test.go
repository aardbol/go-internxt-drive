package internxtclient_test

import (
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
