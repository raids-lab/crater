package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
	"webdav/config"
	"webdav/dao/model"
	"webdav/dao/query"
	"webdav/logutils"
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

func checkfs() {
	fsonce.Do(func() {
		fs = &webdav.Handler{
			Prefix:     "/api/ss",
			FileSystem: webdav.Dir("/crater"),
			LockSystem: webdav.NewMemLS(),
		}
	})
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
	} else if token.AccountID == util.QueueIDNull {
		return model.FilePermission(token.PublicAccessMode)
	} else {
		return model.FilePermission(token.AccountAccessMode)
	}
}

// 是列出用户当前账户、公共账户和自己用户空间的地址和实际地址
// 目前用户的space没有前缀/，账户的有前缀/
func ListMySpace(userID, accountID uint, c *gin.Context) (allspace, allRealSpace []string) {
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if err != nil {
		fmt.Println("can't find user")
		return nil, nil
	}
	var space, realSpace []string
	if user.Space != "" {
		space = append(space, user.Space)
		realSpace = append(realSpace, config.GetConfig().UserSpacePrefix+"/"+user.Space)
	}
	a := query.Account
	publicaccount, err := a.WithContext(c).Where(a.ID.Eq(model.DefaultAccountID)).First()
	if err != nil {
		fmt.Println("can't find public account, ", err)
		return space, realSpace
	}
	space = append(space, strings.TrimLeft(publicaccount.Space, "/"))
	realSpace = append(realSpace, config.GetConfig().PublicSpacePrefix)
	if accountID != 0 && accountID != model.DefaultAccountID {
		account, err := a.WithContext(c).Where(a.ID.Eq(accountID)).First()
		if err != nil {
			fmt.Println("user has no account, ", err)
			return space, realSpace
		}
		space = append(space, strings.TrimLeft(account.Space, "/"))
		realSpace = append(realSpace, config.GetConfig().AccountSpacePrefix+account.Space)
	}
	space = append(space, config.GetConfig().PublicSpacePrefix)
	return space, realSpace
}

// 获取所有账户空间实际地址
func ListAllAccountSpaces(c *gin.Context) []string {
	accountSpacePrefix := config.GetConfig().AccountSpacePrefix
	var data []string
	a := query.Account
	accounts, err := a.WithContext(c).Where(a.ID.IsNotNull()).Find()
	if err != nil || len(accounts) == 0 {
		fmt.Println("can't find account, ", err)
		return data
	}
	for i := range accounts {
		if accounts[i].Space != "" {
			if strings.HasPrefix(accounts[i].Space, "/") {
				data = append(data, accountSpacePrefix+accounts[i].Space)
			} else {
				data = append(data, accountSpacePrefix+"/"+accounts[i].Space)
			}
		}
	}
	return data
}

// 获取所有用户空间实际地址
func ListAllUserSpaces(c *gin.Context) []string {
	userSpacePrefix := config.GetConfig().UserSpacePrefix
	var data []string
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.IsNotNull()).Find()
	if len(user) == 0 || err != nil {
		fmt.Println("can't find user,err: ", err)
		return data
	}
	for j := range user {
		if user[j].Space != "" {
			if strings.HasPrefix(user[j].Space, "/") {
				data = append(data, userSpacePrefix+user[j].Space)
			} else {
				data = append(data, userSpacePrefix+"/"+user[j].Space)
			}
		}
	}
	return data
}

func WebDav(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss")
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		response.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", response.InvalidRole)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "DELETE"}
	if permission == model.ReadOnly && containsString(rwMethods, c.Request.Method) {
		response.HTTPError(c, http.StatusUnauthorized, "You have no permission to do this", response.NotSpecified)
		return
	}
	http.StripPrefix("/api/ss", fs)
	c.Request.URL.Path = "/api/ss/" + realPath
	fs.ServeHTTP(c.Writer, c.Request)
	// 直接创建文件夹使用777权限也没有用，可能是因为父目录有设置SetGID位，权限是drwxr-sr-x，于是选择直接修改权限
	if c.Request.Method == "MKCOL" || c.Request.Method == "PUT" {
		chmod(c, model.RWXFolderPerm)
	}
}

func chmod(c *gin.Context, mode os.FileMode) {
	reqPath := strings.TrimPrefix(c.Request.URL.Path, fs.Prefix)
	if fs.Prefix != "" && len(reqPath) == len(c.Request.URL.Path) {
		response.Error(c, "prefix mismatch error", response.InvalidRequest)
		return
	}
	var realPath string
	dir, _ := fs.FileSystem.(webdav.Dir)
	realPath = string(dir) + reqPath
	if err := os.Chmod(realPath, mode); err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
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
		response.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", response.NotSpecified)
		return
	}
	realPath, err := Redirect(c, path, jwttoken)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	f, err := fs.FileSystem.OpenFile(c.Request.Context(), realPath, os.O_RDWR, 0)
	if err != nil {
		fmt.Println("err:", err)
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

func GetBasicFiles(c *gin.Context, token util.JWTMessage, isadmin bool) []Files {
	userSpacePrefix := config.GetConfig().UserSpacePrefix
	accountSpacePrefix := config.GetConfig().AccountSpacePrefix
	publicSpacePrefix := config.GetConfig().PublicSpacePrefix
	var s []string
	s = append(s, userSpacePrefix, publicSpacePrefix)
	if isadmin || (token.AccountID != 0 && token.AccountID != model.DefaultAccountID) {
		s = append(s, accountSpacePrefix)
	}
	files := GetFilesByPaths(s, c)
	for i, f := range files {
		if f.Name == userSpacePrefix {
			files[i].Name = model.UserPath
		} else if f.Name == publicSpacePrefix {
			files[i].Name = model.PublicPath
		} else if f.Name == accountSpacePrefix {
			files[i].Name = model.AccountPath
		}
	}
	return files
}

func GetRWFiles(c *gin.Context, token util.JWTMessage) []Files {
	userSpacePrefix := config.GetConfig().UserSpacePrefix
	accountSpacePrefix := config.GetConfig().AccountSpacePrefix
	publicSpacePrefix := config.GetConfig().PublicSpacePrefix
	var s []string
	s = append(s, userSpacePrefix)
	if token.PublicAccessMode == model.AccessModeRW {
		s = append(s, publicSpacePrefix)
	}
	if token.AccountID != 0 && token.AccountID != model.DefaultAccountID && token.AccountAccessMode == model.AccessModeRW {
		s = append(s, accountSpacePrefix)
	}
	files := GetFilesByPaths(s, c)
	for i, f := range files {
		if f.Name == userSpacePrefix {
			files[i].Name = model.UserPath
		} else if f.Name == publicSpacePrefix {
			files[i].Name = model.PublicPath
		} else if f.Name == accountSpacePrefix {
			files[i].Name = model.AccountPath
		}
	}
	return files
}

func GetFilesByPaths(paths []string, c *gin.Context) []Files {
	var data []Files
	data = nil
	for _, p := range paths {
		fi, err := fs.FileSystem.Stat(c.Request.Context(), p)
		if err == nil {
			var tmp Files
			tmp.IsDir = fi.IsDir()
			tmp.ModifyTime = fi.ModTime()
			tmp.Name = fi.Name()
			tmp.Size = fi.Size()
			tmp.Sys = fi.Sys()
			data = append(data, tmp)
		}
	}
	return data
}

// 用户获取文件
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
	token := getFirstToken(param)
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		response.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", response.NotSpecified)
		return
	}
	if token == "" {
		data = GetBasicFiles(c, jwttoken, false)
		response.Success(c, data)
	} else {
		realPath, err := Redirect(c, param, jwttoken)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

// 用户获取有读写权限的可移动文件
func GetFilesWithRWAcc(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/rwfiles")
	token := getFirstToken(param)
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed || (permission == model.ReadOnly && token != "") {
		response.HTTPError(c, http.StatusUnauthorized, "You have no permission to get these files", response.NotSpecified)
		return
	}
	if token == "" {
		data = GetRWFiles(c, jwttoken)
		response.Success(c, data)
	} else {
		realPath, err := Redirect(c, param, jwttoken)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

// 去除前面的/后，使用/作为分隔符的第一个token
func getFirstToken(path string) string {
	path = strings.TrimLeft(path, "/")
	cleanedPath := filepath.Clean(path)
	tokens := strings.Split(cleanedPath, "/")
	if len(tokens) > 0 && tokens[0] != "." {
		return tokens[0]
	}
	return ""
}

// admin获取文件
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
		response.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", response.NotSpecified)
		return
	}
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/admin/files")
	token := getFirstToken(path)
	if token == "" {
		data = GetBasicFiles(c, jwttoken, true)
		response.Success(c, data)
	} else {
		realPath, err := Redirect(c, path, jwttoken)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			response.Error(c, err.Error(), response.NotSpecified)
			return
		}
		response.Success(c, data)
	}
}

type DatasetRequest struct {
	ID uint `uri:"id" binding:"required"`
}

func GetDatasetPermission(c *gin.Context, datasetID uint, token util.JWTMessage) model.FilePermission {
	ud := query.UserDataset
	d := query.Dataset
	ad := query.AccountDataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetID)).First()
	if err != nil {
		return model.NotAllowed
	}
	if dataset.UserID == token.UserID || token.RolePlatform == model.RoleAdmin {
		return model.ReadWrite
	}
	accountDataset, err := ad.WithContext(c).Where(ad.DatasetID.Eq(datasetID)).Find()
	if err == nil && len(accountDataset) != 0 {
		for i := range accountDataset {
			if accountDataset[i].AccountID == model.DefaultAccountID || accountDataset[i].AccountID == token.AccountID {
				return model.ReadOnly
			}
		}
	}
	_, err = ud.WithContext(c).Where(ud.DatasetID.Eq(datasetID), ud.UserID.Eq(token.UserID)).First()
	if err == nil {
		return model.ReadOnly
	}
	return model.NotAllowed
}

func GetDatasetURLByID(c *gin.Context, datasetID uint) (string, error) {
	d := query.Dataset
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetID)).First()
	if err != nil {
		return "", err
	}
	return dataset.URL, err
}

// 通过数据集获取文件
func GetDatasetFiles(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	var data []Files
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
	permission := GetDatasetPermission(c, datasetReq.ID, jwttoken)
	if permission == model.NotAllowed {
		response.Error(c, "This dataset does not exist or you do not have permission", response.NotSpecified)
		return
	}
	URL, err := GetDatasetURLByID(c, datasetReq.ID)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	ss := "/api/ss/dataset/" + strconv.FormatUint(uint64(datasetReq.ID), 10)
	path := strings.TrimPrefix(c.Request.URL.Path, ss)
	token := getFirstToken(path)
	if token == "" {
		var datasetpaths []string
		datasetpaths = append(datasetpaths, URL)
		files := GetFilesByPaths(datasetpaths, c)
		if len(files) == 0 {
			response.Error(c, "The dataset's URL does not exist. ", response.NotSpecified)
			return
		}
		response.Success(c, files)
	} else {
		realPath := URL + "/" + strings.TrimPrefix(path, "/"+token)
		data, err = handleDirsList(fs.FileSystem, realPath)
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

func checkSpace() {
	ctx := context.Background()
	var baseSpace []string
	baseSpace = append(baseSpace, config.GetConfig().AccountSpacePrefix,
		config.GetConfig().UserSpacePrefix, config.GetConfig().PublicSpacePrefix)
	for _, space := range baseSpace {
		_, err := fs.FileSystem.Stat(ctx, space)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", space)
				err = fs.FileSystem.Mkdir(ctx, space, os.FileMode(model.RWXFolderPerm))
				if err != nil {
					fmt.Println("can't create dir:", space)
					fmt.Println("err:", err)
					return
				}
			}
		}
	}
	u := query.User
	a := query.Account
	user, err := u.WithContext(ctx).Where(u.ID.IsNotNull()).Find()
	if err != nil {
		fmt.Println("can't get user")
		return
	}
	account, err := a.WithContext(ctx).Where(a.ID.IsNotNull()).Find()
	if err != nil {
		fmt.Println("can't get account")
		return
	}
	for _, us := range user {
		space := config.GetConfig().UserSpacePrefix + "/" + us.Space
		_, err := fs.FileSystem.Stat(ctx, space)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", space)
				err = fs.FileSystem.Mkdir(ctx, space, os.FileMode(model.RWXFolderPerm))
				if err != nil {
					fmt.Println("can't create dir:", us.Space)
					fmt.Println("err:", err)
					return
				}
			}
		}
	}
	for _, acc := range account {
		space := config.GetConfig().AccountSpacePrefix + acc.Space
		_, err := fs.FileSystem.Stat(ctx, space)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				fmt.Println("create dir:", space)
				err = fs.FileSystem.Mkdir(ctx, space, os.FileMode(model.RWXFolderPerm))
				if err != nil {
					fmt.Println("can't create dir:", acc.Space)
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
	if permission != model.ReadWrite {
		response.HTTPError(c, http.StatusUnauthorized, "You have no permission to delete file", response.NotSpecified)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	err = fs.FileSystem.RemoveAll(c, realPath)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	response.Success(c, "Delete file successfully ")
}

// 获得用户权限，不判断是否为管理员
func GetPermission(path string, token util.JWTMessage, c *gin.Context) model.FilePermission {
	path = strings.TrimLeft(path, "/")
	cleanedPath := filepath.Clean(path)
	if path == "" {
		return model.ReadOnly
	}
	part := strings.Split(cleanedPath, "/")
	if part[0] == model.AccountPath {
		a := query.Account
		_, err := a.WithContext(c).Where(a.ID.Eq(token.AccountID)).First()
		if err != nil {
			return model.NotAllowed
		}
		return model.FilePermission(token.AccountAccessMode)
	} else if part[0] == model.PublicPath {
		return model.FilePermission(token.PublicAccessMode)
	} else if part[0] == model.UserPath {
		u := query.User
		_, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if err != nil {
			return model.NotAllowed
		}
		return model.ReadWrite
	} else if part[0] == model.AdminAccountPath || part[0] == model.AdminPublicPath || part[0] == model.AdminUserPath {
		if token.RolePlatform != model.RoleAdmin {
			return model.NotAllowed
		}
		return model.ReadWrite
	} else {
		return model.NotAllowed
	}
}

func CheckUser(userid uint, space string, c *gin.Context) error {
	u := query.User
	_, err := u.WithContext(c).Where(u.ID.Eq(userid), u.Space.Eq(space)).First()
	return err
}

func CheckAccount(accountID uint, space string, c *gin.Context) error {
	a := query.Account
	_, err := a.WithContext(c).Where(a.ID.Eq(accountID), a.Space.Eq(space)).First()
	return err
}

const defaultTime = 60

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
type AccountSpaceResp struct {
	Accountname string `json:"queuename"`
	Space       string `json:"space"`
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

func GetAccountSpace(c *gin.Context) {
	AlloweOption(c)
	checkfs()
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	a := query.Account
	account, err := a.WithContext(c).Where(a.ID.IsNotNull()).Find()
	if err != nil {
		response.Error(c, err.Error(), response.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		response.Error(c, "has no permission to get queue", response.InvalidRole)
		return
	}
	var accountSpaceResp []AccountSpaceResp
	for i := range account {
		var accountspace AccountSpaceResp
		accountspace.Accountname = account[i].Name
		accountspace.Space = account[i].Space
		accountSpaceResp = append(accountSpaceResp, accountspace)
	}
	response.Success(c, accountSpaceResp)
}

// 文件地址重定向，指向实际文件地址,public，user，account，admin-public，admin-user，admin-account对应六种不同地址，
// 普通用户使用的path是前三种，直接重定向到自己所在的文件地址，管理员对应后三种，要验证管理员权限。
func Redirect(c *gin.Context, path string, token util.JWTMessage) (string, error) {
	userSpacePrefix := config.GetConfig().UserSpacePrefix
	accountSpacePrefix := config.GetConfig().AccountSpacePrefix
	publicSpacePrefix := config.GetConfig().PublicSpacePrefix
	path = strings.TrimLeft(path, "/")
	var res string
	if strings.HasPrefix(path, model.PublicPath) {
		res = strings.TrimPrefix(path, model.PublicPath)
		res = publicSpacePrefix + res
	} else if strings.HasPrefix(path, model.UserPath) {
		res = strings.TrimPrefix(path, model.UserPath)
		u := query.User
		user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if err != nil {
			return "", fmt.Errorf("user does not exist")
		}
		res = userSpacePrefix + "/" + user.Space + res
	} else if strings.HasPrefix(path, model.AccountPath) {
		res = strings.TrimPrefix(path, model.AccountPath)
		a := query.Account
		account, err := a.WithContext(c).Where(a.ID.Eq(token.AccountID)).First()
		if err != nil {
			return "", fmt.Errorf("account does not exist")
		}
		res = accountSpacePrefix + account.Space + res
	} else if strings.HasPrefix(path, model.AdminPublicPath) {
		res = strings.TrimPrefix(path, model.AdminPublicPath)
		res = publicSpacePrefix + res
	} else if strings.HasPrefix(path, model.AdminUserPath) {
		res = strings.TrimPrefix(path, model.AdminUserPath)
		if token.RolePlatform != model.RoleAdmin {
			return "", fmt.Errorf("your role is not admin")
		}
		res = userSpacePrefix + res
	} else if strings.HasPrefix(path, model.AdminAccountPath) {
		res = strings.TrimPrefix(path, model.AdminAccountPath)
		res = accountSpacePrefix + res
	} else {
		return "", fmt.Errorf("an incorrect path")
	}
	cleanedPath := filepath.Clean(path)
	tokens := strings.Split(cleanedPath, "/")
	if len(tokens) > 0 && tokens[0] != "." {
		return res, nil
	}
	return "", fmt.Errorf("an illegal path")
}

func RegisterFile(r *gin.Engine) {
	r.Handle("OPTIONS", "/api/ss", AlloweOption)
	r.Handle("OPTIONS", "/api/ss/*path", AlloweOption)
	r.Handle("GET", "/api/ss/files", GetFiles)
	r.Handle("GET", "/api/ss/files/*path", GetFiles)
	r.Handle("GET", "/api/ss/rwfiles", GetFilesWithRWAcc)
	r.Handle("GET", "/api/ss/rwfiles/*path", GetFilesWithRWAcc)
	r.Handle("GET", "/api/ss/admin/*path", GetAllFiles)
	r.Handle("GET", "/api/ss/download/*path", Download)
	r.Handle("DELETE", "/api/ss/delete/*path", DeleteFile)
	r.Handle("GET", "/api/ss/userspace", GetUserSpace)
	r.Handle("GET", "/api/ss/queuespace", GetAccountSpace)
	r.Handle("GET", "/api/ss/dataset/:id", GetDatasetFiles)
	r.Handle("GET", "/api/ss/dataset/:id/*path", GetDatasetFiles)
}
