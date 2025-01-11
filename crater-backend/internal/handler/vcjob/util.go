package vcjob

import (
	"context"
	"fmt"
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
	DatasetType
)

const userSpacePrefix = "***REMOVED***"
const accountSpacePrefix = "***REMOVED***"
const publicSpacePrefix = "***REMOVED***"

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

// 解析 SubPath 和空间类型
func resolveSubPathAndSpaceType(c context.Context, token util.JWTMessage, vm VolumeMount) (
	subPath, volumeName string, readOnly bool, err error,
) {
	rwxPVCName := config.GetConfig().Workspace.RWXPVCName
	roxPVCName := config.GetConfig().Workspace.ROXPVCName

	processSubPath := func(subPath string, accessMode model.AccessMode, isDatasetType bool) (string, string, bool, error) {
		switch {
		case strings.HasPrefix(subPath, "public"):
			subPath = publicSpacePrefix + strings.TrimPrefix(subPath, "public")
			if isDatasetType {
				return subPath, roxPVCName, true, nil
			}
			return determineAccessMode(subPath, accessMode)
		case strings.HasPrefix(subPath, "account") && !isDatasetType:
			subPath = accountSpacePrefix + strings.TrimPrefix(subPath, "account")
			return determineAccessMode(subPath, accessMode)
		case strings.HasPrefix(subPath, "user"):
			subPath = userSpacePrefix + strings.TrimPrefix(subPath, "user")
			if isDatasetType {
				return subPath, roxPVCName, true, nil
			}
			return subPath, rwxPVCName, false, nil
		default:
			return "", "", false, fmt.Errorf("mount path error")
		}
	}

	switch vm.Type {
	case DatasetType:
		// DatasetType 按只读挂载，复用 SubPath 处理逻辑
		subPath, err = GetSubPathByDatasetVolume(c, token.UserID, vm.DatasetID)
		if err != nil {
			return "", "", false, err
		}
		return processSubPath(subPath, model.AccessModeRO, true)
	default:
		// FileType
		subPath = vm.SubPath
		return processSubPath(subPath, token.PublicAccessMode, false)
	}
}

// 根据权限模式判断 Volume 名称和只读属性
func determineAccessMode(subPath string, accessModel model.AccessMode) (
	volumeSubPath, volumeName string, readOnly bool, err error,
) {
	rwxPVCName := config.GetConfig().Workspace.RWXPVCName
	roxPVCName := config.GetConfig().Workspace.ROXPVCName

	switch accessModel {
	case model.AccessModeNA:
		return "", "", false, fmt.Errorf("access to public directory is not allowed")
	case model.AccessModeRO, model.AccessModeAO:
		return subPath, roxPVCName, true, nil
	case model.AccessModeRW:
		return subPath, rwxPVCName, false, nil
	default:
		return "", "", false, fmt.Errorf("unknown access mode for public directory")
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
	volumes []v1.Volume,
	volumeMounts []v1.VolumeMount,
	envs []v1.EnvVar,
	ports []v1.ContainerPort,
) (podSpec v1.PodSpec) {
	podSpec = v1.PodSpec{
		Affinity: affinity,
		Volumes:  volumes,
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
		podSpec.Containers[0].Command = []string{"sh", "-c", *task.Command}
	}
	if task.WorkingDir != nil {
		podSpec.Containers[0].WorkingDir = *task.WorkingDir
	}
	return podSpec
}
