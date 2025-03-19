package vcjob

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

type VolumeType uint

const (
	_ VolumeType = iota
	FileType
	DataType
)

func GenerateVolumeMounts(
	c context.Context,
	volumes []VolumeMount,
	token util.JWTMessage, // 传入 token 信息
) (pvc []v1.Volume, volumeMounts []v1.VolumeMount, err error) {
	// 初始化返回的 PVC 和 VolumeMount 列表
	pvcMap := make(map[string]bool) // 用于避免重复创建同一 PVC
	volumeMounts = []v1.VolumeMount{
		{
			Name:      VolumeCache,
			MountPath: "/dev/shm",
		},
	}
	pvc = []v1.Volume{
		{
			Name: VolumeCache,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMediumMemory,
				},
			},
		},
	}

	// 遍历 volumes，根据权限动态创建 PVC
	for _, vm := range volumes {
		subPath, volumeName, readOnly, err := resolveSubPathAndSpaceType(c, token, vm)
		if err != nil {
			return nil, nil, err
		}

		// 如果该 PVC 尚未创建，添加到 PVC 列表
		if !pvcMap[volumeName] {
			pvc = append(pvc, createVolume(volumeName))
			pvcMap[volumeName] = true
		}

		// 添加挂载点
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      volumeName,
			SubPath:   subPath,
			MountPath: vm.MountPath,
			ReadOnly:  readOnly,
		})
	}

	return pvc, volumeMounts, nil
}

// resolveSubPathAndSpaceType resolves the subpath and volume type based on the mount configuration
func resolveSubPathAndSpaceType(c context.Context, token util.JWTMessage, vm VolumeMount) (
	subPath, volumeName string, readOnly bool, err error,
) {
	// Get PVC names from config
	rwxPVCName := config.GetConfig().Workspace.RWXPVCName
	roxPVCName := config.GetConfig().Workspace.ROXPVCName

	// Handle dataset type volumes - always read-only
	if vm.Type == DataType {
		// Get dataset path from database
		datasetPath, err := GetSubPathByDatasetVolume(c, token.UserID, vm.DatasetID)
		if err != nil {
			return "", "", false, err
		}
		// Datasets are always mounted as read-only
		return datasetPath, roxPVCName, true, nil
	}

	// Handle file type volumes based on path prefix
	switch {
	case strings.HasPrefix(vm.SubPath, "public"):
		// Public space paths - permission based on PublicAccessMode
		subPath := filepath.Clean(config.GetConfig().PublicSpacePrefix + strings.TrimPrefix(vm.SubPath, "public"))
		return determineAccessMode(subPath, token.PublicAccessMode, rwxPVCName, roxPVCName)

	case strings.HasPrefix(vm.SubPath, "account"):
		// Account space paths - permission based on AccountAccessMode
		a := query.Account
		account, err := a.WithContext(c).Where(a.ID.Eq(token.AccountID)).First()
		if err != nil {
			return "", "", false, err
		}
		subPath := filepath.Clean(config.GetConfig().AccountSpacePrefix + account.Space + strings.TrimPrefix(vm.SubPath, "account"))
		return determineAccessMode(subPath, token.AccountAccessMode, rwxPVCName, roxPVCName)

	case strings.HasPrefix(vm.SubPath, "user"):
		// User space paths - always read-write
		u := query.User
		user, err := u.WithContext(c).Where(u.ID.Eq(token.UserID)).First()
		if err != nil {
			return "", "", false, err
		}
		subPath := config.GetConfig().UserSpacePrefix + "/" + user.Space + strings.TrimPrefix(vm.SubPath, "user")
		// User's own space always gets read-write access
		return subPath, rwxPVCName, false, nil

	default:
		return "", "", false, fmt.Errorf("invalid mount path format: %s", vm.SubPath)
	}
}

// determineAccessMode determines the appropriate PVC and read-only flag based on the access mode
func determineAccessMode(subPath string, accessMode model.AccessMode, rwxPVCName, roxPVCName string) (
	volumeSubPath, volumeName string, readOnly bool, err error,
) {
	switch accessMode {
	case model.AccessModeNA:
		return "", "", false, fmt.Errorf("access to directory is not allowed")
	case model.AccessModeRO, model.AccessModeAO:
		// Read-only access
		return subPath, roxPVCName, true, nil
	case model.AccessModeRW:
		// Read-write access
		return subPath, rwxPVCName, false, nil
	default:
		return "", "", false, fmt.Errorf("unknown access mode: %v", accessMode)
	}
}

// 创建 PVC Volume
func createVolume(volumeName string) v1.Volume {
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
	userID, datasetID uint) (string, error) {
	ud := query.UserDataset
	d := query.Dataset
	ad := query.AccountDataset
	ua := query.UserAccount
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetID)).First()
	if err != nil {
		return "", err
	}
	// Find()方法没找到不会报err，而是返回nil
	accountDatasets, err := ad.WithContext(c).Where(ad.DatasetID.Eq(datasetID)).Find()
	if err != nil {
		return "", err
	}
	for _, accountDataset := range accountDatasets {
		_, err = ua.WithContext(c).Where(ua.AccountID.Eq(accountDataset.AccountID), ua.UserID.Eq(userID)).First()
		if err == nil {
			return dataset.URL, nil
		}
	}
	_, err = ud.WithContext(c).Where(ud.UserID.Eq(userID), ud.DatasetID.Eq(datasetID)).First()
	if err != nil {
		return "", err
	}

	return dataset.URL, nil
}

func GenerateTaintTolerationsForAccount(token util.JWTMessage) (tolerations []v1.Toleration) {
	// If current account is not default account (which account id is 1), add toleration
	if token.AccountID == model.DefaultAccountID {
		return nil
	}
	return []v1.Toleration{
		{
			Key:      "crater.raids.io/account",
			Operator: v1.TolerationOpEqual,
			Value:    token.AccountName,
			Effect:   v1.TaintEffectNoSchedule,
		},
	}
}

func GenerateNodeAffinity(expressions []v1.NodeSelectorRequirement, totalRequests v1.ResourceList) (affinity *v1.Affinity) {
	gpuCount := GetGPUCountFromResource(totalRequests)

	// expressions will override prefer
	if len(expressions) > 0 {
		return ptr.To(v1.Affinity{
			NodeAffinity: ptr.To(v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: ptr.To(v1.NodeSelector{
					NodeSelectorTerms: []v1.NodeSelectorTerm{
						{
							MatchExpressions: expressions,
						},
					},
				}),
			}),
		})
	} else if gpuCount == 0 {
		return ptr.To(v1.Affinity{
			NodeAffinity: ptr.To(v1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
					{
						Weight: 50,
						Preference: v1.NodeSelectorTerm{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      "nvidia.com/gpu.present",
									Operator: v1.NodeSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			}),
		})
	} else if gpuCount == 1 {
		return ptr.To(v1.Affinity{
			NodeAffinity: ptr.To(v1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: []v1.PreferredSchedulingTerm{
					{
						Weight: 50,
						Preference: v1.NodeSelectorTerm{
							MatchExpressions: []v1.NodeSelectorRequirement{
								{
									Key:      "feature.node.kubernetes.io/pci-15b3.present",
									Operator: v1.NodeSelectorOpDoesNotExist,
								},
							},
						},
					},
				},
			}),
		})
	}
	return affinity
}

func GetGPUCountFromResource(resources v1.ResourceList) (gpuCount int64) {
	// with prefix nvidia.com
	for k, v := range resources {
		if strings.HasPrefix(k.String(), "nvidia.com") {
			return v.Value()
		}
	}
	return 0
}

func generatePodSpec(
	task *TaskReq,
	affinity *v1.Affinity,
	tolerations []v1.Toleration,
	volumes []v1.Volume,
	volumeMounts []v1.VolumeMount,
	envs []v1.EnvVar,
	ports []v1.ContainerPort,
) (podSpec v1.PodSpec) {
	podSpec = v1.PodSpec{
		Affinity:    affinity,
		Tolerations: tolerations,
		Volumes:     volumes,
		Containers: []v1.Container{
			{
				Name:  task.Name,
				Image: task.Image,
				Resources: v1.ResourceRequirements{
					Limits:   task.Resource,
					Requests: task.Resource,
				},
				Env:   envs,
				Ports: ports,
				SecurityContext: &v1.SecurityContext{
					AllowPrivilegeEscalation: ptr.To(true),
					RunAsUser:                ptr.To(int64(0)),
					RunAsGroup:               ptr.To(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
		RestartPolicy: v1.RestartPolicyNever,
	}
	if task.Command != nil {
		if task.Shell != nil {
			podSpec.Containers[0].Command = []string{*task.Shell, "-c", *task.Command}
		} else {
			podSpec.Containers[0].Command = []string{"sh", "-c", *task.Command}
		}
	}
	if task.WorkingDir != nil {
		podSpec.Containers[0].WorkingDir = *task.WorkingDir
	}
	return podSpec
}
