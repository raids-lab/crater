package storage

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"

	"github.com/gin-gonic/gin"
)

type MoveFileReq struct {
	Dst string `json:"dst"  binding:"required"`
}

var (
	errMoveSourceNotFound       = errors.New("move source does not exist")
	errMoveTargetExists         = errors.New("move destination exists")
	errMoveNoReplaceUnsupported = errors.New("atomic no-replace move is unsupported")
)

type moveFileHandlerDeps struct {
	authenticate func(*gin.Context) (util.JWTMessage, error)
	permission   func(string, util.JWTMessage, *gin.Context) model.FilePermission
	redirect     func(*gin.Context, string, util.JWTMessage) (string, error)
	openTarget   func(string, string, string) (*os.Root, string, error)
	move         func(*os.Root, string, *os.Root, string) error
	storageRoot  string
}

func defaultMoveFileHandlerDeps() moveFileHandlerDeps {
	return moveFileHandlerDeps{
		authenticate: CheckJWTToken,
		permission:   GetPermission,
		redirect:     Redirect,
		openTarget:   openUploadTarget,
		move:         moveStorageEntry,
		storageRoot:  storageRootDir,
	}
}

func MoveFile(c *gin.Context) {
	AlloweOption(c)
	moveFileWithDeps(c, defaultMoveFileHandlerDeps())
}

//nolint:gocyclo // Keep each authorization and filesystem failure mapped to its specific public error contract.
func moveFileWithDeps(c *gin.Context, deps moveFileHandlerDeps) {
	jwttoken, err := deps.authenticate(c)
	if err != nil {
		resputil.HandleError(c, bizerr.Auth.TokenInvalid.New("invalid token"))
		return
	}
	var moveFileReq MoveFileReq
	if err := c.ShouldBindJSON(&moveFileReq); err != nil {
		resputil.HandleError(c, bizerr.BadRequest.InvalidRequest.Wrap(err, "invalid move request"))
		return
	}

	sourcePath, err := normalizeWebDAVMutationLogicalPath(c.Param("path"))
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid source path"))
		return
	}
	destinationPath, err := normalizeWebDAVMutationLogicalPath(moveFileReq.Dst)
	if err != nil {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("invalid destination path"))
		return
	}
	if sourcePath == destinationPath || strings.HasPrefix(destinationPath, sourcePath+"/") {
		resputil.HandleError(c, bizerr.BadRequest.ParameterError.New("destination must be outside the source path"))
		return
	}

	sourcePermission := deps.permission(sourcePath, jwttoken, c)
	dstPermission := deps.permission(destinationPath, jwttoken, c)
	if sourcePermission != model.ReadWrite || dstPermission != model.ReadWrite {
		resputil.HandleError(c, bizerr.Forbidden.PermissionDenied.New("write permission is required for source and destination"))
		return
	}

	realSource, err := deps.redirect(c, sourcePath, jwttoken)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve move source"))
		return
	}
	realDestination, err := deps.redirect(c, destinationPath, jwttoken)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve move destination"))
		return
	}

	sourceRoot, err := deps.redirect(c, strings.Split(sourcePath, "/")[0], jwttoken)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve move source"))
		return
	}
	destinationRoot, err := deps.redirect(c, strings.Split(destinationPath, "/")[0], jwttoken)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to resolve move destination"))
		return
	}

	sourceParent, sourceName, err := deps.openTarget(deps.storageRoot, sourceRoot, realSource)
	if err != nil {
		handleMoveTargetOpenError(c, err, true, "source parent directory is unavailable")
		return
	}
	defer sourceParent.Close()
	destinationParent, destinationName, err := deps.openTarget(
		deps.storageRoot,
		destinationRoot,
		realDestination,
	)
	if err != nil {
		handleMoveTargetOpenError(c, err, false, "destination parent directory is unavailable")
		return
	}
	defer destinationParent.Close()

	if err := deps.move(sourceParent, sourceName, destinationParent, destinationName); err != nil {
		switch {
		case errors.Is(err, errMoveSourceNotFound):
			resputil.HandleError(c, bizerr.NotFound.StorageResourceNotFound.New("source path does not exist"))
		case errors.Is(err, errMoveTargetExists):
			resputil.HandleError(c, bizerr.Conflict.ResourceAlreadyExists.New("destination path already exists"))
		default:
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to move storage entry"))
		}
		return
	}

	resputil.Success(c, "move files successfully")
}

func handleMoveTargetOpenError(c *gin.Context, err error, source bool, message string) {
	if errors.Is(err, errUploadParentInvalid) {
		if source && isUploadParentMissing(err) {
			resputil.HandleError(c, bizerr.NotFound.StorageResourceNotFound.New("source path does not exist"))
			return
		}
		if isUploadParentInfrastructureFailure(err) {
			resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
			return
		}
		resputil.HandleError(c, bizerr.Conflict.ResourceStatusError.New(message))
		return
	}
	resputil.HandleError(c, bizerr.Internal.FileSystemError.Wrap(err, "failed to access storage"))
}

//nolint:gocyclo // The explicit checks preserve no-clobber and source-not-found semantics around one rename.
func moveStorageEntry(
	sourceParent *os.Root,
	sourceName string,
	destinationParent *os.Root,
	destinationName string,
) error {
	if sourceParent == nil || destinationParent == nil ||
		sourceName == "" || sourceName == "." || sourceName == parentPathSegment ||
		destinationName == "" || destinationName == "." || destinationName == parentPathSegment ||
		filepath.Base(sourceName) != sourceName || filepath.Base(destinationName) != destinationName {
		return errUploadParentInvalid
	}
	if _, err := sourceParent.Lstat(sourceName); err != nil {
		if os.IsNotExist(err) {
			return errMoveSourceNotFound
		}
		return err
	}
	if _, err := destinationParent.Lstat(destinationName); err == nil {
		return errMoveTargetExists
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := renameStorageNoReplace(sourceParent, sourceName, destinationParent, destinationName); err != nil {
		if os.IsExist(err) {
			return errMoveTargetExists
		}
		if os.IsNotExist(err) {
			if _, sourceErr := sourceParent.Lstat(sourceName); os.IsNotExist(sourceErr) {
				return errMoveSourceNotFound
			}
		}
		return err
	}
	return nil
}

func MoveDatasetOrModel(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	var datasetReq DatasetRequest
	if err = c.ShouldBindUri(&datasetReq); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", resputil.NotSpecified)
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetReq.ID)).First()
	if err != nil {
		resputil.Error(c, "Dataset don't exist", resputil.NotSpecified)
		return
	}
	var dest string
	switch dataset.Type {
	case model.DataTypeModel:
		dest = model.ModelPrefix
	case model.DataTypeDataset:
		dest = model.DatasetPrefix
	default:
		resputil.Error(c, "The type of dataset is incorrect", resputil.NotSpecified)
		return
	}
	dest = dest + "/" + strconv.FormatUint(uint64(datasetReq.ID), 10)
	dest = filepath.Join(dest, filepath.Base(dataset.URL))
	err = moveFiles(c.Request.Context(), dataset.URL, dest, false)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	dataset.URL = dest
	if _, err := d.WithContext(c).Updates(dataset); err != nil {
		resputil.Error(c, "failed to update dataset URL", resputil.NotSpecified)
		return
	}
	resputil.Success(c, "move dataset or model successfully")
}

type RestoreFileReq struct {
	ID  uint   `json:"id" binding:"required"`
	Dst string `json:"dst"  binding:"required"`
}

// 浼犺繘鏉ョ殑鐩爣璺緞搴旇鏄疄闄呰矾寰勶紝鑰屼笉鑳芥槸user/111杩欐牱鐨勮櫄鎷熻矾寰?
func RestoreDatasetOrModel(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	var restoreFileReq RestoreFileReq
	if err = c.ShouldBind(&restoreFileReq); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", resputil.NotSpecified)
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(restoreFileReq.ID)).First()
	if err != nil {
		resputil.Error(c, "Dataset don't exist", resputil.NotSpecified)
		return
	}
	soure := dataset.URL
	dstPath := restoreFileReq.Dst

	if stat, ferr := fs.FileSystem.Stat(c.Request.Context(), dstPath); ferr == nil && stat.IsDir() {
		srcName := filepath.Base(soure)
		dstPath = filepath.Join(dstPath, srcName)
	}
	err = moveFiles(c.Request.Context(), soure, dstPath, false)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	dataset.URL = dstPath
	if _, err := d.WithContext(c).Updates(dataset); err != nil {
		resputil.Error(c, "failed to update dataset URL", resputil.NotSpecified)
		return
	}
	resputil.Success(c, "restore dataset or model successfully")
}

func moveFiles(ctx context.Context, src, dst string, overwrite bool) error {
	if !overwrite {
		if _, err := fs.FileSystem.Stat(ctx, dst); err == nil {
			return fmt.Errorf("destination %s already exists", dst)
		} else if !os.IsNotExist(err) {
			return err
		}
	} else {
		if _, err := fs.FileSystem.Stat(ctx, dst); err == nil {
			if rerr := fs.FileSystem.RemoveAll(ctx, dst); rerr != nil {
				return rerr
			}
		} else if !os.IsNotExist(err) {
			return err
		}
	}

	dstDir := filepath.Dir(dst)
	if _, err := fs.FileSystem.Stat(ctx, dstDir); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		if err := fs.FileSystem.Mkdir(ctx, dstDir, model.RWXFolderPerm); err != nil {
			return err
		}
	}

	return fs.FileSystem.Rename(ctx, src, dst)
}

func RegisterDataset(webdavGroup *gin.RouterGroup) {
	webdavGroup.POST("/move/*path", MoveFile)
	webdavGroup.POST("/datasets/:id/move", MoveDatasetOrModel)
	webdavGroup.POST("/datasets/restore", RestoreDatasetOrModel)
}
