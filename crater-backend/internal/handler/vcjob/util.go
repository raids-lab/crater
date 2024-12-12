package vcjob

import (
	"context"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	v1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const FileType = 1
const DatasetType = 2

func GenerateVolumeMounts(
	_ context.Context,
	_ uint,
	volumes []VolumeMount,
) (pvc []v1.Volume, volumeMounts []v1.VolumeMount, err error) {
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

	if len(volumes) > 0 {
		fs := v1.Volume{
			Name: VolumeData,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.GetConfig().Workspace.PVCName,
				},
			},
		}
		pvc = append(pvc, fs)
	}

	volumeMounts = make([]v1.VolumeMount, len(volumes)+1)

	volumeMounts[0] = v1.VolumeMount{
		Name:      VolumeCache,
		MountPath: "/dev/shm",
	}

	for i, vm := range volumes {
		volumeMounts[i+1] = v1.VolumeMount{
			Name:      VolumeData,
			SubPath:   vm.SubPath,
			MountPath: vm.MountPath,
		}
	}

	return pvc, volumeMounts, nil
}

func GenerateNewVolumeMounts(
	c context.Context,
	userID uint,
	volumes []VolumeMount,
) (pvc []v1.Volume, volumeMounts []v1.VolumeMount, err error) {
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
	if len(volumes) > 0 {
		fs := v1.Volume{
			Name: VolumeData,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.GetConfig().Workspace.PVCName,
				},
			},
		}
		pvc = append(pvc, fs)
	}

	volumeMounts = make([]v1.VolumeMount, len(volumes)+1)
	volumeMounts[0] = v1.VolumeMount{
		Name:      VolumeCache,
		MountPath: "/dev/shm",
	}

	for i, vm := range volumes {
		var subPath string
		if vm.Type == FileType {
			subPath = vm.SubPath
		} else if vm.Type == DatasetType {
			subPath, err = GetSubPathByDatasetVolume(c, userID, vm.DatasetID)
			if err != nil {
				return nil, nil, err
			}
		} else {
			return nil, nil, err
		}

		volumeMounts[i+1] = v1.VolumeMount{
			Name:      VolumeData,
			SubPath:   subPath,
			MountPath: vm.MountPath,
			ReadOnly:  vm.Type == DatasetType,
		}
	}

	return pvc, volumeMounts, nil
}

func GetSubPathByDatasetVolume(c context.Context,
	userID, datasetID uint) (string, error) {
	ud := query.UserDataset
	d := query.Dataset
	_, err := ud.WithContext(c).Where(ud.UserID.Eq(userID), ud.DatasetID.Eq(datasetID)).First()
	if err != nil {
		return "", err
	}
	dataset, err := d.WithContext(c).Where(d.ID.Eq(datasetID)).First()
	if err != nil {
		return "", err
	}
	return dataset.URL, nil
}

func GenerateNodeAffinity(expressions []v1.NodeSelectorRequirement) (affinity *v1.Affinity) {
	if len(expressions) > 0 {
		affinity = ptr.To(v1.Affinity{
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
	}
	return affinity
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
