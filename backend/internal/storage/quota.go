package storage

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/storagequota"
)

const (
	cephDirectoryBytesXattr = "ceph.dir.rbytes"
	cephQuotaMaxBytesXattr  = "ceph.quota.max_bytes"
	parentDirectoryPath     = ".."
)

func RegisterQuotaRoutes(r *gin.Engine) {
	group := r.Group("/internal/storage", requireInternalStorageToken())
	group.GET("/capabilities", GetQuotaCapabilities)
	group.GET("/usage", GetDirectoryUsage)
	group.GET("/quota", GetDirectoryQuota)
	group.PUT("/quota", SetDirectoryQuota)
}

func requireInternalStorageToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		suppliedToken := c.GetHeader(storagequota.InternalTokenHeader)
		if token := strings.TrimSpace(os.Getenv(storagequota.InternalTokenEnv)); token != "" {
			if !storagequota.AuthenticateToken(token, suppliedToken) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal storage token"})
				return
			}
			c.Next()
			return
		}

		secret := strings.TrimSpace(os.Getenv(storagequota.InternalSecretEnv))
		if secret == "" {
			secret = config.GetConfig().Auth.Token.AccessTokenSecret
		}
		if secret == "" || !storagequota.Authenticate(secret, suppliedToken) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid internal storage token"})
			return
		}
		c.Next()
	}
}

func GetQuotaCapabilities(c *gin.Context) {
	capabilities := inspectXattrCapabilities(storageRootDir)
	c.JSON(http.StatusOK, capabilities)
}

func GetDirectoryUsage(c *gin.Context) {
	targetPath, relativePath, err := resolveStorageUsagePath(c.Query("path"))
	if err != nil {
		writeStoragePathError(c, err)
		return
	}

	value, err := readXattrInt64(targetPath, cephDirectoryBytesXattr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, storagequota.Usage{Path: relativePath, Bytes: value})
}

//nolint:gocritic // The tuple distinguishes the resolved filesystem path from the API-relative path.
func resolveStorageUsagePath(rawPath string) (string, string, error) {
	if strings.TrimSpace(rawPath) != "." {
		return resolveStorageDirectory(rawPath)
	}

	root, err := filepath.Abs(storageRootDir)
	if err != nil {
		return "", "", err
	}
	root, _, err = secureStoragePaths(root, root)
	if err != nil {
		return "", "", err
	}
	return root, ".", nil
}

func GetDirectoryQuota(c *gin.Context) {
	targetPath, relativePath, err := resolveStorageDirectory(c.Query("path"))
	if err != nil {
		writeStoragePathError(c, err)
		return
	}

	value, err := readXattrInt64(targetPath, cephQuotaMaxBytesXattr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if value == 0 {
		value = -1
	}
	c.JSON(http.StatusOK, storagequota.Quota{Path: relativePath, MaxBytes: value})
}

func SetDirectoryQuota(c *gin.Context) {
	var request storagequota.Quota
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quota request: " + err.Error()})
		return
	}
	if request.MaxBytes < -1 || request.MaxBytes == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "max_bytes must be -1 or greater than zero"})
		return
	}

	targetPath, relativePath, err := resolveStorageDirectory(request.Path)
	if err != nil {
		writeStoragePathError(c, err)
		return
	}

	maxBytes := request.MaxBytes
	if maxBytes == -1 {
		maxBytes = 0
	}
	if err := writeXattr(targetPath, cephQuotaMaxBytesXattr, []byte(strconv.FormatInt(maxBytes, 10))); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, storagequota.Quota{Path: relativePath, MaxBytes: request.MaxBytes})
}

//nolint:gocritic // The tuple distinguishes the resolved filesystem path from the API-relative path.
func resolveStorageDirectory(rawPath string) (string, string, error) {
	rawPath = strings.TrimSpace(strings.ReplaceAll(rawPath, "\\", "/"))
	if rawPath == "" || strings.ContainsRune(rawPath, '\x00') {
		return "", "", errInvalidStoragePath
	}
	if strings.HasPrefix(rawPath, "/") {
		return "", "", errInvalidStoragePath
	}

	cleanRelative := filepath.Clean(filepath.FromSlash(rawPath))
	if cleanRelative == "." || cleanRelative == parentDirectoryPath || filepath.IsAbs(cleanRelative) ||
		strings.HasPrefix(cleanRelative, parentDirectoryPath+string(filepath.Separator)) {
		return "", "", errInvalidStoragePath
	}

	root, err := filepath.Abs(storageRootDir)
	if err != nil {
		return "", "", err
	}
	target := filepath.Join(root, cleanRelative)
	root, target, err = secureStoragePaths(root, target)
	if err != nil {
		return "", "", err
	}
	relativeToRoot, err := filepath.Rel(root, target)
	if err != nil || relativeToRoot == ".." || strings.HasPrefix(relativeToRoot, ".."+string(filepath.Separator)) {
		return "", "", errInvalidStoragePath
	}

	info, err := os.Stat(target)
	if err != nil {
		return "", "", err
	}
	if !info.IsDir() {
		return "", "", errStoragePathNotDirectory
	}
	return target, filepath.ToSlash(relativeToRoot), nil
}

var (
	errInvalidStoragePath      = errors.New("storage path must be a relative path below the storage root")
	errStoragePathNotDirectory = errors.New("storage path is not a directory")
)

func writeStoragePathError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, errInvalidStoragePath), errors.Is(err, errStoragePathNotDirectory):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case os.IsNotExist(err):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	}
}
