package vcjob

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/utils"
)

type ForwardType uint

const (
	_ ForwardType = iota
	IngressType
	NodePortType
)

type Forward struct {
	Type ForwardType `json:"type"`
	Name string      `json:"name"`
	Port int32       `json:"port"`
}

const (
	annotationKeyTaskName     = "crater.raids.io/task-name"
	annotationKeyTaskTemplate = "crater.raids.io/task-template"
	annotationKeyAlertEnabled = "crater.raids.io/alert-enabled"
	AnnotationKeyUserID       = "crater.raids.io/user-id"
	annotationKeyForwards     = "crater.raids.io/forwards"
	// AnnotationKeyMountedDatasetIDs stores mounted dataset IDs as JSON array on job annotations.
	AnnotationKeyMountedDatasetIDs = "crater.raids.io/mounted-dataset-ids"

	jobTypeJupyter = "jupyter"
	jobTypeWebIDE  = "webide"
	jupyterPort    = 8888
)

func CalculateJobResources(job *batch.Job) v1.ResourceList {
	return utils.CalculateReplicatedResources(
		job.Spec.Tasks,
		func(task batch.TaskSpec) v1.ResourceList {
			return utils.CalculateRequsetsByContainers(task.Template.Spec.Containers)
		},
		func(task batch.TaskSpec) int32 {
			return task.Replicas
		},
	)
}

func RestoreJobFromRecord(record *model.Job) (*batch.Job, error) {
	if record == nil || record.Attributes.Data() == nil {
		return nil, fmt.Errorf("job record has no stored template")
	}

	job := record.Attributes.Data().DeepCopy()
	job.Status = batch.JobStatus{}
	job.UID = ""
	job.ResourceVersion = ""
	job.Generation = 0
	job.CreationTimestamp = metav1.Time{}
	job.ManagedFields = nil
	job.DeletionTimestamp = nil
	job.DeletionGracePeriodSeconds = nil
	return job, nil
}

func CreateForwardIngresses(
	ctx context.Context,
	serviceManager crclient.ServiceManagerInterface,
	job *batch.Job,
	forwards []Forward,
	labels map[string]string,
	username string,
) error {
	for _, forward := range forwards {
		port := &v1.ServicePort{
			Name:       forward.Name,
			Port:       forward.Port,
			TargetPort: intstr.FromInt(int(forward.Port)),
			Protocol:   v1.ProtocolTCP,
		}

		_, err := serviceManager.CreateIngress(
			ctx,
			[]metav1.OwnerReference{
				*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
			},
			labels,
			port,
			config.GetConfig().Host,
			username,
		)
		if err != nil {
			return fmt.Errorf("failed to create ingress for %s: %w", forward.Name, err)
		}
	}
	return nil
}

func ActivateJob(
	ctx context.Context,
	cli client.Client,
	serviceManager crclient.ServiceManagerInterface,
	job *batch.Job,
) error {
	if err := cli.Create(ctx, job); err != nil {
		return err
	}

	cleanup := true
	defer func() {
		if cleanup {
			_ = cli.Delete(context.Background(), job)
		}
	}()

	if err := ensureJobAccessResources(ctx, serviceManager, job); err != nil {
		return err
	}

	cleanup = false
	if err := increaseDatasetMountCount(ctx, job); err != nil {
		klog.Warningf("failed to increase dataset mount count for job %s: %v", job.Name, err)
	}
	return nil
}

func ensureJobAccessResources(
	ctx context.Context,
	serviceManager crclient.ServiceManagerInterface,
	job *batch.Job,
) error {
	jobType := job.Labels[crclient.LabelKeyTaskType]
	username := job.Labels[crclient.LabelKeyTaskUser]
	baseURL := job.Labels[crclient.LabelKeyBaseURL]
	userID, err := strconv.ParseUint(job.Annotations[AnnotationKeyUserID], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid user id annotation: %w", err)
	}

	switch jobType {
	case jobTypeJupyter:
		port := &v1.ServicePort{
			Name:       "notebook",
			Port:       jupyterPort,
			TargetPort: intstr.FromInt(jupyterPort),
			Protocol:   v1.ProtocolTCP,
		}
		if _, err = serviceManager.CreateIngressWithPrefix(
			ctx,
			[]metav1.OwnerReference{
				*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
			},
			job.Labels,
			port,
			config.GetConfig().Host,
			baseURL,
		); err != nil {
			return fmt.Errorf("failed to create jupyter ingress: %w", err)
		}
	case jobTypeWebIDE:
		port := &v1.ServicePort{
			Name:       "webide",
			Port:       jupyterPort,
			TargetPort: intstr.FromInt(jupyterPort),
			Protocol:   v1.ProtocolTCP,
		}
		randomSuffix := fmt.Sprintf("%s-%d", job.Name[len(job.Name)-5:], userID)
		if _, err = serviceManager.CreateNamedIngress(
			ctx,
			[]metav1.OwnerReference{
				*metav1.NewControllerRef(job, batch.SchemeGroupVersion.WithKind("Job")),
			},
			job.Labels,
			port,
			config.GetConfig().Host,
			username,
			randomSuffix,
		); err != nil {
			return fmt.Errorf("failed to create webide ingress: %w", err)
		}
	}

	forwards, err := parseForwards(job.Annotations[annotationKeyForwards])
	if err != nil {
		return err
	}
	if len(forwards) == 0 {
		return nil
	}

	return CreateForwardIngresses(ctx, serviceManager, job, forwards, job.Labels, username)
}

func parseForwards(raw string) ([]Forward, error) {
	if raw == "" {
		return nil, nil
	}

	var forwards []Forward
	if err := json.Unmarshal([]byte(raw), &forwards); err != nil {
		return nil, fmt.Errorf("failed to parse forwards: %w", err)
	}
	return forwards, nil
}
