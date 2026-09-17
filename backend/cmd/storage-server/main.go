package main

import (
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/storage"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/storagequota"
)

var (
	AppVersion string
	CommitSHA  string
	BuildType  string
	BuildTime  string
)

const (
	storageModeFull       = "full"
	storageModeQuotaAgent = "quota-agent"
)

func initVersionInfo() {
	if AppVersion == "" {
		AppVersion = "dev-local"
	}
	if CommitSHA == "" {
		CommitSHA = "unknown"
	}
	if BuildType == "" {
		BuildType = "development"
	}
	if BuildTime == "" {
		BuildTime = "unknown"
	}
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if val := strings.TrimSpace(os.Getenv(key)); val != "" {
			return val
		}
	}
	return ""
}

func normalizePort(port string) string {
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

//nolint:gocyclo // Startup validates mode-specific dependencies in one linear flow.
func main() {
	initVersionInfo()

	if gin.IsDebugging() {
		if err := godotenv.Load(".debug.env"); err != nil {
			klog.Infof("skip loading .debug.env: %v", err)
			if fallbackErr := godotenv.Load(".env"); fallbackErr != nil {
				klog.Infof("skip loading .env: %v", fallbackErr)
			}
		}
	}

	mode := strings.ToLower(firstNonEmptyEnv("CRATER_STORAGE_MODE"))
	if mode == "" {
		mode = storageModeFull
	}
	if mode != storageModeFull && mode != storageModeQuotaAgent {
		klog.Fatalf("unsupported storage-server mode %q; expected full or quota-agent", mode)
	}
	hasInternalCredential := strings.TrimSpace(os.Getenv(storagequota.InternalTokenEnv)) != "" ||
		strings.TrimSpace(os.Getenv(storagequota.InternalSecretEnv)) != ""
	if mode == storageModeFull || !hasInternalCredential {
		_ = config.GetConfig()
	}
	if mode == storageModeFull {
		query.SetDefault(query.GetDB())
	}

	port := firstNonEmptyEnv("CRATER_STORAGE_PORT", "PORT")
	if port == "" {
		port = "7320"
	}

	rootDir := firstNonEmptyEnv("CRATER_STORAGE_ROOT", "ROOTDIR")
	if rootDir == "" {
		rootDir = "/crater"
	}
	if err := os.MkdirAll(rootDir, os.ModePerm); err != nil {
		klog.Fatalf("failed to create storage root directory %s: %v", rootDir, err)
	}
	storage.SetRootDir(rootDir)

	r := gin.Default()
	if mode == storageModeQuotaAgent {
		storage.RegisterQuotaRoutes(r)
	} else {
		go storage.StartCheckSpace()
		storage.RegisterRoutes(r)
	}

	addr := normalizePort(port)
	klog.Infof("storage-server starting on %s (mode=%s, version=%s, commit=%s, buildType=%s, buildTime=%s)",
		addr, mode, AppVersion, CommitSHA, BuildType, BuildTime)
	if err := r.Run(addr); err != nil {
		klog.Fatalf("failed to run storage-server: %v", err)
	}
}
