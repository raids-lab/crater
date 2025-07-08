package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"webdav/dao/model"
	"webdav/dao/query"
	"webdav/response"

	"github.com/gin-gonic/gin"
)

type MoveFileReq struct {
	Dst string `json:"dst"  binding:"required"`
}

func MoveFile(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	var moveFileReq MoveFileReq
	err = c.ShouldBind(&moveFileReq)
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/move")
	sourcePermission := GetPermission(param, jwttoken, c)
	dstPermission := GetPermission(moveFileReq.Dst, jwttoken, c)
	if sourcePermission != model.ReadWrite || dstPermission != model.ReadWrite {
		response.HTTPError(c, http.StatusUnauthorized, "You have no permission to move files or move files to this location ",
			response.NotSpecified)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	realDst, err := Redirect(c, moveFileReq.Dst, jwttoken)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
	}
	err = moveFiles(c.Request.Context(), realPath, realDst, false)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	response.Success(c, "move files successfully")
}

func MoveDatasetOrModel(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	var datasetReq DatasetRequest
	if err = c.ShouldBindUri(&datasetReq); err != nil {
		response.HTTPError(c, http.StatusBadRequest, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		response.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", response.NotSpecified)
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetReq.ID)).First()
	if err != nil {
		response.Error(c, "Dataset don't exist", response.NotSpecified)
		return
	}
	var dest string
	if dataset.Type == model.DataTypeModel {
		dest = model.ModelPrefix
	} else if dataset.Type == model.DataTypeDataset {
		dest = model.DatasetPrefix
	} else {
		response.Error(c, "The type of dataset is incorrect", response.NotSpecified)
		return
	}
	dest = dest + "/" + strconv.FormatUint(uint64(datasetReq.ID), 10)
	dest = filepath.Join(dest, filepath.Base(dataset.URL))
	err = moveFiles(c.Request.Context(), dataset.URL, dest, false)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	dataset.URL = dest
	if _, err := d.WithContext(c).Updates(dataset); err != nil {
		response.Error(c, "failed to update dataset URL", response.NotSpecified)
		return
	}
	response.Success(c, "move dataset or model successfully")
}

type RestoreFileReq struct {
	ID  uint   `json:"id" binding:"required"`
	Dst string `json:"dst"  binding:"required"`
}

// 传进来的目标路径应该是实际路径，而不能是user/111这样的虚拟路径
func RestoreDatasetOrModel(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	var restoreFileReq RestoreFileReq
	if err = c.ShouldBind(&restoreFileReq); err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		response.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", response.NotSpecified)
		return
	}
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(restoreFileReq.ID)).First()
	if err != nil {
		response.Error(c, "Dataset don't exist", response.NotSpecified)
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
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	dataset.URL = dstPath
	if _, err := d.WithContext(c).Updates(dataset); err != nil {
		response.Error(c, "failed to update dataset URL", response.NotSpecified)
		return
	}
	response.Success(c, "restore dataset or model successfully")
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
