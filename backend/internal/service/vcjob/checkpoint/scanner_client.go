package checkpoint

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
)

const defaultServiceScanTimeout = 30 * time.Second

type ServiceScannerOptions struct {
	Endpoint string
	Timeout  time.Duration
}

func ScanJobWithService(ctx context.Context, record *model.Job, opts ServiceScannerOptions) (*ScanResult, error) {
	info, storagePath, err := prepareScan(record)
	if err != nil {
		return nil, err
	}
	opts = normalizeServiceScannerOptions(opts)
	if opts.Endpoint == "" {
		return nil, errServiceScannerDisabled
	}

	resp, err := requestServiceScan(ctx, opts, ServiceScanRequest{
		JobName:       record.JobName,
		Framework:     info.Framework,
		CheckpointDir: info.CheckpointDir,
		StoragePath:   storagePath,
	})
	if err != nil {
		return nil, err
	}

	candidates := make([]model.JobCheckpoint, 0, len(resp.Items))
	for _, item := range resp.Items {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			name = filepath.Base(item.StoragePath)
		}
		path := strings.TrimSpace(item.Path)
		if path == "" {
			path = filepath.ToSlash(filepath.Join(info.CheckpointDir, name))
		}
		itemStoragePath := strings.TrimSpace(item.StoragePath)
		if itemStoragePath == "" {
			itemStoragePath = filepath.ToSlash(filepath.Join(storagePath, name))
		}
		checkpoint := newCheckpointRecord(record, info, name, path, itemStoragePath, item.SizeBytes, item.ModTime)
		if item.Step >= 0 {
			checkpoint.Step = item.Step
		}
		if checkpoint.Metadata == nil {
			checkpoint.Metadata = datatypes.JSONMap{}
		}
		checkpoint.Metadata["scanBackend"] = scannerBackendService
		candidates = append(candidates, checkpoint)
	}
	return finishScan(ctx, record, info, storagePath, candidates)
}

var errServiceScannerDisabled = fmt.Errorf("checkpoint scanner service endpoint is not configured")

func normalizeServiceScannerOptions(opts ServiceScannerOptions) ServiceScannerOptions {
	if opts.Endpoint == "" {
		opts.Endpoint = strings.TrimSpace(os.Getenv("CRATER_CHECKPOINT_SCANNER_ENDPOINT"))
	}
	if opts.Endpoint == "" {
		cfg := config.GetConfig()
		opts.Endpoint = strings.TrimSpace(cfg.CheckpointScanner.Endpoint)
		if opts.Timeout <= 0 && cfg.CheckpointScanner.TimeoutSeconds > 0 {
			opts.Timeout = time.Duration(cfg.CheckpointScanner.TimeoutSeconds) * time.Second
		}
	}
	if opts.Timeout <= 0 {
		if timeoutEnv := strings.TrimSpace(os.Getenv("CRATER_CHECKPOINT_SCANNER_TIMEOUT_SECONDS")); timeoutEnv != "" {
			if seconds, err := strconv.Atoi(timeoutEnv); err == nil && seconds > 0 {
				opts.Timeout = time.Duration(seconds) * time.Second
			}
		}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultServiceScanTimeout
	}
	opts.Endpoint = strings.TrimRight(opts.Endpoint, "/")
	return opts
}

func requestServiceScan(ctx context.Context, opts ServiceScannerOptions, body ServiceScanRequest) (ServiceScanResponse, error) {
	reqCtx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	payload, err := json.Marshal(body)
	if err != nil {
		return ServiceScanResponse{}, err
	}
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, opts.Endpoint+"/scan", bytes.NewReader(payload))
	if err != nil {
		return ServiceScanResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return ServiceScanResponse{}, fmt.Errorf("call checkpoint scanner service: %w", err)
	}
	defer resp.Body.Close()

	var scanResp ServiceScanResponse
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		if errResp.Error != "" {
			return ServiceScanResponse{}, fmt.Errorf("checkpoint scanner service returned %d: %s", resp.StatusCode, errResp.Error)
		}
		return ServiceScanResponse{}, fmt.Errorf("checkpoint scanner service returned %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&scanResp); err != nil {
		return ServiceScanResponse{}, fmt.Errorf("decode checkpoint scanner service response: %w", err)
	}
	if scanResp.Items == nil {
		scanResp.Items = []ServiceScanItem{}
	}
	return scanResp, nil
}
