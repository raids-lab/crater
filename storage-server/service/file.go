package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"webdav/logutils"
	"webdav/model"
	"webdav/orm"
	"webdav/response"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/webdav"
)

var fs *webdav.Handler
var fsonce sync.Once
var clientonce sync.Once
var httpClient *http.Client

type Filereq struct {
	Userid    *int   `json:"userid" binding:"required"`
	Projectid *int   `json:"projectid"`
	Path      string `json:"path" `
}

type File struct {
	Name       string `json:"name"`
	Size       int64  `json:"size"`
	IsDir      bool   `json:"isdir"`
	ModifyTime string `json:"modifytime"`
	Sys        any    `json:"sys"`
}

const defaultFolderPerm = 0755

func checkfs() {
	fsonce.Do(func() {
		fs = &webdav.Handler{
			Prefix:     "/api/ss",
			FileSystem: webdav.Dir("/crater"),
			LockSystem: webdav.NewMemLS(),
		}
	})
}

func checkclient() {
	clientonce.Do(func() {
		httpClient = &http.Client{}
	})
}

func CheckFilePermission(userID, projectID uint) model.FilePermission {
	db := orm.DB()
	var UserPro model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ? AND project_id = ?", userID, projectID).First(&UserPro).Error
	if err != nil || UserPro.ID == 0 {
		fmt.Println("user has no this project, ", err)
		return model.NotAllowed
	}
	switch UserPro.Role {
	case model.RoleAdmin:
		return model.ReadWrite
	case model.RoleGuest:
		return model.NotAllowed
	case model.RoleUser:
		return model.ReadOnly
	}
	return model.NotAllowed
}

func CheckJWTToken(c *gin.Context) (model.TokenResp, error) {
	checkclient()
	url := "http://crater.act.buaa.edu.cn/api/v1/storage/verify"
	req, err := http.NewRequestWithContext(c.Request.Context(), "GET", url, http.NoBody)
	if err != nil {
		return model.TokenResp{}, fmt.Errorf("can't create request")
	}
	req.Header.Set("Authorization", c.GetHeader("Authorization"))
	req.Header.Set("Content-Type", "application/json")
	resp, err := httpClient.Do(req)
	if err != nil {
		return model.TokenResp{}, fmt.Errorf("can't get resp")
	}
	body, _ := io.ReadAll(resp.Body)
	var tokenResp model.TokenResp
	if err := json.Unmarshal([]byte(string(body)), &tokenResp); err != nil {
		return model.TokenResp{}, fmt.Errorf("returned json error")
	}
	defer resp.Body.Close()
	return tokenResp, nil
}

func ListMyProject(userID uint) []string {
	db := orm.DB()
	var UserPro []model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ?", userID).Find(&UserPro).Error
	if err != nil {
		fmt.Println("user has no project, ", err)
		return nil
	}
	var spacepath []string
	spacepath = nil
	for i := range UserPro {
		var space model.Space
		err = db.Model(&model.Space{}).Where("project_id = ?", UserPro[i].ProjectID).First(&space).Error
		if err == nil && space.ID != 0 {
			spacepath = append(spacepath, space.Path)
		}
	}
	var tmp model.Space
	err = db.Model(&model.Space{}).Where("project_id= 1").First(&tmp).Error
	if err == nil && tmp.ID != 0 {
		spacepath = append(spacepath, tmp.Path)
	}
	return spacepath
}

func GetMyProject(userID uint) model.Project {
	db := orm.DB()
	var UserPro []model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ?", userID).Find(&UserPro).Error
	if err != nil || UserPro[0].ID == 0 {
		fmt.Println("user has no project, ", err)
		return model.Project{}
	}
	for i := range UserPro {
		var project model.Project
		err := db.Model(&model.Project{}).Where("id = ? AND is_personal = ?", UserPro[i].ProjectID, true).First(&project).Error
		if err == nil && project.ID != 0 {
			return project
		}
	}
	return model.Project{}
}

func GetSpaceByProjectID(pid uint) string {
	db := orm.DB()
	var space model.Space
	err := db.Model(&model.Space{}).Where("project_id = ?", pid).First(&space).Error
	if err != nil && space.ID != 0 {
		return ""
	}
	return space.Path
}

func WebDav(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}

	if jwttoken.Data.Permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "MOVE", "LOCK", "UNLOCK"}
	if jwttoken.Data.Permission == model.ReadOnly && containsString(rwMethods, c.Request.Method) {
		c.String(http.StatusUnauthorized, "Unauthorized 2")
		return
	}
	http.StripPrefix("/api/ss", fs)
	fs.ServeHTTP(c.Writer, c.Request)
}

func AlloweOption(c *gin.Context) {
	c.Header("Content-Type", "application/json")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Header("Access-Control-Allow-Credentials", "true")
	c.Header("Access-Control-Allow-Headers", "*")
	c.Header("Access-Control-Allow-Methods", "*")
}

func Download(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}

	if jwttoken.Data.Permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	path := strings.TrimPrefix(c.Request.URL.Path, "api/ss/download/")
	fmt.Println(path)
	f, err := fs.FileSystem.OpenFile(c.Request.Context(), path, os.O_RDWR, 0)
	if err != nil {
		response.BadRequestError(c, "can't find file")
		return
	}
	defer f.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%q\"", c.Request.URL.Path))
	_, err = io.Copy(c.Writer, f)
	if err != nil {
		response.Error(c, "can't download file", response.NotSpecified)
		return
	}
}

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func GetMyProjectDir(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}
	myproject := ListMyProject(jwttoken.Data.UserID)
	response.Success(c, myproject)
}

func GetMyDir(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []File
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}
	path := jwttoken.Data.RootPath
	data, err = handleDirList(fs.FileSystem, c.Writer, path)
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}
	response.Success(c, data)
}

func GetFileByPaths(paths []string, c *gin.Context) ([]File, error) {
	var data []File
	data = nil
	for _, p := range paths {
		fi, err := fs.FileSystem.Stat(c.Request.Context(), p)
		if err != nil {
			fmt.Println("cann't find file:", err)
			return nil, err
		}
		var tmp File
		tmp.IsDir = fi.IsDir()
		tmp.ModifyTime = fi.ModTime().String()
		tmp.Name = fi.Name()
		tmp.Size = fi.Size()
		tmp.Sys = fi.Sys()
		data = append(data, tmp)
	}
	return data, nil
}

func GetFile(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []File
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/file")
	if jwttoken.Data.Permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	if param == "" || param == "/" {
		fmt.Println("userid:", jwttoken.Data.UserID)
		paths := ListMyProject(jwttoken.Data.UserID)
		fmt.Println(paths)
		data, err = GetFileByPaths(paths, c)
		if err != nil {
			response.Error(c, "no project or porject has no dir", response.NotSpecified)
			return
		}
		response.Success(c, data)
	} else {
		data, err = handleDirList(fs.FileSystem, c.Writer, param)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

func handleDirList(fs webdav.FileSystem, w http.ResponseWriter, path string) ([]File, error) {
	ctx := context.Background()
	f, err := fs.OpenFile(ctx, path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var files []File
	defer f.Close()
	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		logutils.Log.Info("cann't read a empty file")
		return files, nil
	}
	dirs, err := f.Readdir(-1)
	if err != nil {
		logutils.Log.Info(w, "Error reading directory", http.StatusInternalServerError)
		return nil, err
	}
	var tmp File
	for _, d := range dirs {
		tmp.Name = d.Name()
		tmp.ModifyTime = d.ModTime().String()
		tmp.Size = d.Size()
		tmp.IsDir = d.IsDir()
		tmp.Sys = d.Sys()
		files = append(files, tmp)
	}
	return files, nil
}

type SpacePaths struct {
	Paths []string `json:"paths"`
}

func CheckFilesExist(c *gin.Context) {
	checkfs()
	var paths SpacePaths
	if err := c.ShouldBind(&paths); err != nil {
		response.BadRequestError(c, err.Error())
	}
	for _, p := range paths.Paths {
		_, err := fs.FileSystem.Stat(c.Request.Context(), p)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", p)
				err = fs.FileSystem.Mkdir(c.Request.Context(), p, os.FileMode(defaultFolderPerm))
				if err != nil {
					response.Error(c, fmt.Sprintf("can't create dir:%s", p), response.NotSpecified)
				}
			}
		}
	}
}
