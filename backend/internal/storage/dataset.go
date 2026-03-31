package storage

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"

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
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	var moveFileReq MoveFileReq
	err = c.ShouldBind(&moveFileReq)
	if err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/move")
	sourcePermission := GetPermission(param, jwttoken, c)
	dstPermission := GetPermission(moveFileReq.Dst, jwttoken, c)
	if sourcePermission != model.ReadWrite || dstPermission != model.ReadWrite {
		resputil.HTTPError(c, http.StatusUnauthorized, "You have no permission to move files or move files to this location ",
			resputil.NotSpecified)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	realDst, err := Redirect(c, moveFileReq.Dst, jwttoken)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
	}
	err = moveFiles(c.Request.Context(), realPath, realDst, false)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "move files successfully")
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
