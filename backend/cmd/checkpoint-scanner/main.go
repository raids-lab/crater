package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"k8s.io/klog/v2"

	checkpointsvc "github.com/raids-lab/crater/internal/service/vcjob/checkpoint"
)

var (
	AppVersion string
	CommitSHA  string
	BuildType  string
	BuildTime  string
)

type scanServer struct {
	scanner checkpointsvc.FileSystemScanner
	sem     chan struct{}
}

func main() {
	initVersionInfo()

	root := firstNonEmptyEnv("CRATER_CHECKPOINT_SCANNER_ROOT", "CRATER_STORAGE_ROOT", "ROOTDIR")
	if root == "" {
		root = checkpointsvc.DefaultScannerMountPath
	}
	port := firstNonEmptyEnv("CRATER_CHECKPOINT_SCANNER_PORT", "PORT")
	if port == "" {
		port = checkpointsvc.DefaultScannerPort
	}
	concurrency := positiveIntEnv("CRATER_CHECKPOINT_SCANNER_CONCURRENCY", 4)

	server := &scanServer{
		scanner: checkpointsvc.NewFileSystemScanner(root),
		sem:     make(chan struct{}, concurrency),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", checkpointsvc.ScannerHealthHandler)
	mux.HandleFunc("/readyz", checkpointsvc.ScannerHealthHandler)
	mux.HandleFunc("/scan", server.scan)

	addr := normalizePort(port)
	klog.Infof("checkpoint-scanner starting on %s root=%s concurrency=%d version=%s commit=%s buildType=%s buildTime=%s",
		addr, root, concurrency, AppVersion, CommitSHA, BuildType, BuildTime)
	if err := http.ListenAndServe(addr, mux); err != nil {
		klog.Fatalf("checkpoint-scanner failed: %v", err)
	}
}

func (s *scanServer) scan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	select {
	case s.sem <- struct{}{}:
		defer func() { <-s.sem }()
	default:
		writeError(w, http.StatusTooManyRequests, "checkpoint scanner is busy")
		return
	}

	defer r.Body.Close()
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var req checkpointsvc.ServiceScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid request body: %v", err))
		return
	}
	if err := checkpointsvc.ValidateServiceScanRequest(req); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	resp, err := s.scanner.Scan(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func firstNonEmptyEnv(keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return value
		}
	}
	return ""
}

func positiveIntEnv(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func normalizePort(port string) string {
	if strings.HasPrefix(port, ":") {
		return port
	}
	return ":" + port
}

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
		BuildTime = time.Now().UTC().Format(time.RFC3339)
	}
}
