package internxtclient

import (
	"encoding/json"
	"net/http"
	"path"
	"strconv"
	"time"
)

type FilesService struct {
	client *Client
}

// File represents the response object for files in a folder
// under GET /drive/folders/content/{uuid}/files
type File struct {
	ID               int64       `json:"id"`
	FileID           string      `json:"fileId"`
	UUID             string      `json:"uuid"`
	Name             string      `json:"name"`
	PlainName        string      `json:"plainName"`
	Type             string      `json:"type"`
	FolderID         json.Number `json:"folderId"`
	FolderUUID       string      `json:"folderUuid"`
	Folder           any         `json:"folder"`
	Bucket           string      `json:"bucket"`
	UserID           json.Number `json:"userId"`
	User             any         `json:"user"`
	EncryptVersion   string      `json:"encryptVersion"`
	Size             json.Number `json:"size"`
	Deleted          bool        `json:"deleted"`
	DeletedAt        *time.Time  `json:"deletedAt"`
	Removed          bool        `json:"removed"`
	RemovedAt        *time.Time  `json:"removedAt"`
	Shares           []any       `json:"shares"`
	Sharings         []any       `json:"sharings"`
	Thumbnails       []any       `json:"thumbnails"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
	CreationTime     time.Time   `json:"creationTime"`
	ModificationTime time.Time   `json:"modificationTime"`
	Status           string      `json:"status"`
}

const filesPath = "/files"

// GetFileMeta gets file with metadata by UUID
func (f *FilesService) GetFileMeta(fileUUID string) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID, "meta")

	var file File
	if resp, err := f.client.Get(APITypeDrive, endpoint, &file, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &file, nil
}

// DeleteFile deletes a file by UUID
func (f *FilesService) DeleteFile(uuid string) error {
	endpoint := path.Join(filesPath, uuid)

	if resp, err := f.client.Delete(APITypeDrive, endpoint, nil, nil, nil); err != nil {
		return f.client.GetError(endpoint, resp, err)
	}

	return nil
}

// UpdateFileMetaRequest is the payload for PUT /files/{uuid}/meta.
// It mirrors the API's UpdateFileMetaDto so only the provided fields are sent.
type UpdateFileMetaRequest struct {
	PlainName *string `json:"plainName,omitempty"`
	Type      *string `json:"type,omitempty"`
}

// UpdateFileMeta updates the metadata of a file with the given UUID.
func (f *FilesService) UpdateFileMeta(fileUUID string, updated *UpdateFileMetaRequest) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID, "meta")
	var updatedFile File

	if resp, err := f.client.Put(APITypeDrive, endpoint, updated, &updatedFile, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &updatedFile, nil
}

// MoveFileRequest is the payload for PATCH /files/{uuid}.
// Name and Type are optional: set them to rename the file or change its extension while moving;
// pass a pointer to an empty string to clear them, and nil to keep the current values.
type MoveFileRequest struct {
	DestinationFolder string  `json:"destinationFolder"`
	Name              *string `json:"name,omitempty"`
	Type              *string `json:"type,omitempty"`
}

// MoveFile moves the file with the given UUID to the destination folder.
func (f *FilesService) MoveFile(fileUUID, destinationFolderUUID string) (*File, error) {
	return f.MoveFileWithRequest(fileUUID, &MoveFileRequest{DestinationFolder: destinationFolderUUID})
}

// MoveFileWithRequest moves the file with the given UUID and applies the
// optional rename fields of the request in the same operation.
func (f *FilesService) MoveFileWithRequest(fileUUID string, req *MoveFileRequest) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID)
	var movedFile File

	if resp, err := f.client.Patch(APITypeDrive, endpoint, req, &movedFile, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &movedFile, nil
}

// GetRecentFiles retrieves a list of recent files with the given limit.
func (f *FilesService) GetRecentFiles(limit int) ([]File, error) {
	endpoint := path.Join(filesPath, "recents")

	var files []File

	if resp, err := f.client.doRequestWithQuery(APITypeDrive, http.MethodGet, endpoint, map[string]string{"limit": strconv.Itoa(limit)}, nil, &files, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return files, nil
}
