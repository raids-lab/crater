package util

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
)

type VolumeType uint

const (
	_ VolumeType = iota
	FileType
	DataType
)

type JobSource uint

const (
	_ JobSource = iota
	Task
	ImageCreate
)

type VolumeMount struct {
	Type      VolumeType `json:"type"`
	DatasetID uint       `json:"datasetID"`
	SubPath   string     `json:"subPath"`
	MountPath string     `json:"mountPath"`
}

// resolveVolumeMount resolves the subpath and volume type based on the mount configuration
func ResolveVolumeMount(c context.Context, token JWTMessage, vm VolumeMount, source JobSource) (
	mount v1.VolumeMount, err error,
) {
	// Get PVC names from config
	pvc := config.GetConfig().Storage.PVC
	rwxPVCName, roxPVCName := pvc.ReadWriteMany, pvc.ReadWriteMany
	if pvc.ReadOnlyMany != nil {
		roxPVCName = *pvc.ReadOnlyMany
	}

	// Handle dataset type volumes - always read-only
	if vm.Type == DataType {
		// Get dataset path from database
		datasetPath, editable, err := GetSubPathByDatasetVolume(c, token.UserID, vm.DatasetID)
		if err != nil {
			return v1.VolumeMount{}, err
		}
		// If editable is true, use RWX PVC
		if editable {
			return v1.VolumeMount{
				Name:      rwxPVCName,
				SubPath:   datasetPath,
				MountPath: vm.MountPath,
				ReadOnly:  false,
			}, nil
		}
		// If editable is false, use ROX PVC
		return v1.VolumeMount{
			Name:      roxPVCName,
			SubPath:   datasetPath,
			MountPath: vm.MountPath,
			ReadOnly:  true,
		}, nil
	}

	// Handle file type volumes based on path prefix
	switch {
	case strings.HasPrefix(vm.SubPath, "public"):
		// Public space paths - permission based on PublicAccessMode
		subPath := filepath.Clean(config.GetConfig().Storage.Prefix.Public + strings.TrimPrefix(vm.SubPath, "public"))
		if IsReadOnly(token.PublicAccessMode) || source == ImageCreate {
			// Read-only access
			return v1.VolumeMount{
				Name:      roxPVCName,
				SubPath:   subPath,
				MountPath: vm.MountPath,
				ReadOnly:  true,
			}, nil
		}
		return v1.VolumeMount{
			Name:      rwxPVCName,
			SubPath:   subPath,
			MountPath: vm.MountPath,
			ReadOnly:  false,
		}, nil
	case strings.HasPrefix(vm.SubPath, "account"):
		// Account space paths - permission based on AccountAccessMode
		a := query.Account
		account, err := a.WithContext(c).Where(a.ID.Eq(token.AccountID)).First()
		if err != nil {
			return v1.VolumeMount{}, err
		}
		subPath := filepath.Clean(config.GetConfig().Storage.Prefix.Account + "/" + account.Space + strings.TrimPrefix(vm.SubPath, "account"))
		if IsReadOnly(token.AccountAccessMode) || source == ImageCreate {
			// Read-only access
			return v1.VolumeMount{
				Name:      roxPVCName,
				SubPath:   subPath,
				MountPath: vm.MountPath,
				ReadOnly:  true,
			}, nil
		}
		// Read-write access
		return v1.VolumeMount{
			Name:      rwxPVCName,
			SubPath:   subPath,
			MountPath: vm.MountPath,
			ReadOnly:  false,
		}, nil
	case strings.HasPrefix(vm.SubPath, "user"):
		// User space paths - always read-write
		u := query.User
		user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if err != nil {
			return v1.VolumeMount{}, err
		}
		subPath := filepath.Clean(config.GetConfig().Storage.Prefix.User + "/" + user.Space + strings.TrimPrefix(vm.SubPath, "user"))
		// User's own space always gets read-write access
		if source == ImageCreate {
			// For image creation jobs, use ROX PVC to avoid accidental modifications
			return v1.VolumeMount{
				Name:      roxPVCName,
				SubPath:   subPath,
				MountPath: vm.MountPath,
				ReadOnly:  true,
			}, nil
		}
		return v1.VolumeMount{
			Name:      rwxPVCName,
			SubPath:   subPath,
			MountPath: vm.MountPath,
			ReadOnly:  false,
		}, nil
	default:
		return v1.VolumeMount{}, fmt.Errorf("invalid mount path format: %s", vm.SubPath)
	}
}

// isReadOnly determines the appropriate PVC and read-only flag based on the access mode
func IsReadOnly(accessMode model.AccessMode) bool {
	switch accessMode {
	case model.AccessModeRO, model.AccessModeAO:
		// Read-only access
		return true
	case model.AccessModeRW:
		// Read-write access
		return false
	default:
		// Invalid access mode
		return true
	}
}

// 创建 PVC Volume
func CreateVolume(volumeName string) v1.Volume {
	return v1.Volume{
		Name: volumeName,
		VolumeSource: v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: volumeName,
			},
		},
	}
}

func GetSubPathByDatasetVolume(c context.Context,
	userID, datasetID uint) (subPath string, editable bool, err error) {
	ud := query.UserDataset
	d := query.Dataset
	ad := query.AccountDataset
	ua := query.UserAccount
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetID)).First()
	if err != nil {
		return "", false, err
	}
	editable = dataset.Extra.Data().Editable
	// Find()方法没找到不会报err，而是返回nil
	accountDatasets, err := ad.WithContext(c).Where(ad.DatasetID.Eq(datasetID)).Find()
	if err != nil {
		return "", false, err
	}
	for _, accountDataset := range accountDatasets {
		_, err = ua.WithContext(c).Where(ua.AccountID.Eq(accountDataset.AccountID), ua.UserID.Eq(userID)).First()
		if err == nil {
			return dataset.URL, editable, nil
		}
	}
	_, err = ud.WithContext(c).Where(ud.UserID.Eq(userID), ud.DatasetID.Eq(datasetID)).First()
	if err != nil {
		return "", false, err
	}

	return dataset.URL, editable, nil
}
