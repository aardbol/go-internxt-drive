package internxtclient

import (
	"context"
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
func (f *FilesService) GetFileMeta(ctx context.Context, fileUUID string) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID, "meta")

	var file File
	if resp, err := f.client.Get(ctx, APITypeDrive, endpoint, &file, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &file, nil
}

// DeleteFile deletes a file by UUID
func (f *FilesService) DeleteFile(ctx context.Context, uuid string) error {
	endpoint := path.Join(filesPath, uuid)

	if resp, err := f.client.Delete(ctx, APITypeDrive, endpoint, nil, nil, nil); err != nil {
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
func (f *FilesService) UpdateFileMeta(ctx context.Context, fileUUID string, updated *UpdateFileMetaRequest) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID, "meta")
	var updatedFile File

	if resp, err := f.client.Put(ctx, APITypeDrive, endpoint, updated, &updatedFile, nil); err != nil {
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
func (f *FilesService) MoveFile(ctx context.Context, fileUUID, destinationFolderUUID string) (*File, error) {
	return f.MoveFileWithRequest(ctx, fileUUID, &MoveFileRequest{DestinationFolder: destinationFolderUUID})
}

// MoveFileWithRequest moves the file with the given UUID and applies the
// optional rename fields of the request in the same operation.
func (f *FilesService) MoveFileWithRequest(ctx context.Context, fileUUID string, req *MoveFileRequest) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID)
	var movedFile File

	if resp, err := f.client.Patch(ctx, APITypeDrive, endpoint, req, &movedFile, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &movedFile, nil
}

// GetRecentFiles retrieves a list of recent files with the given limit.
func (f *FilesService) GetRecentFiles(ctx context.Context, limit int) ([]File, error) {
	endpoint := path.Join(filesPath, "recents")

	var files []File

	if resp, err := f.client.doRequestWithQuery(ctx, APITypeDrive, http.MethodGet, endpoint, map[string]string{"limit": strconv.Itoa(limit)}, nil, &files, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return files, nil
}

// GetFilesOptions is the query set for GET /files (paginated listing).
// Limit and Offset are required by the API; the rest are optional filters.
type GetFilesOptions struct {
	Limit     int    `url:"limit"`
	Offset    int    `url:"offset"`
	Status    string `url:"status,omitempty"`    // EXISTS | TRASHED | DELETED | ALL
	Sort      string `url:"sort,omitempty"`      // updatedAt | uuid
	Order     string `url:"order,omitempty"`     // ASC | DESC
	UpdatedAt string `url:"updatedAt,omitempty"` // return files updated after this date
}

// GetFiles lists the account's files with pagination and filtering.
func (f *FilesService) GetFiles(ctx context.Context, opts GetFilesOptions) ([]File, error) {
	var files []File

	if resp, err := f.client.doRequestWithStruct(ctx, APITypeDrive, http.MethodGet, filesPath, opts, nil, &files, nil); err != nil {
		return nil, f.client.GetError(filesPath, resp, err)
	}

	return files, nil
}

// GetFileCount returns the total number of files in the account.
// The live API rejects the documented status filter on this endpoint with
// HTTP 400 for every value (probed 2026-10-08), so no query is sent.
func (f *FilesService) GetFileCount(ctx context.Context) (int64, error) {
	endpoint := path.Join(filesPath, "count")
	var result struct {
		Count int64 `json:"count"`
	}

	if resp, err := f.client.Get(ctx, APITypeDrive, endpoint, &result, nil); err != nil {
		return -1, f.client.GetError(endpoint, resp, err)
	}

	return result.Count, nil
}

// GetFileMetaByPath gets file metadata by its full decrypted path,
// e.g. "/folder/subfolder/file.txt". The path is URL-encoded as a query value.
func (f *FilesService) GetFileMetaByPath(ctx context.Context, filePath string) (*File, error) {
	endpoint := path.Join(filesPath, "meta")

	var file File

	if resp, err := f.client.doRequestWithQuery(ctx, APITypeDrive, http.MethodGet, endpoint, map[string]string{"path": filePath}, nil, &file, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &file, nil
}

// ReplaceFileRequest is the payload for PUT /files/{uuid}: the encrypted fileId
// and size of the newly uploaded content to point the file entry at.
type ReplaceFileRequest struct {
	FileID string `json:"fileId"`
	Size   int64  `json:"size"`
}

// ReplaceFile points the file entry with the given UUID at new content
// (PUT /files/{uuid}), replacing its fileId and size.
func (f *FilesService) ReplaceFile(ctx context.Context, fileUUID string, req *ReplaceFileRequest) (*File, error) {
	endpoint := path.Join(filesPath, fileUUID)
	var replacedFile File

	if resp, err := f.client.Put(ctx, APITypeDrive, endpoint, req, &replacedFile, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &replacedFile, nil
}

// CreateThumbnailRequest is the payload for POST /files/thumbnail, matching the API's CreateThumbnailDto.
// The deprecated numeric fileId field is omitted; the file is referenced by FileUUID.
type CreateThumbnailRequest struct {
	FileUUID       string `json:"fileUuid"`
	Type           string `json:"type"`
	Size           int64  `json:"size"`
	MaxWidth       int    `json:"maxWidth"`
	MaxHeight      int    `json:"maxHeight"`
	BucketID       string `json:"bucketId"`
	BucketFile     string `json:"bucketFile"`
	EncryptVersion string `json:"encryptVersion"`
}

// Thumbnail is a thumbnail entry as returned by POST /files/thumbnail
// (ThumbnailDto). FileID is the numeric id of the parent file record.
type Thumbnail struct {
	ID             int64       `json:"id"`
	FileID         json.Number `json:"fileId"`
	FileUUID       string      `json:"fileUuid"`
	Type           string      `json:"type"`
	Size           json.Number `json:"size"`
	MaxWidth       int         `json:"maxWidth"`
	MaxHeight      int         `json:"maxHeight"`
	BucketID       string      `json:"bucketId"`
	BucketFile     string      `json:"bucketFile"`
	EncryptVersion string      `json:"encryptVersion"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

// CreateThumbnail registers a thumbnail entry for the file with the given UUID.
// It only creates the metadata record; the thumbnail content itself must be uploaded separately under BucketFile.
func (f *FilesService) CreateThumbnail(ctx context.Context, req *CreateThumbnailRequest) (*Thumbnail, error) {
	endpoint := path.Join(filesPath, "thumbnail")
	var thumbnail Thumbnail

	if resp, err := f.client.Post(ctx, APITypeDrive, endpoint, req, &thumbnail, nil); err != nil {
		return nil, f.client.GetError(endpoint, resp, err)
	}

	return &thumbnail, nil
}
