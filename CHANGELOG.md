# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased]

### Added

- `FilesService.MoveFileWithRequest` — move a file and optionally rename it / change its
  extension in the same operation (`MoveFileRequest`, matching the API's `MoveFileDto`).
- `BucketsService.DownloadFileStreamVerified` — whole-file download that verifies the
  encrypted shard's SHA-1 against the hash recorded in the file info while streaming
  (`ErrShardHashMismatch` on failure, including when the stream is closed early).
- Typed API errors: `*APIError` (status, method, endpoint, body) returned for all non-2xx
  responses, with helpers `AsAPIError`, `IsNotFound`, `IsRateLimited`, `IsPaymentRequired`,
  `IsUnauthorized`, `IsServerError`.
- `Client.StreamClient` — timeout-free HTTP client used for shard transfers and downloads;
  their lifetime is governed by the caller's `context.Context`.
- Offline test suite (`go test -short`, no credentials/network): request-payload pinning,
  full encrypt→upload→download→decrypt roundtrip against a mock bridge, unaligned/aligned
  range downloads, download guards, 429 + `Retry-After` retry, `APIError` classification,
  context cancellation.
- Integration test for download roundtrip, ranged download, and verified download.
- `FilesService.GetFiles` — paginated file listing (`GET /files`) with optional
  status/sort/order/updatedAt filters (`GetFilesOptions`).
- `FilesService.GetFileCount` — total number of files (`GET /files/count`). Takes no
  status filter: the live API rejects the documented `status` query parameter with
  HTTP 400 for every value (probed 2026-10-08).
- `FilesService.GetFileMetaByPath` — file metadata by full decrypted path
  (`GET /files/meta?path=…`), URL-encoded as a query value.

### Changed

- **BREAKING:** every public method (and `NewWithCredentials`) now takes a
  `context.Context` as its first argument. Cancellation and deadlines propagate to the
  underlying HTTP requests and to retry backoff sleeps.
- **BREAKING:** `FilesService.UpdateFileMeta` takes a dedicated `*UpdateFileMetaRequest`
  (`plainName`/`type` only, pointer fields with `omitempty`) instead of the full `File`
  struct, matching the API's `UpdateFileMetaDto` exactly.
- **BREAKING:** downloads now fail loudly instead of returning bad data:
  legacy version-1 files are rejected (`ErrFileVersionOne`) and files with more than one
  shard are rejected (`ErrMultiShardUnsupported`); shards are sorted by index.
- **BREAKING:** `GetUserCredentialsResponse.User.Mnemonic` is now a plain `string`
  (matching the API and the SDK's `UserResponseDto`); the former JWK-style
  `{type, data}` object shape no longer exists on the live API.
- **BREAKING:** `GetUserCredentialsResponse.User.RootFolderID` (was mapped to the
  string-typed `rootFolderId` JSON field) is now split to match the API and SDK:
  `RootFolderID int` (`root_folder_id`, legacy numeric id) plus a new
  `RootFolderUUID string` (`rootFolderId`, the root folder UUID).
- Uploads reject a non-positive size up front (`ErrInvalidUploadSize`).
- `Transfer` no longer applies a hardcoded 15-minute timeout; use the context.

### Removed

- **BREAKING:** unused legacy helpers `CalculateFileHash`, `GenerateBucketKey`,
  `GetDeterministicKey` (v1-era / duplicated by `GenerateFileBucketKey`).

### Fixed

- Transfers longer than 30 seconds were silently killed by the global `HTTPClient`
  timeout; they now run on `StreamClient` with ctx-controlled lifetime.
- `UpdateFileMeta` no longer sends ~25 zero-value fields (e.g. `"size":""`,
  `"createdAt":"0001-01-01T00:00:00Z"`) that the server only ignored by luck.
- `go test -short` no longer requires `INTERNXT_TEST_EMAIL`/`INTERNXT_TEST_PASSWORD`.
- `UsersService.GetUserCredentials` failed with an unmarshal error against the live API
  (`user.mnemonic` is returned as a hex string, not a `{type, data}` object).
