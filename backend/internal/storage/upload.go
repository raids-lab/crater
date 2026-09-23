package storage

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

const (
	uploadFileMode         os.FileMode = 0o644
	uploadStageDirMode     os.FileMode = 0o700
	uploadStageFileMode    os.FileMode = 0o600
	uploadStageAttempts                = 16
	uploadStageRandomBytes             = 16
	uploadStagePayload                 = "payload"
	parentPathSegment                  = ".."
)

var (
	errUploadTargetExists     = errors.New("upload target exists")
	errUploadTargetNotRegular = errors.New("upload target is not a regular file")
	errUploadParentInvalid    = errors.New("upload parent is missing, invalid, or outside the authorized storage root")
)

type uploadSourceError struct {
	cause error
}

func (e *uploadSourceError) Error() string {
	return "read upload source: " + e.cause.Error()
}

func (e *uploadSourceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

type uploadOutcome struct {
	Bytes       int64
	Overwritten bool
}

type uploadResponse struct {
	RemotePath  string `json:"remote_path"`
	Bytes       int64  `json:"bytes"`
	Overwritten bool   `json:"overwritten"`
}

type uploadHandlerDeps struct {
	authenticate func(*gin.Context) (util.JWTMessage, error)
	permission   func(string, util.JWTMessage, *gin.Context) model.FilePermission
	redirect     func(*gin.Context, string, util.JWTMessage) (string, error)
	openTarget   func(string, string, string) (*os.Root, string, error)
	stagePublish func(io.Reader, *os.Root, string, bool, os.FileMode) (uploadOutcome, error)
	storageRoot  string
}

func defaultUploadHandlerDeps() uploadHandlerDeps {
	return uploadHandlerDeps{
		authenticate: CheckJWTToken,
		permission:   GetPermission,
		redirect:     Redirect,
		openTarget:   openUploadTarget,
		stagePublish: stageAndPublishFile,
		storageRoot:  storageRootDir,
	}
}

// UploadFile atomically publishes one raw request body into an ordinary-user
// storage path. It deliberately uses a dedicated endpoint because the bundled
// WebDAV PUT handler truncates an existing target before the request completes.
func UploadFile(c *gin.Context) {
	uploadFileWithDeps(c, defaultUploadHandlerDeps())
}

func uploadFileWithDeps(c *gin.Context, deps uploadHandlerDeps) {
	token, err := deps.authenticate(c)
	if err != nil {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid token"))
		return
	}

	overwrite, err := parseUploadOverwrite(c.Query("overwrite"))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("overwrite must be true or false"))
		return
	}

	logicalPath, err := normalizeUploadLogicalPath(c.Param("path"))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid remote file path"))
		return
	}

	if permission := deps.permission(logicalPath, token, c); permission != model.ReadWrite {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("write permission is required"))
		return
	}

	realPath, err := deps.redirect(c, logicalPath, token)
	if err != nil {
		klog.Errorf("resolve upload target: %v", err)
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve upload target"))
		return
	}
	logicalRoot := strings.Split(logicalPath, "/")[0]
	realRoot, err := deps.redirect(c, logicalRoot, token)
	if err != nil {
		klog.Errorf("resolve upload storage root: %v", err)
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve upload target"))
		return
	}

	parentRoot, targetName, err := deps.openTarget(deps.storageRoot, realRoot, realPath)
	if err != nil {
		if errors.Is(err, errUploadParentInvalid) {
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("upload parent directory is unavailable"))
			return
		}
		klog.Errorf("open upload target: %v", err)
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access upload storage"))
		return
	}
	defer parentRoot.Close()

	outcome, err := deps.stagePublish(c.Request.Body, parentRoot, targetName, overwrite, uploadFileMode)
	if err != nil {
		switch {
		case errors.Is(err, errUploadTargetExists):
			resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("target file already exists"))
		case errors.Is(err, errUploadTargetNotRegular):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("target path is not a regular file"))
		case errors.Is(err, errUploadParentInvalid):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("upload parent directory is unavailable"))
		default:
			var sourceErr *uploadSourceError
			if errors.As(err, &sourceErr) {
				resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(sourceErr, "failed to read upload body"))
				return
			}
			klog.Errorf("publish uploaded file: %v", err)
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to store uploaded file"))
		}
		return
	}

	status := http.StatusCreated
	if outcome.Overwritten {
		status = http.StatusOK
	}
	c.JSON(status, resputil.Response[uploadResponse]{
		Code: resputil.OK,
		Data: uploadResponse{
			RemotePath:  logicalPath,
			Bytes:       outcome.Bytes,
			Overwritten: outcome.Overwritten,
		},
		Message: "",
	})
}

func parseUploadOverwrite(raw string) (bool, error) {
	switch raw {
	case "", "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, errors.New("invalid overwrite value")
	}
}

func normalizeUploadLogicalPath(raw string) (string, error) {
	if strings.ContainsRune(raw, '\\') {
		return "", errors.New("backslashes are not allowed")
	}
	for _, character := range raw {
		if unicode.IsControl(character) {
			return "", errors.New("control characters are not allowed")
		}
	}

	trimmed := strings.Trim(raw, "/")
	rawSegments := strings.Split(trimmed, "/")
	segments := make([]string, 0, len(rawSegments))
	for _, segment := range rawSegments {
		if segment == parentPathSegment {
			return "", errors.New("parent traversal is not allowed")
		}
		if segment == "" || segment == "." {
			continue
		}
		segments = append(segments, segment)
	}
	if len(segments) < 2 {
		return "", errors.New("a file below a logical root is required")
	}
	switch segments[0] {
	case model.UserPath, model.PublicPath, model.AccountPath:
	default:
		return "", errors.New("invalid logical root")
	}
	return strings.Join(segments, "/"), nil
}

// openUploadTarget returns a directory handle anchored to the resolved target
// parent. os.Root rejects symlink traversal outside both the configured storage
// root and the caller's authorized real root, and remains anchored if a parent
// directory is renamed concurrently.
func openUploadTarget(storageRoot, authorizedRealRoot, targetRealPath string) (*os.Root, string, error) {
	storage, err := os.OpenRoot(storageRoot)
	if err != nil {
		return nil, "", err
	}
	defer storage.Close()

	if !strings.HasPrefix(targetRealPath, strings.TrimSuffix(authorizedRealRoot, "/")+"/") {
		return nil, "", errUploadParentInvalid
	}
	authorizedPath, err := cleanStorageRelativePath(authorizedRealRoot)
	if err != nil {
		return nil, "", errUploadParentInvalid
	}
	targetPath, err := cleanStorageRelativePath(targetRealPath)
	if err != nil {
		return nil, "", errUploadParentInvalid
	}
	targetRelative, err := filepath.Rel(authorizedPath, targetPath)
	if err != nil || targetRelative == "." || pathEscapesRoot(targetRelative) {
		return nil, "", errUploadParentInvalid
	}

	authorized, err := storage.OpenRoot(authorizedPath)
	if err != nil {
		return nil, "", errUploadParentInvalid
	}
	defer authorized.Close()

	parentRelative := filepath.Dir(targetRelative)
	targetName := filepath.Base(targetRelative)
	if targetName == "." || targetName == string(filepath.Separator) {
		return nil, "", errUploadParentInvalid
	}
	parent, err := authorized.OpenRoot(parentRelative)
	if err != nil {
		return nil, "", errUploadParentInvalid
	}
	return parent, targetName, nil
}

func cleanStorageRelativePath(raw string) (string, error) {
	if raw == "" || strings.ContainsRune(raw, '\\') {
		return "", errUploadParentInvalid
	}
	for _, character := range raw {
		if unicode.IsControl(character) {
			return "", errUploadParentInvalid
		}
	}

	raw, err := trimLegacyStorageRootSlash(raw)
	if err != nil {
		return "", err
	}
	canonical, err := canonicalStorageSegments(raw)
	if err != nil {
		return "", err
	}
	normalized := filepath.FromSlash(canonical)
	if filepath.Clean(normalized) != normalized || filepath.IsAbs(normalized) || pathEscapesRoot(normalized) {
		return "", errUploadParentInvalid
	}
	return normalized, nil
}

func trimLegacyStorageRootSlash(raw string) (string, error) {
	if !strings.HasPrefix(raw, "/") {
		return raw, nil
	}
	raw = strings.TrimPrefix(raw, "/")
	if raw == "" || strings.HasPrefix(raw, "/") {
		return "", errUploadParentInvalid
	}
	return raw, nil
}

func canonicalStorageSegments(raw string) (string, error) {
	// Historical User.Space and Account.Space records may start with "/".
	// Redirect joins them after the configured prefix and produces one empty
	// separator segment (for example "users//space/alice"). Accept exactly one
	// such legacy marker, while rejecting every other ambiguous empty segment.
	segments := strings.Split(raw, "/")
	canonical := make([]string, 0, len(segments))
	legacyEmptySeen := false
	for index, segment := range segments {
		if segment == "" {
			if legacyEmptySeen || index == 0 || index == len(segments)-1 {
				return "", errUploadParentInvalid
			}
			legacyEmptySeen = true
			continue
		}
		if segment == "." || segment == parentPathSegment {
			return "", errUploadParentInvalid
		}
		canonical = append(canonical, segment)
	}
	return strings.Join(canonical, "/"), nil
}

func pathEscapesRoot(path string) bool {
	return path == parentPathSegment ||
		strings.HasPrefix(path, parentPathSegment+string(filepath.Separator))
}

func stageAndPublishFile(
	source io.Reader,
	parent *os.Root,
	targetName string,
	overwrite bool,
	mode os.FileMode,
) (uploadOutcome, error) {
	if err := validateUploadTarget(parent, targetName, overwrite); err != nil {
		return uploadOutcome{}, err
	}

	stageName, stageRoot, staged, err := createUploadStage(parent)
	if err != nil {
		return uploadOutcome{}, err
	}
	defer cleanupUploadStage(parent, stageRoot, stageName)

	written, err := writeUploadStage(source, staged, mode)
	if err != nil {
		return uploadOutcome{}, err
	}
	return publishStagedUpload(stageRoot, parent, targetName, overwrite, written)
}

func validateUploadTarget(parent *os.Root, targetName string, overwrite bool) error {
	if parent == nil || targetName == "" || targetName == "." ||
		targetName == parentPathSegment || filepath.Base(targetName) != targetName {
		return errUploadParentInvalid
	}
	targetInfo, err := parent.Lstat(targetName)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !targetInfo.Mode().IsRegular() {
		return errUploadTargetNotRegular
	}
	if !overwrite {
		return errUploadTargetExists
	}
	return nil
}

func writeUploadStage(source io.Reader, staged *os.File, mode os.FileMode) (int64, error) {
	tracked := &trackedUploadSource{source: source}
	written, copyErr := io.Copy(staged, tracked)
	if copyErr != nil {
		_ = staged.Close()
		if tracked.readErr != nil {
			return 0, &uploadSourceError{cause: tracked.readErr}
		}
		return 0, copyErr
	}
	if err := staged.Chmod(mode); err != nil {
		_ = staged.Close()
		return 0, err
	}
	if err := staged.Sync(); err != nil {
		_ = staged.Close()
		return 0, err
	}
	if err := staged.Close(); err != nil {
		return 0, err
	}
	return written, nil
}

func publishStagedUpload(
	stageRoot, parent *os.Root,
	targetName string,
	overwrite bool,
	written int64,
) (uploadOutcome, error) {
	for retry := 0; retry < 2; retry++ {
		if err := publishUploadNoClobber(stageRoot, parent, targetName); err == nil {
			return uploadOutcome{Bytes: written}, nil
		} else if !os.IsExist(err) {
			return uploadOutcome{}, err
		}

		targetInfo, statErr := parent.Lstat(targetName)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return uploadOutcome{}, statErr
		}
		if !targetInfo.Mode().IsRegular() {
			return uploadOutcome{}, errUploadTargetNotRegular
		}
		if !overwrite {
			return uploadOutcome{}, errUploadTargetExists
		}
		if err := publishUploadOverwrite(stageRoot, parent, targetName); err != nil {
			return uploadOutcome{}, err
		}
		return uploadOutcome{Bytes: written, Overwritten: true}, nil
	}
	return uploadOutcome{}, errUploadTargetExists
}

func createUploadStage(parent *os.Root) (string, *os.Root, *os.File, error) {
	for attempt := 0; attempt < uploadStageAttempts; attempt++ {
		random := make([]byte, uploadStageRandomBytes)
		if _, err := rand.Read(random); err != nil {
			return "", nil, nil, err
		}
		stageName := ".crater-upload-" + hex.EncodeToString(random)
		if err := parent.Mkdir(stageName, uploadStageDirMode); err != nil {
			if os.IsExist(err) {
				continue
			}
			return "", nil, nil, err
		}
		stageRoot, err := parent.OpenRoot(stageName)
		if err != nil {
			_ = parent.Remove(stageName)
			return "", nil, nil, err
		}
		staged, err := stageRoot.OpenFile(
			uploadStagePayload,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			uploadStageFileMode,
		)
		if err != nil {
			_ = stageRoot.Close()
			_ = parent.Remove(stageName)
			return "", nil, nil, err
		}
		return stageName, stageRoot, staged, nil
	}
	return "", nil, nil, errors.New("could not allocate a private upload staging directory")
}

func cleanupUploadStage(parent, stageRoot *os.Root, stageName string) {
	if stageRoot != nil {
		_ = stageRoot.Remove(uploadStagePayload)
		_ = stageRoot.Close()
	}
	if parent != nil && stageName != "" {
		_ = parent.Remove(stageName)
	}
}

type trackedUploadSource struct {
	source  io.Reader
	readErr error
}

func (reader *trackedUploadSource) Read(data []byte) (int, error) {
	read, err := reader.source.Read(data)
	if err != nil && !errors.Is(err, io.EOF) {
		reader.readErr = err
	}
	return read, err
}
