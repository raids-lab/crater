package service

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"webdav/logutils"
	"webdav/model"
	"webdav/orm"

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

func ListMySharedProject(userID uint) []string {
	db := orm.DB()
	var UserPro []model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ?", userID).Find(&UserPro).Error
	if err != nil {
		fmt.Println("user has no project, ", err)
		return nil
	}
	var projetcname []string
	projetcname = nil
	for i := range UserPro {
		var project model.Project
		err := db.Model(&model.Project{}).Where("id = ? AND is_personal = ?", UserPro[i].ProjectID, false).Find(&project).Error
		if err == nil && project.ID != 0 {
			projetcname = append(projetcname, project.Name)
		}
	}
	return projetcname
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
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  jwttoken.Msg,
		})
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

func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func GetSharedProjectDir(c *gin.Context) {
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  jwttoken.Msg,
		})
		return
	}
	myproject := ListMySharedProject(jwttoken.Data.UserID)
	if myproject != nil {
		c.JSON(http.StatusOK, gin.H{
			"data": myproject,
			"msg":  "",
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"data": myproject,
			"msg":  "no shared project",
		})
	}
}

func GetMyDir(c *gin.Context) {
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  jwttoken.Msg,
		})
		return
	}
	path := jwttoken.Data.RootPath
	data, err := handleDirList(fs.FileSystem, c.Writer, path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"msg":  "",
	})
}

func GetFile(c *gin.Context) {
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil || jwttoken.Code != 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  jwttoken.Msg,
		})
		return
	}
	rootpath := jwttoken.Data.RootPath
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/file")
	path := rootpath + param
	if jwttoken.Data.Permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	data, err := handleDirList(fs.FileSystem, c.Writer, path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err,
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"msg":  "",
	})
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
		logutils.Log.Info("cann't read a empty dir")
		return nil, nil
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

func Testtoken(c *gin.Context) {
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": jwttoken,
			"msg":  err.Error(),
		})
	}
	fmt.Println(c.Request.URL.Path)
	fmt.Println(jwttoken)
}
