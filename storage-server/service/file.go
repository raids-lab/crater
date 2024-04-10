package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"webdav/model"
	"webdav/orm"

	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/webdav"
)

var fs *webdav.Handler
var fsLock sync.Mutex

type Filereq struct {
	Userid    *int   `json:"userid" binding:"required"`
	Projectid *int   `json:"projectid"`
	Path      string `json:"path" `
}

type File struct {
	Name  string `json:"name"`
	IsDir bool   `json:"isdir"`
}

func checkfs() {
	if fs == nil {
		fsLock.Lock()
		if fs == nil {
			fs = &webdav.Handler{
				Prefix:     "/files",
				FileSystem: webdav.Dir("/crater"),
				LockSystem: webdav.NewMemLS(),
				Logger: func(h *http.Request, e error) {
					if e != nil {
						log.Error(e)
					}
				},
			}
		}
	}
}

func CheckFilePermission(user_id uint, project_id uint) model.FilePermission {
	// pathPart := strings.FieldsFunc(path, func(s rune) bool { return s == '/' })
	// realpath := strings.TrimPrefix(path, "/files")
	// if len(pathPart) <= 1 || pathPart[0] != "files" {
	// 	fmt.Println("CheckFilePermission path too short ", path)
	// 	return NotAllowed
	// }
	db := orm.DB()
	var UserPro model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ? AND project_id = ?", user_id, project_id).First(&UserPro).Error
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

func ListMyProject(user_id int) []string {
	db := orm.DB()
	var UserPro []model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ?", user_id).Find(&UserPro).Error
	if err != nil {
		fmt.Println("user has no project, ", err)
		return nil
	}
	var projetcname []string
	projetcname = nil
	for _, up := range UserPro {
		var project model.Project
		err := db.Model(&model.Project{}).Where("id = ? AND is_personal = ?", up.ProjectID, false).Find(&project).Error
		if err == nil && project.ID != 0 {
			projetcname = append(projetcname, project.Name)
		}
	}
	return projetcname
}

func GetMyProject(user_id int) model.Project {
	db := orm.DB()
	var UserPro []model.UserProject
	err := db.Model(&model.UserProject{}).Where("user_id = ?", user_id).Find(&UserPro).Error
	if err != nil || UserPro[0].ID == 0 {
		fmt.Println("user has no project, ", err)
		return model.Project{}
	}
	for _, up := range UserPro {
		var project model.Project
		err := db.Model(&model.Project{}).Where("id = ? AND is_personal = ?", up.ProjectID, true).First(&project).Error
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

	proj_id, _ := strconv.Atoi(c.Query("projectid"))
	user_id, _ := strconv.Atoi(c.Query("userid"))

	permission := CheckFilePermission(uint(user_id), uint(proj_id))
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "MOVE", "LOCK", "UNLOCK"}
	if permission == model.ReadOnly && containsString(rwMethods, c.Request.Method) {
		c.String(http.StatusUnauthorized, "Unauthorized 2")
		return
	}
	http.StripPrefix("/files", fs)
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
	var req Filereq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err.Error(),
		})
		return
	}
	myproject := ListMyProject(*req.Userid)
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
	var req Filereq
	if err := c.ShouldBind(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err.Error(),
		})
		return
	}
	myproject := GetMyProject(*req.Userid)
	if myproject.ID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  "user has no personal project",
		})
		return
	}
	rootpath := GetSpaceByProjectID(myproject.ID)
	path := rootpath + req.Path
	data, err := handleDirList(fs.FileSystem, c.Writer, c.Request, true, path)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err,
		})
		return
	}
	// v, _ := json.Marshal(data)
	// jsonStr := string(v) // 结构体转为json对象
	c.JSON(http.StatusOK, gin.H{
		"data": data,
		"msg":  "",
	})
}

func GetFile(c *gin.Context) {
	checkfs()
	var req Filereq
	if err := c.ShouldBind(&req); err != nil || *req.Projectid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"data": nil,
			"msg":  err.Error(),
		})
		return
	}
	rootpath := GetSpaceByProjectID(uint(*req.Projectid))
	path := rootpath + req.Path
	permission := CheckFilePermission(uint(*req.Userid), uint(*req.Projectid))
	if permission == model.NotAllowed {
		c.String(http.StatusUnauthorized, "Unauthorized 1")
		return
	}
	data, err := handleDirList(fs.FileSystem, c.Writer, c.Request, false, path)
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

func handleDirList(fs webdav.FileSystem, w http.ResponseWriter, req *http.Request, ispersonal bool, path string) ([]File, error) {
	ctx := context.Background()
	// if ispersonal {
	// 	req.URL.Path = strings.TrimPrefix(req.URL.Path, "/files/mydir")
	// 	req.URL.Path = path + req.URL.Path
	// } else {
	// 	req.URL.Path = strings.TrimPrefix(req.URL.Path, "/files/sharedfile")
	// 	req.URL.Path = path + req.URL.Path
	// }
	f, err := fs.OpenFile(ctx, path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	var files []File
	defer f.Close()
	if fi, _ := f.Stat(); fi != nil && !fi.IsDir() {
		log.Print("cann't read a empty dir")
		return nil, nil
	}
	dirs, err := f.Readdir(-1)
	if err != nil {
		log.Print(w, "Error reading directory", http.StatusInternalServerError)
		return nil, err
	}
	// w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// fmt.Fprintf(w, "<pre>\n")
	var tmp File
	for _, d := range dirs {
		tmp.Name = d.Name()

		if d.IsDir() {
			tmp.IsDir = true
		} else {
			tmp.IsDir = false
		}

		files = append(files, tmp)

		// fmt.Fprintf(w, "<a href=\"%s\">%s</a>\n", tmp.name, tmp.name)
	}
	// fmt.Fprintf(w, "</pre>\n")
	return files, nil
}

func Testtoken(c *gin.Context) {
	userID, ok := c.Get("x-user-id")
	if !ok {
		c.JSON(400, gin.H{
			"msg": "user id not found",
		})
		return
	}

	projectID, ok := c.Get("x-project-id")
	if !ok {
		c.JSON(400, gin.H{
			"msg": "project id not found",
		})
		return
	}
	c.JSON(200, gin.H{
		"userid":    userID,
		"projectid": projectID,
	})
}
