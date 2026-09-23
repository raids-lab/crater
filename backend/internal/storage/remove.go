package storage

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unicode"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

var (
	errRemoveTargetNotFound       = errors.New("remove target does not exist")
	errRemoveRecursiveRequired    = errors.New("recursive removal is required for a directory")
	errRemoveTargetChanged        = errors.New("remove target changed during the operation")
	errRemoveCrossDevice          = errors.New("recursive removal cannot cross a filesystem boundary")
	errRemoveOperationUnsupported = errors.New("safe removal is unsupported on this platform")
)

type removeFileResponse struct {
	RemotePath string `json:"remote_path"`
	Recursive  bool   `json:"recursive"`
}

type removeFileHandlerDeps struct {
	authenticate func(*gin.Context) (util.JWTMessage, error)
	permission   func(string, util.JWTMessage, *gin.Context) model.FilePermission
	redirect     func(*gin.Context, string, util.JWTMessage) (string, error)
	openTarget   func(string, string, string) (*os.Root, string, error)
	remove       func(*os.Root, string, bool) error
	storageRoot  string
}

func defaultRemoveFileHandlerDeps() removeFileHandlerDeps {
	return removeFileHandlerDeps{
		authenticate: CheckJWTToken,
		permission:   GetPermission,
		redirect:     Redirect,
		openTarget:   openUploadTarget,
		remove:       removeStorageEntry,
		storageRoot:  storageRootDir,
	}
}

// RemoveFile deletes exactly one ordinary-user storage path. It is separate
// from the legacy /delete route so callers must explicitly opt in before a
// directory tree can be removed.
func RemoveFile(c *gin.Context) {
	removeFileWithDeps(c, defaultRemoveFileHandlerDeps())
}

func removeFileWithDeps(c *gin.Context, deps removeFileHandlerDeps) {
	token, err := deps.authenticate(c)
	if err != nil {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid token"))
		return
	}

	recursiveValues, present := c.Request.URL.Query()["recursive"]
	if !present {
		resputil.HandleError(c, bizerr.BadRequest.MissingParameter.New("recursive is required"))
		return
	}
	if len(recursiveValues) != 1 {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("recursive must be true or false"))
		return
	}
	recursive, err := parseRemoveRecursive(recursiveValues[0])
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("recursive must be true or false"))
		return
	}

	logicalPath, err := normalizeRemoveLogicalPath(c.Param("path"))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid remote path"))
		return
	}
	if permission := deps.permission(logicalPath, token, c); permission != model.ReadWrite {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("write permission is required"))
		return
	}

	realPath, err := deps.redirect(c, logicalPath, token)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve remove target"))
		return
	}
	logicalRoot := strings.SplitN(logicalPath, "/", 2)[0]
	realRoot, err := deps.redirect(c, logicalRoot, token)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve remove target"))
		return
	}

	parent, targetName, err := deps.openTarget(deps.storageRoot, realRoot, realPath)
	if err != nil {
		handleRemoveTargetOpenError(c, err)
		return
	}
	defer parent.Close()

	if err := deps.remove(parent, targetName, recursive); err != nil {
		switch {
		case errors.Is(err, errRemoveTargetNotFound):
			resputil.HandleError(c, bizerr.NotFound.StorageResourceNotFound.New("remote path does not exist"))
		case errors.Is(err, errRemoveRecursiveRequired):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("directory removal requires recursive=true"))
		case errors.Is(err, errRemoveTargetChanged):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("remote path changed during removal"))
		case errors.Is(err, errRemoveCrossDevice):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("remote path crosses a filesystem boundary"))
		default:
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to remove storage entry"))
		}
		return
	}

	c.JSON(http.StatusOK, resputil.Response[removeFileResponse]{
		Code: resputil.OK,
		Data: removeFileResponse{
			RemotePath: logicalPath,
			Recursive:  recursive,
		},
		Message: "",
	})
}

func handleRemoveTargetOpenError(c *gin.Context, err error) {
	if errors.Is(err, errUploadParentInvalid) {
		if isUploadParentMissing(err) {
			resputil.HandleError(c, bizerr.NotFound.StorageResourceNotFound.New("remote path does not exist"))
			return
		}
		if isUploadParentInfrastructureFailure(err) {
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
			return
		}
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("remote path parent is unavailable"))
		return
	}
	resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
}

func parseRemoveRecursive(raw string) (bool, error) {
	switch raw {
	case "false":
		return false, nil
	case "true":
		return true, nil
	default:
		return false, errors.New("invalid recursive value")
	}
}

func normalizeRemoveLogicalPath(raw string) (string, error) {
	if strings.ContainsRune(raw, '\\') {
		return "", errors.New("backslashes are not allowed")
	}
	for _, character := range raw {
		if unicode.IsControl(character) {
			return "", errors.New("control characters are not allowed")
		}
	}

	raw = strings.TrimPrefix(raw, "/")
	if raw == "" || strings.HasPrefix(raw, "/") || strings.HasSuffix(raw, "/") {
		return "", errors.New("empty path segments are not allowed")
	}
	segments := strings.Split(raw, "/")
	if len(segments) < 2 {
		return "", errors.New("a path below a logical root is required")
	}
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == parentPathSegment {
			return "", errors.New("ambiguous path segments are not allowed")
		}
	}
	switch segments[0] {
	case model.UserPath, model.PublicPath, model.AccountPath:
	default:
		return "", errors.New("invalid logical root")
	}
	return strings.Join(segments, "/"), nil
}

type removeStorageEntryDeps struct {
	lstat              func(*os.Root, string) (os.FileInfo, error)
	unlinkNonDirectory func(*os.Root, string) error
	removeDirectory    func(*os.Root, string) error
}

func defaultRemoveStorageEntryDeps() removeStorageEntryDeps {
	return removeStorageEntryDeps{
		lstat: func(parent *os.Root, name string) (os.FileInfo, error) {
			return parent.Lstat(name)
		},
		unlinkNonDirectory: removeStorageNonDirectory,
		removeDirectory:    removeStorageDirectoryRecursive,
	}
}

func removeStorageEntry(parent *os.Root, name string, recursive bool) error {
	return removeStorageEntryWithDeps(parent, name, recursive, defaultRemoveStorageEntryDeps())
}

//nolint:gocyclo // Type-race classification must never upgrade a non-directory unlink into recursive removal.
func removeStorageEntryWithDeps(
	parent *os.Root,
	name string,
	recursive bool,
	deps removeStorageEntryDeps,
) error {
	if parent == nil || name == "" || name == "." || name == parentPathSegment ||
		filepath.Base(name) != name {
		return errUploadParentInvalid
	}

	info, err := deps.lstat(parent, name)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
			return errRemoveTargetNotFound
		}
		return err
	}
	if info.IsDir() {
		if !recursive {
			return errRemoveRecursiveRequired
		}
		return deps.removeDirectory(parent, name)
	}

	err = deps.unlinkNonDirectory(parent, name)
	if err == nil {
		return nil
	}
	if os.IsNotExist(err) || errors.Is(err, syscall.ENOTDIR) {
		return errRemoveTargetNotFound
	}
	if errors.Is(err, syscall.EISDIR) || errors.Is(err, syscall.EPERM) {
		current, statErr := deps.lstat(parent, name)
		switch {
		case statErr == nil && current.IsDir():
			return errRemoveTargetChanged
		case os.IsNotExist(statErr), errors.Is(statErr, syscall.ENOTDIR):
			return errRemoveTargetNotFound
		case statErr != nil:
			return statErr
		}
	}
	return err
}
