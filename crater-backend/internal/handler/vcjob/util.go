package vcjob

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
)

func GenerateVolumeMounts(c *gin.Context, userID uint, volumes []VolumeMount) (pvc []v1.Volume, volumeMounts []v1.VolumeMount, err error) {
	pvc = []v1.Volume{
		{
			Name: VolumeData,
			VolumeSource: v1.VolumeSource{
				PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
					ClaimName: config.GetConfig().Workspace.PVCName,
				},
			},
		},
		{
			Name: VolumeCache,
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{
					Medium: v1.StorageMediumMemory,
				},
			},
		},
	}

	volumeMounts = make([]v1.VolumeMount, len(volumes)+2)
	u := query.User
	user, err := u.WithContext(c).Where(u.ID.Eq(userID)).First()
	if err != nil {
		return nil, nil, err
	}
	volumeMounts[0] = v1.VolumeMount{
		Name:      VolumeData,
		MountPath: "/home/" + user.Name,
		SubPath:   user.Space,
	}
	volumeMounts[1] = v1.VolumeMount{
		Name:      VolumeCache,
		MountPath: "/dev/shm",
	}

	for i, vm := range volumes {
		volumeMounts[i+2] = v1.VolumeMount{
			Name:      VolumeData,
			SubPath:   vm.SubPath,
			MountPath: vm.MountPath,
		}
	}

	return pvc, volumeMounts, nil
}

func generateNodeAffinity(expressions []v1.NodeSelectorRequirement) (affinity *v1.Affinity) {
	if len(expressions) > 0 {
		affinity = lo.ToPtr(v1.Affinity{
			NodeAffinity: lo.ToPtr(v1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: lo.ToPtr(v1.NodeSelector{
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
					AllowPrivilegeEscalation: lo.ToPtr(true),
					RunAsUser:                lo.ToPtr(int64(0)),
					RunAsGroup:               lo.ToPtr(int64(0)),
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
