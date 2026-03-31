package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	urlpath "path"
	"strconv"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"

	"github.com/gin-gonic/gin"
	"golang.org/x/net/webdav"
)

var fs *webdav.Handler
var fsonce sync.Once
var storageRootDir = "/crater"

func SetRootDir(rootDir string) {
	if rootDir == "" {
		storageRootDir = "/crater"
		return
	}
	storageRootDir = rootDir
}

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
			FileSystem: webdav.Dir(storageRootDir),
			LockSystem: webdav.NewMemLS(),
		}
		klog.Infof("Storage root directory: %s", storageRootDir)
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

func WebDAVMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		AlloweOption(c)
		checkfs()
		c.Next()
	}
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

// 闂佸搫瀚烽崹浼村垂椤忓牆绀勯柧蹇氼潐閺嗗繘鏌熼弶鎴濇Щ缂傚秴顑夊畷婊冾吋閸繍妲遍梺鐟扮摠閻忔岸鍩€椤戣法顦﹂柛娆愭礋瀹曟娼忛銏╂П闂佽娼欓崲鏌ュ箯娴煎瓨鍤婃い蹇撳缁犳帡鏌ｉ～顒€濡介柛鈺傜〒缁艾煤椤忓拑绱甸梺姹囧妼鐎氼剙锕㈤幘顔奸敜闁逞屽墴瀹曨亞浠﹂悾灞炬緰闂傚倸瀚幊搴★耿閹绢喖閿ら柍?
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
		realSpace = append(realSpace, config.GetConfig().Storage.Prefix.User+"/"+user.Space)
	}
	a := query.Account
	publicaccount, err := a.WithContext(c).Where(a.ID.Eq(model.DefaultAccountID)).First()
	if err != nil {
		fmt.Println("can't find public account, ", err)
		return space, realSpace
	}
	space = append(space, strings.TrimLeft(publicaccount.Space, "/"))
	realSpace = append(realSpace, config.GetConfig().Storage.Prefix.Public)
	if accountID != 0 && accountID != model.DefaultAccountID {
		account, err := a.WithContext(c).Where(a.ID.Eq(accountID)).First()
		if err != nil {
			fmt.Println("user has no account, ", err)
			return space, realSpace
		}
		space = append(space, strings.TrimLeft(account.Space, "/"))
		realSpace = append(realSpace, config.GetConfig().Storage.Prefix.Account+"/"+account.Space)
	}
	space = append(space, config.GetConfig().Storage.Prefix.Public)
	return space, realSpace
}

// 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绠ラ柍褜鍓熷鍨緞鎼粹槅妲遍梺瑙勬尦椤ユ捇鍩為弽顓熲拻闁哄鍨归弶浠嬫⒒閸曨偅鍣规繝鈧幘顔奸敜闁?
func ListAllAccountSpaces(c *gin.Context) []string {
	accountSpacePrefix := config.GetConfig().Storage.Prefix.Account
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

// 闂佸吋鍎抽崲鑼躲亹閸ヮ剙绠ラ柍褜鍓熷鍨緞鐏炵偓娈㈤梺瑙勬尦椤ユ捇鍩為弽顓熲拻闁哄鍨归弶浠嬫⒒閸曨偅鍣规繝鈧幘顔奸敜闁?
func ListAllUserSpaces(c *gin.Context) []string {
	userSpacePrefix := config.GetConfig().Storage.Prefix.User
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
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss")
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", resputil.UserNotAllowed)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	rwMethods := []string{"PROPPATCH", "MKCOL", "PUT", "DELETE"}
	if permission == model.ReadOnly && containsString(rwMethods, c.Request.Method) {
		resputil.HTTPError(c, http.StatusUnauthorized, "You have no permission to do this", resputil.NotSpecified)
		return
	}
	http.StripPrefix("/api/ss", fs)
	c.Request.URL.Path = "/api/ss/" + realPath
	fs.ServeHTTP(c.Writer, c.Request)
	// 闂佺儵鏅涢悺銊ф暜閹绢喖绀嗘繛鎴烆焽缁憋箓鏌￠崒姘煑婵炲棎鍨哄鍕焻濞戞ǚ鏋忛梺?77闂佸搫顦崯鏉戭瀶閻戞鈻曢柣鏂挎憸濮婇箖鏌￠崼婵愭Ш闁轰降鍊濋弫宥囦沪閽樺顔夐梺鐓庣枃婵倕危閹间礁鐐婇柣妯跨簿缁€瀣煟閺嵮冾暢婵炲弶濯介妵鎰板即閻斿摜鐣抽柣鐘辩婢т粙鎮块崫鐖€tGID婵炶揪绲界粙鍕濠靛绾ч柛鎰靛枟椤庢瑩鏌￠崟顓燁棤rwxr-sr-x闂佹寧绋戞總鏃傝姳椤掑嫬鍙婃い鏍亹閸嬫挻寰勭€ｎ亶浠撮梺鐑╂櫅閻°劎鏁€涙鈹嶆い鏃囧Г閺嗩參鏌℃径濠傛殻婵?
	if c.Request.Method == "MKCOL" || c.Request.Method == "PUT" {
		chmod(c, model.RWXFolderPerm)
	}
}

func chmod(c *gin.Context, mode os.FileMode) {
	reqPath := strings.TrimPrefix(c.Request.URL.Path, fs.Prefix)
	if fs.Prefix != "" && len(reqPath) == len(c.Request.URL.Path) {
		resputil.Error(c, "prefix mismatch error", resputil.InvalidRequest)
		return
	}
	var realPath string
	dir, _ := fs.FileSystem.(webdav.Dir)
	realPath = string(dir) + reqPath
	if err := os.Chmod(realPath, mode); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
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
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/download/")
	permission := GetPermission(path, jwttoken, c)
	if permission == model.NotAllowed {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", resputil.NotSpecified)
		return
	}
	realPath, err := Redirect(c, path, jwttoken)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	f, err := fs.FileSystem.OpenFile(c.Request.Context(), realPath, os.O_RDWR, 0)
	if err != nil {
		fmt.Println("err:", err)
		resputil.BadRequestError(c, "can't find file")
		return
	}
	defer f.Close()
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%q\"", c.Request.URL.Path))
	_, err = io.Copy(c.Writer, f)
	if err != nil {
		resputil.Error(c, "can't download file", resputil.NotSpecified)
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

func cleanURLPath(raw string) string {
	// URL paths must always use "/" semantics across all OS.
	normalized := strings.ReplaceAll(raw, "\\", "/")
	trimmed := strings.TrimLeft(normalized, "/")
	cleaned := urlpath.Clean(trimmed)
	if cleaned == "." {
		return ""
	}
	return cleaned
}

func splitURLPath(raw string) []string {
	cleaned := cleanURLPath(raw)
	if cleaned == "" {
		return nil
	}
	return strings.Split(cleaned, "/")
}
func GetBasicFiles(c *gin.Context, token util.JWTMessage, isadmin bool) []Files {
	userSpacePrefix := config.GetConfig().Storage.Prefix.User
	accountSpacePrefix := config.GetConfig().Storage.Prefix.Account
	publicSpacePrefix := config.GetConfig().Storage.Prefix.Public
	var s []string
	s = append(s, userSpacePrefix, publicSpacePrefix)
	if isadmin || (token.AccountID != 0 && token.AccountID != model.DefaultAccountID) {
		s = append(s, accountSpacePrefix)
	}
	files := GetFilesByPaths(s, c)
	for i, f := range files {
		switch f.Name {
		case userSpacePrefix:
			files[i].Name = model.UserPath
		case publicSpacePrefix:
			files[i].Name = model.PublicPath
		case accountSpacePrefix:
			files[i].Name = model.AccountPath
		}
	}
	return files
}

func GetRWFiles(c *gin.Context, token util.JWTMessage) []Files {
	userSpacePrefix := config.GetConfig().Storage.Prefix.User
	accountSpacePrefix := config.GetConfig().Storage.Prefix.Account
	publicSpacePrefix := config.GetConfig().Storage.Prefix.Public
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
		switch f.Name {
		case userSpacePrefix:
			files[i].Name = model.UserPath
		case publicSpacePrefix:
			files[i].Name = model.PublicPath
		case accountSpacePrefix:
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

// 闂佹椿娼块崝宥夊春濞戙垺鍤旂€瑰嫭婢樼徊鍧楁煛閸屾碍鐭楁繛?
func GetFiles(c *gin.Context) {
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/files")
	token := getFirstToken(param)
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your permission is notAllowed", resputil.NotSpecified)
		return
	}
	if token == "" {
		data = GetBasicFiles(c, jwttoken, false)
		resputil.Success(c, data)
	} else {
		realPath, err := Redirect(c, param, jwttoken)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		resputil.Success(c, data)
	}
}

// 闂佹椿娼块崝宥夊春濞戙垺鍤旂€瑰嫭婢樼徊鍧楁煛閸繍妲兼い鏇氬嵆瀹曟ê鈻庨幇顓犵К闂傚倸瀚崝妤€鈻撻幋锕€鐭楁い鏍ㄧ敖閳哄懎绀夐柕濠忛檮閻庮喖霉?
func GetFilesWithRWAcc(c *gin.Context) {
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/rwfiles")
	token := getFirstToken(param)
	permission := GetPermission(param, jwttoken, c)
	if permission == model.NotAllowed || (permission == model.ReadOnly && token != "") {
		resputil.HTTPError(c, http.StatusUnauthorized, "You have no permission to get these files", resputil.NotSpecified)
		return
	}
	if token == "" {
		data = GetRWFiles(c, jwttoken)
		resputil.Success(c, data)
	} else {
		realPath, err := Redirect(c, param, jwttoken)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		resputil.Success(c, data)
	}
}

// 闂佸憡锚椤嘲鈻嶆惔銊ョ鐎广儱顦板銊╂煟?闂佸憡鑹鹃崙鐣屾濠靛洦濯撮悹鎭掑妽閺?婵炶揪绲剧划鍫㈡嫻閻旂厧绀嗛柛鈩冪⊕椤撳墽绱掑Δ瀣洭婵炲牊鍨圭划顓㈩敄鐠侯煈浼囨繛鎴炴尵鐏忕灒ken
func getFirstToken(path string) string {
	tokens := splitURLPath(path)
	if len(tokens) > 0 {
		return tokens[0]
	}
	return ""
}

// admin闂佸吋鍎抽崲鑼躲亹閸ヮ剙妫橀柛銉檮椤?
func GetAllFiles(c *gin.Context) {
	var data []Files
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		resputil.HTTPError(c, http.StatusUnauthorized, "Your RolePlatform is not RoleAdmin", resputil.NotSpecified)
		return
	}
	path := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/admin/files")
	token := getFirstToken(path)
	if token == "" {
		data = GetBasicFiles(c, jwttoken, true)
		resputil.Success(c, data)
	} else {
		realPath, err := Redirect(c, path, jwttoken)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		resputil.Success(c, data)
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

// 闂備緡鍋呮穱铏规崲閸愵喖鏋侀柣妤€鐗嗙粊锕傛⒒閸℃顥℃鐐茬箻瀹曪綁寮介妸锔锯偓顔济?
func GetDatasetFiles(c *gin.Context) {
	var data []Files
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
	permission := GetDatasetPermission(c, datasetReq.ID, jwttoken)
	if permission == model.NotAllowed {
		resputil.Error(c, "This dataset does not exist or you do not have permission", resputil.NotSpecified)
		return
	}
	URL, err := GetDatasetURLByID(c, datasetReq.ID)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
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
			resputil.Error(c, "The dataset's URL does not exist. ", resputil.NotSpecified)
			return
		}
		resputil.Success(c, files)
	} else {
		realPath := URL + "/" + strings.TrimPrefix(path, "/"+token)
		data, err = handleDirsList(fs.FileSystem, realPath)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
		resputil.Success(c, data)
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
		klog.Info("cann't read a empty file")
		return files, nil
	}
	dirs, err := f.Readdir(-1)
	if err != nil {
		klog.Info("Error reading directory")
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
	baseSpace = append(baseSpace, config.GetConfig().Storage.Prefix.Account,
		config.GetConfig().Storage.Prefix.User, config.GetConfig().Storage.Prefix.Public, model.DatasetPrefix, model.ModelPrefix)
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
		space := config.GetConfig().Storage.Prefix.User + "/" + us.Space
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
		space := config.GetConfig().Storage.Prefix.Account + "/" + acc.Space
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
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	param := strings.TrimPrefix(c.Request.URL.Path, "/api/ss/delete/")
	permission := GetPermission(param, jwttoken, c)
	if permission != model.ReadWrite {
		resputil.HTTPError(c, http.StatusUnauthorized, "You have no permission to delete file", resputil.NotSpecified)
		return
	}
	realPath, err := Redirect(c, param, jwttoken)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	err = fs.FileSystem.RemoveAll(c, realPath)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Delete file successfully ")
}

// 闂佸吋鍎抽崲鑼闁秵鍋ㄩ柕濠忕畱閻撴洟鏌℃径濠傛殻婵?
func GetPermission(path string, token util.JWTMessage, c *gin.Context) model.FilePermission {
	cleanedPath := cleanURLPath(path)
	if cleanedPath == "" {
		return model.ReadOnly
	}
	part := strings.Split(cleanedPath, "/")
	switch part[0] {
	case model.AccountPath:
		a := query.Account
		_, err := a.WithContext(c).Where(a.ID.Eq(token.AccountID)).First()
		if err != nil {
			return model.NotAllowed
		}
		return model.FilePermission(token.AccountAccessMode)
	case model.PublicPath:
		return model.FilePermission(token.PublicAccessMode)
	case model.UserPath:
		u := query.User
		_, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if err != nil {
			return model.NotAllowed
		}
		return model.ReadWrite
	case model.AdminAccountPath, model.AdminPublicPath, model.AdminUserPath:
		if token.RolePlatform != model.RoleAdmin {
			return model.NotAllowed
		}
		return model.ReadWrite
	default:
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
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		resputil.Error(c, "can't get user", resputil.UserNotAllowed)
		return
	}
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.IsNotNull()).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	var userSpaceResp []UserSpaceResp
	for i := range user {
		var userspace UserSpaceResp
		userspace.Space = user[i].Space
		userspace.Username = user[i].Name
		userSpaceResp = append(userSpaceResp, userspace)
	}
	resputil.Success(c, userSpaceResp)
}

func GetAccountSpace(c *gin.Context) {
	jwttoken, err := CheckJWTToken(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	a := query.Account
	account, err := a.WithContext(c).Where(a.ID.IsNotNull()).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	if jwttoken.RolePlatform != model.RoleAdmin {
		resputil.Error(c, "has no permission to get queue", resputil.UserNotAllowed)
		return
	}
	var accountSpaceResp []AccountSpaceResp
	for i := range account {
		var accountspace AccountSpaceResp
		accountspace.Accountname = account[i].Name
		accountspace.Space = account[i].Space
		accountSpaceResp = append(accountSpaceResp, accountspace)
	}
	resputil.Success(c, accountSpaceResp)
}

// 闂佸搫鍊稿ú锝呪枎閵忋倕鎹堕柡澶嬪缁插姊洪幓鎺斝㈤柣锝夌畺瀹曘儵骞嬮婊咁槷闂佸湱顭堝ú銈夊箖濠婂應鍋撻崷顓炰槐婵＄虎鍨跺顒勫炊閿旂瓔鍋ㄩ梺闈╅檮濠㈡ê顭?public闂佹寧绋戦惁鍧癳r闂佹寧绋戦悾鐢ount闂佹寧绋戦悾鐢in-public闂佹寧绋戦悾鐢in-user闂佹寧绋戦悾鐢in-account闁诲海鏁搁幊鎾惰姳閺屻儱绀傛い鎾跺濞兼艾鈽夐幘宕囆㈤柟顔芥崌瀹曠兘寮堕幐搴ｇシ闂?// 闂佸搫鎷嬮崳锝夊焵椤掍焦鐨戦柡浣靛€濋獮瀣憥閸屾埃鏋忛梺娲绘娇閸斿骸鈻撻幉鍒焧h闂佸搫瀚烽崹鐗堟櫠閻樺磭鈻斿璺虹焾濞兼岸鏌ㄥ☉妯肩伇婵炴彃娼￠獮鎺楀Ψ閳轰焦顏熼柣搴ゎ潐閼归箖骞冨鍫濈闁瑰嘲鑻▓鎵偓瑙勭摃妞村憡鏅跺澶婃嵍闁靛ě鍕殸闂佸搫鍊稿ú锝呪枎閵忋倕鎹堕柡澶嬪缁插鏌ㄥ☉妯肩伇妞ゆ挻鎮傞幃鍫曞幢濡や礁鏋€闁诲海鏁搁幊鎾惰姳閺屻儱瑙﹂幖瀛樼箘閻熷繒绱掓径濠勨枌缂佽鲸绻勯幉鐗堢瑹閳ь剟鎮板▎鎴炲珰濞达絼璀﹂崥鈧梺鑽ゅ仜濡骞夐幎钘夌骇闁告劦鍠楅娆撴煏?
func Redirect(c *gin.Context, path string, token util.JWTMessage) (string, error) {
	userSpacePrefix := config.GetConfig().Storage.Prefix.User
	accountSpacePrefix := config.GetConfig().Storage.Prefix.Account
	publicSpacePrefix := config.GetConfig().Storage.Prefix.Public
	path = cleanURLPath(path)
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
		res = accountSpacePrefix + "/" + account.Space + res
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
	tokens := splitURLPath(path)
	if len(tokens) > 0 {
		return res, nil
	}
	return "", fmt.Errorf("an illegal path")
}

func RegisterFile(webdavGroup *gin.RouterGroup) {
	webdavGroup.Handle("OPTIONS", "")
	webdavGroup.Handle("OPTIONS", "/*path")
	webdavGroup.GET("/files", GetFiles)
	webdavGroup.GET("/files/*path", GetFiles)
	webdavGroup.GET("/rwfiles", GetFilesWithRWAcc)
	webdavGroup.GET("/rwfiles/*path", GetFilesWithRWAcc)
	webdavGroup.GET("/admin/files", GetAllFiles)
	webdavGroup.GET("/admin/files/*path", GetAllFiles)
	webdavGroup.GET("/download/*path", Download)
	webdavGroup.DELETE("/delete/*path", DeleteFile)
	webdavGroup.GET("/userspace", GetUserSpace)
	webdavGroup.GET("/queuespace", GetAccountSpace)
	webdavGroup.GET("/dataset/:id", GetDatasetFiles)
	webdavGroup.GET("/dataset/:id/*path", GetDatasetFiles)
}
