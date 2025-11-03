package vcjob

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/aitaskctl"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/raids-lab/crater/pkg/packer"
	"github.com/raids-lab/crater/pkg/utils"
)

const (
	ThreeDaySeconds    int32 = 259200
	IngressLabelPrefix       = "ingress.crater.raids.io/" // Annotation Ingress Key
	NodePortLabelKey         = "nodeport.crater.raids.io/"
)

type (
	CreateJupyterReq struct {
		CreateJobCommon `json:",inline"`
		Resource        v1.ResourceList `json:"resource"`
		Image           ImageBaseInfo   `json:"image" binding:"required"`
	}
)

// CreateJupyterJob godoc
//
//	@Summary		Create a Jupyter job
//	@Description	Create a Jupyter job
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			CreateJupyterReq	body		CreateJupyterReq		true	"Create Jupyter Job Request"
//	@Success		200					{object}	resputil.Response[any]	"Success"
//	@Failure		400					{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500					{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/vcjobs/jupyter [post]
func (mgr *VolcanojobMgr) CreateJupyterJob(c *gin.Context) {
	token := util.GetToken(c)

	var req CreateJupyterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	if err := aitaskctl.CheckJupyterLimitBeforeCreateJupyter(c, token.UserID, token.AccountID); err != nil {
		resputil.Error(c, err.Error(), resputil.ServiceError)
		return
	}

	exceededResources := aitaskctl.CheckResourcesBeforeCreateJob(c, token.UserID, token.AccountID, req.Resource)
	if len(exceededResources) > 0 {
		resputil.Error(c, fmt.Sprintf("%v", exceededResources), resputil.ServiceError)
		return
	}

	// 如果希望接受邮件，则需要确保邮箱已验证
	if req.AlertEnabled && !utils.CheckUserEmail(c, token.UserID) {
		resputil.Error(c, "Email not verified", resputil.UserEmailNotVerified)
		return
	}

	// Ingress base URL
	baseURL := fmt.Sprintf("%s-%s", token.Username, uuid.New().String()[:5])
	jobName := fmt.Sprintf("jupyter-%s", baseURL)

	// Command to start Jupyter
	var commandSchema string
	var command string

	// 1. Volume Mounts
	volumes, volumeMounts, err := GenerateVolumeMounts(c, req.VolumeMounts, token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// 1.1 Configure jupyter images
	if strings.Contains(req.Image.ImageLink, "jupyter") {
		commandSchema = "start.sh jupyter lab --allow-root " +
			"--notebook-dir=/home/%s " +
			"--NotebookApp.base_url=/ingress/%s/ " +
			"--ResourceUseDisplay.track_cpu_percent=True"
		command = fmt.Sprintf(commandSchema, token.Username, baseURL)
	} else {
		var startScriptConfigMap string
		var jupyterPath string
		if strings.Contains(req.Image.ImageLink, "envd") {
			// 1.2 Configure envd images
			startScriptConfigMap = "envd-jupyter-start-configmap"
			jupyterPath = "/opt/conda/envs/envd/bin/jupyter"
		} else {
			// 1.3 Configure NGC images
			startScriptConfigMap = "jupyter-start-configmap"
			jupyterPath = "jupyter"
		}
		volumes = append(volumes, v1.Volume{
			Name: "bash-script-volume",
			VolumeSource: v1.VolumeSource{
				ConfigMap: &v1.ConfigMapVolumeSource{
					LocalObjectReference: v1.LocalObjectReference{
						Name: startScriptConfigMap,
					},
					//nolint:mnd // 0755 is the default mode
					DefaultMode: ptr.To(int32(0755)),
				},
			},
		})
		volumeMounts = append(volumeMounts, v1.VolumeMount{
			Name:      "bash-script-volume",
			MountPath: "/usr/bin/start.sh",
			ReadOnly:  true,
			SubPath:   "start.sh",
		})

		commandSchema = "/usr/bin/start.sh %s lab --ip=0.0.0.0 --no-browser --allow-root " +
			"--notebook-dir=/home/%s --NotebookApp.base_url=/ingress/%s/ "
		command = fmt.Sprintf(commandSchema, jupyterPath, token.Username, baseURL)
	}

	// 2. Env Vars
	envs := GenerateEnvs(c, token, req.Envs)
	envs = append(
		envs,
		v1.EnvVar{Name: "GRANT_SUDO", Value: "1"},
		v1.EnvVar{Name: "CHOWN_HOME", Value: "1"},
	)

	// 3. Node Affinity and Tolerations
	baseAffinity := GenerateNodeAffinity(req.Selectors, req.Resource)
	affinity := GenerateArchitectureNodeAffinity(req.Image, baseAffinity)

	tolerations := GenerateTaintTolerationsForAccount(token)

	// 4. Labels and Annotations
	labels, jobAnnotations, podAnnotations := getLabelAndAnnotations(
		CraterJobTypeJupyter,
		token,
		baseURL,
		req.Name,
		req.Template,
		req.AlertEnabled,
	)

	imagePullSecrets := []v1.LocalObjectReference{}
	if config.GetConfig().Secrets.ImagePullSecretName != "" {
		imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{
			Name: config.GetConfig().Secrets.ImagePullSecretName,
		})
	}

	// 5. Create the pod spec
	podSpec := v1.PodSpec{
		Affinity:         affinity,
		Tolerations:      tolerations,
		Volumes:          volumes,
		ImagePullSecrets: imagePullSecrets,
		Containers: []v1.Container{
			{
				Name:    string(CraterJobTypeJupyter),
				Image:   req.Image.ImageLink,
				Command: []string{"bash", "-c", command},
				Resources: v1.ResourceRequirements{
					Limits:   req.Resource,
					Requests: req.Resource,
				},
				WorkingDir: fmt.Sprintf("/home/%s", token.Username),

				Env: envs,
				Ports: []v1.ContainerPort{
					{ContainerPort: JupyterPort, Name: "notebook", Protocol: v1.ProtocolTCP},
				},
				SecurityContext: &v1.SecurityContext{
					RunAsUser:  ptr.To(int64(0)),
					RunAsGroup: ptr.To(int64(0)),
				},
				TerminationMessagePath:   "/dev/termination-log",
				TerminationMessagePolicy: v1.TerminationMessageReadFile,
				VolumeMounts:             volumeMounts,
			},
		},
		RestartPolicy:      v1.RestartPolicyNever,
		EnableServiceLinks: ptr.To(false),
	}

	fmt.Printf("Affinity:\n %+v\n", podSpec.Affinity)

	// 6. Create volcano job
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:        jobName,
			Namespace:   config.GetConfig().Namespaces.Job,
			Labels:      labels,
			Annotations: jobAnnotations,
		},
		Spec: batch.JobSpec{
			// 3 days
			TTLSecondsAfterFinished: ptr.To(ThreeDaySeconds),
			MinAvailable:            1,
			MaxRetry:                1,
			SchedulerName:           VolcanoSchedulerName,
			Queue:                   token.AccountName,
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
			},
			Tasks: []batch.TaskSpec{
				{
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      labels,
							Annotations: podAnnotations,
						},
						Spec: podSpec,
					},
				},
			},
		},
	}

	if err = mgr.client.Create(c, &job); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// create jupyter notebook ingress
	port := &v1.ServicePort{
		Name:       "notebook",
		Port:       JupyterPort,
		TargetPort: intstr.FromInt(JupyterPort),
		Protocol:   v1.ProtocolTCP,
	}

	ingressPath, err := mgr.serviceManager.CreateIngressWithPrefix(
		c,
		[]metav1.OwnerReference{
			*metav1.NewControllerRef(&job, batch.SchemeGroupVersion.WithKind("Job")),
		},
		labels,
		port,
		config.GetConfig().Host,
		baseURL,
	)
	if err != nil {
		resputil.Error(c, fmt.Sprintf("failed to create ingress: %v", err), resputil.NotSpecified)
		return
	}

	log.Printf("Ingress created at path: %s", ingressPath)

	// create forward ing rules in template
	//nolint:dupl // ignore duplicate code
	for _, forward := range req.Forwards {
		port := &v1.ServicePort{
			Name:       forward.Name,
			Port:       forward.Port,
			TargetPort: intstr.FromInt(int(forward.Port)),
			Protocol:   v1.ProtocolTCP,
		}

		ingressPath, err := mgr.serviceManager.CreateIngress(
			c,
			[]metav1.OwnerReference{
				*metav1.NewControllerRef(&job, batch.SchemeGroupVersion.WithKind("Job")),
			},
			labels,
			port,
			config.GetConfig().Host,
			token.Username,
		)
		if err != nil {
			resputil.Error(c, fmt.Sprintf("failed to create ingress for %s: %v", forward.Name, err), resputil.NotSpecified)
			return
		}
		fmt.Printf("Ingress created for %s at path: %s\n", forward.Name, ingressPath)
	}

	resputil.Success(c, job)
}

// GetJobToken godoc
//
//	@Summary		Get the ingress base url and jupyter token of the job
//	@Description	Get the token of the job by logs
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			jobName	path		string							true	"Job Name"
//	@Success		200		{object}	resputil.Response[JobTokenResp]	"Success"
//	@Failure		400		{object}	resputil.Response[any]			"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]			"Other errors"
//	@Router			/v1/vcjobs/{name}/token [get]
func (mgr *VolcanojobMgr) GetJobToken(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	if job.JobType != model.JobTypeJupyter {
		resputil.Error(c, "Job type is not Jupyter", resputil.NotSpecified)
		return
	}

	vcjob := &batch.Job{}
	namespace := config.GetConfig().Namespaces.Job
	if err = mgr.client.Get(c, client.ObjectKey{Name: req.JobName, Namespace: namespace}, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	// Check if the job is running
	status := vcjob.Status.State.Phase
	if status != batch.Running {
		resputil.Error(c, "Job not running", resputil.NotSpecified)
		return
	}

	baseURL := vcjob.Labels[crclient.LabelKeyBaseURL]

	podName, _ := getPodNameAndLabelFromJob(vcjob)

	// Construct the full URL directly
	host := config.GetConfig().Host
	fullURL := fmt.Sprintf("https://%s/ingress/%s", host, baseURL)

	// Check if jupyter token has been cached in the job annotations
	jupyterToken, ok := vcjob.Annotations[AnnotationKeyJupyter]
	if ok {
		resputil.Success(c, JobTokenResp{
			BaseURL:   baseURL,
			Token:     jupyterToken,
			FullURL:   fullURL,
			PodName:   podName,
			Namespace: namespace,
		})
		return
	}

	// Fetch the pod logs to extract the token
	buf, err := mgr.getPodLog(c, namespace, podName)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	re := regexp.MustCompile(`\?token=([a-zA-Z0-9]+)`)
	matches := re.FindStringSubmatch(buf.String())
	if len(matches) >= 2 {
		jupyterToken = matches[1]
	} else {
		resputil.Error(c, "Jupyter token not found", resputil.NotSpecified)
		return
	}

	// Cache the jupyter token in the job annotations
	vcjob.Annotations[AnnotationKeyJupyter] = jupyterToken
	if err := mgr.client.Update(c, vcjob); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, JobTokenResp{
		BaseURL:   baseURL,
		Token:     jupyterToken,
		FullURL:   fullURL,
		PodName:   podName,
		Namespace: namespace,
	})
}

// CreateJupyterSnapshot godoc
//
//	@Summary		Create a snapshot of the jupyter notebook
//	@Description	Create nerdctl docker commit to snapshot the jupyter notebook
//	@Tags			VolcanoJob
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			name	path		string							true	"Job Name"
//	@Success		200		{object}	resputil.Response[JobTokenResp]	"Success"
//	@Failure		400		{object}	resputil.Response[any]			"Request parameter error"
//	@Failure		500		{object}	resputil.Response[any]			"Other errors"
//	@Router			/v1/vcjobs/jupyter/{name}/snapshot [post]
func (mgr *VolcanojobMgr) CreateJupyterSnapshot(c *gin.Context) {
	var req JobActionReq
	if err := c.ShouldBindUri(&req); err != nil {
		resputil.BadRequestError(c, err.Error())
		return
	}

	token := util.GetToken(c)

	// find from db
	job, err := getJob(c, req.JobName, &token)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	vcjob := job.Attributes.Data()
	nodes := job.Nodes.Data()

	if len(nodes) != 1 {
		resputil.Error(c, "invalid node", resputil.NotSpecified)
		return
	}

	nodeName := nodes[0]

	// get pod events
	var podList = &v1.PodList{}
	if value, ok := vcjob.Labels[crclient.LabelKeyBaseURL]; !ok {
		resputil.Error(c, "label not found", resputil.NotSpecified)
		return
	} else {
		labels := client.MatchingLabels{crclient.LabelKeyBaseURL: value}
		err = mgr.client.List(c, podList, client.InNamespace(vcjob.Namespace), labels)
		if err != nil {
			resputil.Error(c, err.Error(), resputil.NotSpecified)
			return
		}
	}

	if len(podList.Items) != 1 {
		resputil.Error(c, "invalid pod", resputil.NotSpecified)
		return
	}

	pod := podList.Items[0]
	if pod.Status.Phase != v1.PodRunning {
		resputil.Error(c, "pod not running", resputil.NotSpecified)
		return
	}

	// get container name
	if len(pod.Spec.Containers) != 1 {
		resputil.Error(c, "invalid container", resputil.NotSpecified)
		return
	}

	containerName := pod.Spec.Containers[0].Name

	// check whether user project exists
	if err = mgr.imageRegistry.CheckOrCreateProjectForUser(c, token.Username); err != nil {
		resputil.Error(c, "create harbor project failed", resputil.NotSpecified)
		return
	}

	// generate image link
	currentImageName := pod.Spec.Containers[0].Image
	imageLink, err := utils.GenerateNewImageLinkForDockerfileBuild(currentImageName, token.Username, "", "")
	if err != nil {
		resputil.Error(c, "generate new image link failed", resputil.NotSpecified)
		return
	}

	err = mgr.imagePacker.CreateFromSnapshot(c, &packer.SnapshotReq{
		UserID:        token.UserID,
		Namespace:     vcjob.Namespace,
		PodName:       pod.Name,
		ContainerName: containerName,
		NodeName:      nodeName,
		Description:   fmt.Sprintf("Snapshot of %s", job.JobName),
		ImageLink:     imageLink,
		BuildSource:   model.Snapshot,
	})
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}

	resputil.Success(c, imageLink)
}
