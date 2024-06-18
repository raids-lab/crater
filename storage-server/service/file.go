package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"webdav/dao/model"
	"webdav/dao/query"
	"webdav/logutils"
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

type Permissions struct {
	Queue  model.FilePermission
	Public model.FilePermission
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
	if token.RolePlatform == model.RoleAdmin {
		return model.ReadWrite
	} else if token.QueueID == util.QueueIDNull {
		return model.FilePermission(token.PublicAccessMode)
	} else {
		return model.FilePermission(token.AccessMode)
	}
}

func ListCurrentProjects(userID, queueID uint, c *gin.Context) []string {
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if err != nil {
		fmt.Println("can't find user")
		return nil
	}
	var data []string
	if user.Space != "" {
		data = append(data, user.Space)
	}
	q := query.Queue
	publicqueue, err := q.WithContext(c).Where(q.ID.Eq(1)).First()
	if err != nil {
		fmt.Println("can't find public queue, ", err)
		return data
	}
	data = append(data, publicqueue.Space)
	if queueID != 0 && queueID != 1 {
		queue, err := q.WithContext(c).Where(q.ID.Eq(queueID)).First()
		if err != nil {
			fmt.Println("user has no project, ", err)
			return data
		}
		data = append(data, queue.Space)
	}
	return data
}

func ListQueueProjects(c *gin.Context) []string {
	var data []string
	q := query.Queue
	queues, err := q.WithContext(c).Where(q.ID.IsNotNull()).Find()
	if err != nil || len(queues) == 0 {
		fmt.Println("can't find queue, ", err)
		return data
	}
	for i := range queues {
		if queues[i].Space != "" {
			data = append(data, queues[i].Space)
		}
	}
	return data
}

func ListUserProjects(c *gin.Context) []string {
	var data []string
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.IsNotNull()).Find()
	if err != nil || len(user) == 0 {
		fmt.Println("can't find user, ", err)
		return data
	}
	for i := range user {
		if user[i].Space != "" {
			data = append(data, user[i].Space)
		}
	}
	return data
}

func ListMyProjects(userID uint, c *gin.Context) []string {
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if err != nil {
		fmt.Println("can't find user")
		return nil
	}
	var data []string
	ftmp := user.Space
	if ftmp != "" {
		data = append(data, ftmp)
	}
	uq := query.UserQueue
	q := query.Queue
	userQueue, err := uq.WithContext(c).Where(uq.UserID.Eq(userID)).Find()
	if err != nil || userQueue[0].ID == 0 {
		fmt.Println("user has no project, ", err)
		return data
	}
	for i := range userQueue {
		var tmp string
		queue, err := q.WithContext(c).Where(q.ID.Eq(userQueue[i].QueueID)).First()
		if err == nil {
			tmp = queue.Space
		}
		if tmp != "" {
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
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "MOVE", "LOCK", "UNLOCK", "DELETE"}
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
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/download/")
	permission := GetPermission(path, jwttoken, c)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}

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

func GetFilesByPaths(paths []string, c *gin.Context) ([]Files, error) {
	var data []Files
	data = nil
	for _, p := range paths {
		fi, err := fs.FileSystem.Stat(c.Request.Context(), p)
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
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	if param == "" || param == "/" {
		paths := ListCurrentProjects(jwttoken.UserID, jwttoken.QueueID, c)
		fmt.Println(paths)
		data, err = GetFilesByPaths(paths, c)
		if err != nil {
			response.Error(c, "no project or porject has no dir", response.NotSpecified)
			return
		}
		response.Success(c, data)
	} else {
		data, err = handleDirsList(fs.FileSystem, param)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

func GetAllFiles(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		c.String(http.StatusUnauthorized, "Unauthorized 3")
		return
	}
	var queueFlag int
	var param string
	if strings.HasPrefix(c.Request.URL.Path, "/api/ss/admin/queue") {
		queueFlag = 1
		param = strings.TrimPrefix(c.Request.URL.Path, "/api/ss/admin/queue")
	} else if strings.HasPrefix(c.Request.URL.Path, "/api/ss/admin/user") {
		queueFlag = 2
		param = strings.TrimPrefix(c.Request.URL.Path, "/api/ss/admin/user")
	} else {
		response.BadRequestError(c, "error url")
		return
	}

	if param == "" || param == "/" {
		var paths []string
		if queueFlag == 1 {
			paths = ListQueueProjects(c)
		} else if queueFlag == 2 {
			paths = ListUserProjects(c)
		} else {
			response.BadRequestError(c, "error url")
			return
		}
		data, err = GetFilesByPaths(paths, c)
		if err != nil {
			response.Error(c, "no project or porject has no dir", response.NotSpecified)
			return
		}
		response.Success(c, data)
	} else {
		data, err = handleDirsList(fs.FileSystem, param)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

func handleDirsList(fs webdav.FileSystem, path string) ([]Files, error) {
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
		logutils.Log.Info("Error reading directory")
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
		return
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

func GetDirSize(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/dirsize")
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	if param == "" || param == "/" {
		response.BadRequestError(c, "Can't get size of all dirs")
		return
	}
	size, err := getDirSize("/crater" + param)
	if err != nil {
		response.Error(c, "Can't Get dirsize", response.NotSpecified)
		return
	}
	response.Success(c, size)
}

func getDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return size, nil
}

func checkSpace() {
	u := query.User
	q := query.Queue
	ctx := context.Background()
	user, err := u.WithContext(ctx).Where(u.ID.IsNotNull()).Find()
	if err != nil {
		fmt.Println("can't get user")
		return
	}
	queue, err := q.WithContext(ctx).Where(q.ID.IsNotNull()).Find()
	if err != nil {
		fmt.Println("can't get queue")
		return
	}
	for _, us := range user {
		_, err := fs.FileSystem.Stat(ctx, us.Space)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", us.Space)
				err = fs.FileSystem.Mkdir(ctx, us.Space, os.FileMode(defaultFolderPerm))
				if err != nil {
					fmt.Println("can't create dir:", us.Space)
					return
				}
			}
		}
	}
	for _, que := range queue {
		_, err := fs.FileSystem.Stat(ctx, que.Space)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", que.Space)
				err = fs.FileSystem.Mkdir(ctx, que.Space, os.FileMode(defaultFolderPerm))
				if err != nil {
					fmt.Println("can't create dir:", que.Space)
					return
				}
			}
		}
	}
}

func DeleteFile(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/delete/")
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed || permission == model.ReadOnly {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	path := strings.TrimLeft(param, "/")
	err = fs.FileSystem.RemoveAll(c, path)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	response.Success(c, "Delete file successfully ")
}

func GetPermission(path string, token util.JWTMessage, c *gin.Context) model.FilePermission {
	path = strings.TrimLeft(path, "/")
	cleanedPath := filepath.Clean(path)
	part := strings.Split(cleanedPath, "/")
	if token.RolePlatform == model.RoleAdmin {
		return model.ReadWrite
	}
	if strings.HasPrefix(part[0], "u-") {
		err := CheckUser(token.UserID, part[0], c)
		if err != nil {
			return model.NotAllowed
		}
		return model.ReadWrite
	} else if strings.HasPrefix(part[0], "q-") {
		return model.FilePermission(token.AccessMode)
	} else if strings.HasPrefix(part[0], "public") {
		return model.FilePermission(token.PublicAccessMode)
	} else {
		return model.NotAllowed
	}
}

func CheckUser(userid uint, space string, c *gin.Context) error {
	u := query.User
	_, err := u.WithContext(c).Where(u.ID.Eq(userid), u.Space.Eq(space)).First()
	return err
}

const defaultTime = 120

func StartCheckSpace() {
	checkfs()
	for {
		checkSpace()
		time.Sleep(time.Second * defaultTime)
	}
}

type UserSpaceResp struct {
	Username string `json:"username"`
	Space    string `json:"space"`
}
type QueueSpaceResp struct {
	Queuename string `json:"queuename"`
	Space     string `json:"space"`
}

func GetUserSpace(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		response.Error(c, "can't get user", response.InvalidRole)
		return
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.IsNotNull()).Find()
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	var userSpaceResp []UserSpaceResp
	for i := range user {
		var userspace UserSpaceResp
		userspace.Space = user[i].Space
		userspace.Username = user[i].Name
		userSpaceResp = append(userSpaceResp, userspace)
	}
	response.Success(c, userSpaceResp)
}

func GetQueueSpace(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	q := query.Queue
	queue, err := q.WithContext(c).Where(q.ID.IsNotNull()).Find()
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		response.Error(c, "has no permission to get queue", response.InvalidRole)
		return
	}
	var queueSpaceResp []QueueSpaceResp
	for i := range queue {
		var queuespace QueueSpaceResp
		queuespace.Queuename = queue[i].Name
		queuespace.Space = queue[i].Space
		queueSpaceResp = append(queueSpaceResp, queuespace)
	}
	response.Success(c, queueSpaceResp)
}

func RegisterFile(r *gin.Engine) {
	r.Handle("OPTIONS", "/api/ss", AlloweOption)
	r.Handle("OPTIONS", "/api/ss/*path", AlloweOption)
	r.Handle("GET", "/api/ss/files", GetFiles)
	r.Handle("GET", "/api/ss/files/*path", GetFiles)
	r.Handle("GET", "/api/ss/admin/*path", GetAllFiles)
	r.Handle("GET", "/api/ss/download/*path", Download)
	r.Handle("POST", "/api/ss/checkspace", CheckFilesExist)
	r.Handle("GET", "/api/ss/dirsize/*path", GetDirSize)
	r.Handle("DELETE", "/api/ss/delete/*path", DeleteFile)
	r.Handle("GET", "/api/ss/userspace", GetUserSpace)
	r.Handle("GET", "/api/ss/queuespace", GetQueueSpace)
}
