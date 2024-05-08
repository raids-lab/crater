package service

import (
	"context"
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
	"webdav/util"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/webdav"
)

var fs *webdav.Handler
var fsonce sync.Once

type Files struct {
	Name       string    `json:"name"`
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

func CheckJWTToken(c *gin.Context) (util.JWTMessage, error) {
	var tmp util.JWTMessage
	authHeader := c.Request.Header.Get("Authorization")
	t := strings.Split(authHeader, " ")
	if len(t) < 2 || t[0] != "Bearer" {
		return tmp, fmt.Errorf("invalid token")
	}
	authToken := t[1]
	token, err := util.GetTokenMgr().CheckToken(authToken)
	if err != nil {
		return tmp, err
	}
	return token, nil
}

func GetPermissionFromToken(token util.JWTMessage) model.FilePermission {
	if token.QueueID == util.QueueIDNull {
		return model.ReadWrite
	} else {
		return model.FilePermission(token.RoleQueue)
	}
}

func ListMyProjects(userID uint) []Files {
	db := orm.DB()
	var data []Files
	var user model.User
	var ftmp Files
	err := db.Model(&model.User{}).Where("id= ?", userID).First(&user).Error
	if err == nil {
		ftmp.Name = user.Space
	}
	if ftmp.Name != "" {
		data = append(data, ftmp)
	}

	var userQueue []model.UserQueue
	err = db.Model(&model.UserQueue{}).Where("user_id = ?", userID).Find(&userQueue).Error
	if err != nil || userQueue[0].ID == 0 {
		fmt.Println("user has no project, ", err)
		return data
	}
	for i := range userQueue {
		var queue model.Queue
		var tmp Files
		err = db.Model(&model.Queue{}).Where("id = ?", userQueue[i].QueueID).First(&queue).Error
		if err == nil {
			tmp.Name = queue.Space
		}
		if tmp.Name != "" {
			data = append(data, tmp)
		}
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
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	permission := GetPermissionFromToken(jwttoken)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "MOVE", "LOCK", "UNLOCK"}
	if permission == model.ReadOnly && containsString(rwMethods, c.Request.Method) {
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
		c.Header("Content-Type", "application/json; charset=utf-8 ")
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
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	permission := GetPermissionFromToken(jwttoken)
	if permission == model.NotAllowed {
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

// func GetMyDir(c *gin.Context) {
// 	AlloweOption(c)
// 	checkfs()
// 	var data []Files
// 	jwttoken, err := CheckJWTToken(c)
// 	if err != nil {
// 		response.Error(c, err.Error(), response.NotSpecified)
// 		return
// 	}
// 	path := jwttoken.Data.RootPath
// 	data, err = handleDirsList(fs.FileSystem, c.Writer, path)
// 	if err != nil {
// 		response.BadRequestError(c, err.Error())
// 		return
// 	}
// 	response.Success(c, data)
// }

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
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/files")
	permission := GetPermissionFromToken(jwttoken)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	if param == "" || param == "/" {
		paths := ListMyProjects(jwttoken.UserID)
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
					return
				}
			}
		}
	}
	response.Success(c, "create dir success")
}
