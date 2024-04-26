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
	"time"
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

type Files struct {
	Name       string    `json:"name"`
	Filename   string    `json:"filename"`
	Size       int64     `json:"size"`
	IsDir      bool      `json:"isdir"`
	ModifyTime time.Time `json:"modifytime"`
	Sys        any       `json:"sys"`
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

func ListMyProjects(userID uint) []Files {
	db := orm.DB()
	var UserPro []model.UserProject
	var data []Files
	err := db.Model(&model.UserProject{}).Where("user_id = ?", userID).Find(&UserPro).Error
	if err != nil {
		fmt.Println("user has no project, ", err)
		return nil
	}
	for i := range UserPro {
		var space model.Space
		var pro model.Project
		var tmp Files
		err = db.Model(&model.Space{}).Where("project_id = ?", UserPro[i].ProjectID).First(&space).Error
		if err == nil {
			tmp.Name = space.Path
		}
		err = db.Model(&model.Project{}).Where("id = ?", UserPro[i].ProjectID).First(&pro).Error
		if err == nil {
			tmp.Filename = pro.Name
		}
		if tmp.Filename != "" && tmp.Name != "" {
			data = append(data, tmp)
		}
	}
	var stmp model.Space
	var ftmp Files
	err = db.Model(&model.Space{}).Where("project_id= 1").First(&stmp).Error
	if err == nil {
		ftmp.Name = stmp.Path
	}
	var ptmp model.Project
	err = db.Model(&model.Project{}).Where("id= 1").First(&ptmp).Error
	if err == nil {
		ftmp.Filename = ptmp.Name
	}
	if ftmp.Filename != "" && ftmp.Name != "" {
		data = append(data, ftmp)
	}
	return data
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
	origin := c.Request.Header.Get("Origin")
	if origin != "" {
		c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
		c.Header("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE,MKCOL,PROPFIND,PROPPATCH,MOVE,COPY")
		c.Header("Content-Type", "application/json")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Length,Token,session,Accept,"+
			"Origin, Host, Connection, Accept-Encoding, Accept-Language,DNT, X-CustomHeader, X-Requested-With,"+
			"Content-Type, Destination,X-Debug-Username")
	}
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
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/download/")
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

func GetMyDir(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}
	path := jwttoken.Data.RootPath
	data, err = handleDirsList(fs.FileSystem, c.Writer, path)
	if err != nil {
		response.BadRequestError(c, err.Error())
		return
	}
	response.Success(c, data)
}

func GetFilesByPaths(paths []Files, c *gin.Context) ([]Files, error) {
	var data []Files
	data = nil
	for _, p := range paths {
		fi, err := fs.FileSystem.Stat(c.Request.Context(), p.Name)
		if err != nil {
			fmt.Println("cann't find file:", err)
			return nil, err
		}
		var tmp Files
		tmp.IsDir = fi.IsDir()
		tmp.ModifyTime = fi.ModTime()
		tmp.Name = fi.Name()
		tmp.Filename = p.Filename
		tmp.Size = fi.Size()
		tmp.Sys = fi.Sys()
		data = append(data, tmp)
	}
	return data, nil
}

func GetFiles(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		response.Error(c, jwttoken.Msg, response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/files")
	if jwttoken.Data.Permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	if param == "" || param == "/" {
		paths := ListMyProjects(jwttoken.Data.UserID)
		fmt.Println(paths)
		data, err = GetFilesByPaths(paths, c)
		if err != nil {
			response.Error(c, "no project or porject has no dir", response.NotSpecified)
			return
		}
		response.Success(c, data)
	} else {
		data, err = handleDirsList(fs.FileSystem, c.Writer, param)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

func handleDirsList(fs webdav.FileSystem, w http.ResponseWriter, path string) ([]Files, error) {
	ctx := context.Background()
	f, err := fs.OpenFile(ctx, path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var files []Files
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
	var tmp Files
	for _, d := range dirs {
		tmp.Name = d.Name()
		tmp.Filename = d.Name()
		tmp.ModifyTime = d.ModTime()
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
