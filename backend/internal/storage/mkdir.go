package storage

import (
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
)

type createDirectoryHandlerDeps struct {
	authenticate func(*gin.Context) (util.JWTMessage, error)
	permission   func(string, util.JWTMessage, *gin.Context) model.FilePermission
	redirect     func(*gin.Context, string, util.JWTMessage) (string, error)
	openTarget   func(string, string, string) (*os.Root, string, error)
	mkdir        func(*os.Root, string, os.FileMode) error
	storageRoot  string
}

func defaultCreateDirectoryHandlerDeps() createDirectoryHandlerDeps {
	return createDirectoryHandlerDeps{
		authenticate: CheckJWTToken,
		permission:   GetPermission,
		redirect:     Redirect,
		openTarget:   openUploadTarget,
		mkdir:        createStorageDirectory,
		storageRoot:  storageRootDir,
	}
}

// CreateDirectory creates exactly one directory through the existing WebDAV
// MKCOL route while returning Crater's stable error envelope on failure.
func CreateDirectory(c *gin.Context) {
	AlloweOption(c)
	createDirectoryWithDeps(c, defaultCreateDirectoryHandlerDeps())
}

func createDirectoryWithDeps(c *gin.Context, deps createDirectoryHandlerDeps) {
	token, err := deps.authenticate(c)
	if err != nil {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid token"))
		return
	}

	logicalPath, err := normalizeWebDAVMutationLogicalPath(c.Param("path"))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid directory path"))
		return
	}
	if permission := deps.permission(logicalPath, token, c); permission != model.ReadWrite {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("write permission is required"))
		return
	}

	realPath, err := deps.redirect(c, logicalPath, token)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve directory path"))
		return
	}
	logicalRoot := strings.Split(logicalPath, "/")[0]
	realRoot, err := deps.redirect(c, logicalRoot, token)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve directory path"))
		return
	}

	parent, targetName, err := deps.openTarget(deps.storageRoot, realRoot, realPath)
	if err != nil {
		if errors.Is(err, errUploadParentInvalid) {
			if isUploadParentInfrastructureFailure(err) {
				resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
				return
			}
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("directory parent is unavailable"))
			return
		}
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
		return
	}
	defer parent.Close()

	if err := deps.mkdir(parent, targetName, model.RWXFolderPerm); err != nil {
		switch {
		case os.IsExist(err):
			resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("directory path already exists"))
		case os.IsNotExist(err):
			resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New("directory parent is unavailable"))
		default:
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to create directory"))
		}
		return
	}
	c.Status(http.StatusCreated)
}

func createStorageDirectory(parent *os.Root, name string, mode os.FileMode) error {
	return createStorageDirectoryWithChmod(parent, name, mode, chmodCreatedStorageDirectory)
}

func createStorageDirectoryWithChmod(parent *os.Root, name string, mode os.FileMode,
	chmod func(*os.Root, string, os.FileMode) error,
) error {
	if parent == nil || name == "" || name == "." || name == parentPathSegment ||
		mode.Perm() != mode {
		return errUploadParentInvalid
	}
	if err := parent.Mkdir(name, mode); err != nil {
		return err
	}
	if err := chmod(parent, name, mode); err != nil {
		// Remove only the empty entry; never recursively erase concurrent writes.
		if cleanupErr := parent.Remove(name); cleanupErr != nil {
			return errors.Join(err, cleanupErr)
		}
		return err
	}
	return nil
}
