package ceph

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	cfgpkg "github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/storagequota"
)

const toolboxOperationTimeout = 20 * time.Second

// ErrQuotaWriteOutcomeUnknown means a failed provider response could not prove whether the quota was applied.
var ErrQuotaWriteOutcomeUnknown = errors.New("CephFS quota write outcome is unknown")

type StoragePrefixConfig struct {
	User    string
	Account string
	Public  string
}

func sharedStoragePVCName() string {
	storagePVCName := "crater-storage"
	if cfg := cfgpkg.GetConfig(); cfg != nil {
		if value := strings.TrimSpace(cfg.Storage.PVC.ReadWriteMany); value != "" {
			storagePVCName = value
		}
	}
	return storagePVCName
}

func StorageQuotaProvider() string {
	cfg := cfgpkg.GetConfig()
	if !storageQuotaEnabled(cfg.Storage.Quota.Enabled) {
		return storagequota.ProviderDisabled
	}
	return storagequota.NormalizeProvider(cfg.Storage.Quota.Provider)
}

func StorageQuotaEnabled() bool {
	cfg := cfgpkg.GetConfig()
	return storageQuotaEnabled(cfg.Storage.Quota.Enabled)
}

func StorageQuotaRookNamespace() string {
	return cfgpkg.GetConfig().StorageQuotaRookNamespace()
}

func StorageQuotaCephFSCSIDriver() string {
	return cfgpkg.GetConfig().StorageQuotaCephFSCSIDriver()
}

func StorageQuotaToolboxLabelSelector() string {
	return cfgpkg.GetConfig().StorageQuotaToolboxLabelSelector()
}

func StorageQuotaCephFSName() string {
	return cfgpkg.GetConfig().StorageQuotaCephFSName()
}

func storageQuotaEnabled(enabled *bool) bool {
	return enabled != nil && *enabled
}

func GetStorageServerQuotaCapabilities(ctx context.Context) (storagequota.Capabilities, error) {
	return storageQuotaClient().GetCapabilities(ctx)
}

func logicalPathToStorageRelativePath(
	logicalPath string,
	prefixConfig StoragePrefixConfig,
) (string, error) {
	trimmedPath := strings.Trim(strings.ReplaceAll(logicalPath, "\\", "/"), "/")
	parts := strings.SplitN(trimmedPath, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("invalid path format: %s", logicalPath)
	}

	var storagePrefix string
	var remainingPath string
	if len(parts) == 2 {
		remainingPath = parts[1]
	}
	switch parts[0] {
	case "user":
		if remainingPath == "" {
			return "", fmt.Errorf("user path must include space name: %s", logicalPath)
		}
		storagePrefix = prefixConfig.User
	case "account":
		storagePrefix = prefixConfig.Account
	case "public":
		storagePrefix = prefixConfig.Public
	default:
		return "", fmt.Errorf("unknown path type: %s", parts[0])
	}

	relativePath := path.Clean(path.Join(storagePrefix, remainingPath))
	cleanPrefix := path.Clean(strings.Trim(storagePrefix, "/"))
	if cleanPrefix == "." || cleanPrefix == ".." || strings.HasPrefix(cleanPrefix, "../") {
		return "", fmt.Errorf("invalid storage prefix for %s", parts[0])
	}
	if relativePath != cleanPrefix && !strings.HasPrefix(relativePath, cleanPrefix+"/") {
		return "", fmt.Errorf("path escapes storage prefix: %s", logicalPath)
	}
	return relativePath, nil
}

func storageQuotaClient() *storagequota.Client {
	cfg := cfgpkg.GetConfig()
	return storagequota.NewClient(
		storagequota.ResolveServerURL(cfg.Storage.Quota.StorageServerURL, cfg.Namespaces.Job),
		cfg.Auth.Token.AccessTokenSecret,
	)
}

func GetCephDirectorySize(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
) (int64, error) {
	relativePath, err := logicalPathToStorageRelativePath(logicalPath, prefixConfig)
	if err != nil {
		return 0, err
	}
	return getDirectoryUsage(clientset, config, namespace, relativePath)
}

// GetCephDirectoryQuota returns the quota enforced by CephFS for a logical path.
// A value of -1 means that no quota is configured.
func GetCephDirectoryQuota(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
) (int64, error) {
	relativePath, err := logicalPathToStorageRelativePath(logicalPath, prefixConfig)
	if err != nil {
		return 0, err
	}
	return getDirectoryQuota(clientset, config, namespace, relativePath)
}

func SetCephDirectoryQuota(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, logicalPath string,
	prefixConfig StoragePrefixConfig,
	maxBytes int64,
) error {
	relativePath, err := logicalPathToStorageRelativePath(logicalPath, prefixConfig)
	if err != nil {
		return err
	}
	return setDirectoryQuota(clientset, config, namespace, relativePath, maxBytes)
}

func getDirectoryUsage(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
) (int64, error) {
	provider := StorageQuotaProvider()
	if provider == storagequota.ProviderDisabled {
		return 0, fmt.Errorf("storage quota provider is disabled")
	}

	var storageServerErr error
	if provider == storagequota.ProviderAuto || provider == storagequota.ProviderStorageServer {
		usage, err := storageQuotaClient().GetUsage(context.Background(), relativePath)
		if err == nil {
			return usage.Bytes, nil
		}
		storageServerErr = fmt.Errorf("read directory usage from storage-server: %w", err)
		if provider == storagequota.ProviderStorageServer {
			return 0, storageServerErr
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), toolboxOperationTimeout)
	defer cancel()
	usage, toolboxErr := getToolboxUsage(ctx, clientset, config, namespace, relativePath)
	if toolboxErr == nil {
		return usage, nil
	}
	if storageServerErr != nil {
		return 0, fmt.Errorf("storage quota providers failed: %w", errors.Join(storageServerErr, toolboxErr))
	}
	return 0, fmt.Errorf("read directory usage through toolbox: %w", toolboxErr)
}

func getDirectoryQuota(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
) (int64, error) {
	provider := StorageQuotaProvider()
	if provider == storagequota.ProviderDisabled {
		return 0, fmt.Errorf("storage quota provider is disabled")
	}

	var storageServerErr error
	if provider == storagequota.ProviderAuto || provider == storagequota.ProviderStorageServer {
		quota, err := storageQuotaClient().GetQuota(context.Background(), relativePath)
		if err == nil {
			return normalizeCephQuota(quota.MaxBytes), nil
		}
		storageServerErr = fmt.Errorf("read directory quota from storage-server: %w", err)
		if provider == storagequota.ProviderStorageServer {
			return 0, storageServerErr
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), toolboxOperationTimeout)
	defer cancel()
	quota, toolboxErr := getToolboxQuota(ctx, clientset, config, namespace, relativePath)
	if toolboxErr == nil {
		return quota, nil
	}
	if storageServerErr != nil {
		return 0, fmt.Errorf("storage quota providers failed: %w", errors.Join(storageServerErr, toolboxErr))
	}
	return 0, fmt.Errorf("read directory quota through toolbox: %w", toolboxErr)
}

func normalizeCephQuota(maxBytes int64) int64 {
	if maxBytes == 0 {
		return -1
	}
	return maxBytes
}

func setDirectoryQuota(
	clientset kubernetes.Interface,
	config *rest.Config,
	namespace, relativePath string,
	maxBytes int64,
) error {
	if maxBytes < -1 || maxBytes == 0 {
		return fmt.Errorf("quota must be -1 for unlimited or greater than zero")
	}

	provider := StorageQuotaProvider()
	if provider == storagequota.ProviderDisabled {
		return fmt.Errorf("storage quota provider is disabled")
	}

	var storageServerErr error
	if provider == storagequota.ProviderAuto || provider == storagequota.ProviderStorageServer {
		storageServerErr = setStorageServerQuota(storageQuotaClient(), relativePath, maxBytes)
		if storageServerErr == nil {
			return nil
		}
		if provider == storagequota.ProviderStorageServer {
			return storageServerErr
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), toolboxOperationTimeout)
	defer cancel()
	toolboxQuota := cephQuotaXattrValue(maxBytes)
	toolboxErr := setToolboxQuota(ctx, clientset, config, namespace, relativePath, toolboxQuota)
	if toolboxErr == nil {
		return nil
	}
	toolboxErr = reconcileQuotaWriteError(toolboxErr, maxBytes, func() (int64, error) {
		return getToolboxQuota(ctx, clientset, config, namespace, relativePath)
	})
	if toolboxErr == nil {
		return nil
	}
	if storageServerErr != nil {
		return fmt.Errorf("storage quota providers failed: %w", errors.Join(storageServerErr, toolboxErr))
	}
	return toolboxErr
}

func setStorageServerQuota(client *storagequota.Client, relativePath string, maxBytes int64) error {
	if _, err := client.SetQuota(context.Background(), relativePath, maxBytes); err == nil {
		return nil
	} else {
		writeErr := fmt.Errorf("write directory quota through storage-server: %w", err)
		return reconcileQuotaWriteError(writeErr, maxBytes, func() (int64, error) {
			quota, readErr := client.GetQuota(context.Background(), relativePath)
			return normalizeCephQuota(quota.MaxBytes), readErr
		})
	}
}

func reconcileQuotaWriteError(
	writeErr error,
	requestedQuota int64,
	readBack func() (int64, error),
) error {
	actualQuota, readErr := readBack()
	if readErr != nil {
		return errors.Join(
			ErrQuotaWriteOutcomeUnknown,
			writeErr,
			fmt.Errorf("read back CephFS quota after write error: %w", readErr),
		)
	}
	if normalizeCephQuota(actualQuota) == requestedQuota {
		return nil
	}
	return fmt.Errorf(
		"%w (readback=%d, requested=%d)",
		writeErr,
		normalizeCephQuota(actualQuota),
		requestedQuota,
	)
}
