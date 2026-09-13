package vcjob

import (
	"context"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/internal/util"
	"github.com/raids-lab/crater/pkg/config"
)

const (
	workloadKindVolcanoJob       = "volcano-job"
	workloadKindKthenaInference  = "kthena-inference"
	workloadJobTypeModelDeploy   = "model-deployment"
	workloadKthenaManagedByLabel = "crater.raids.io/managed-by"
	workloadKthenaManagedByValue = "inference-service"
	workloadKthenaUserIDLabel    = "crater.raids.io/user-id"
	workloadKthenaAccountIDLabel = "crater.raids.io/account-id"
	workloadKthenaUserAnnotation = "crater.raids.io/user"
	workloadKthenaAccountAnno    = "crater.raids.io/account"

	workloadStatusPending   = "Pending"
	workloadStatusRunning   = "Running"
	workloadStatusCompleted = "Completed"
	workloadStatusFailed    = "Failed"
	workloadPhaseReady      = "Ready"
	workloadConditionActive = "Active"
	workloadConditionTrue   = "True"

	workloadSortID                = "id"
	workloadSortName              = "name"
	workloadSortJobName           = "jobName"
	workloadSortOwner             = "owner"
	workloadSortQueue             = "queue"
	workloadSortJobType           = "jobType"
	workloadSortScheduleType      = "scheduleType"
	workloadFieldStatus           = "status"
	workloadSortBilledPointsTotal = "billedPointsTotal"
	workloadSortCreatedAt         = "createdAt"
	workloadSortStartedAt         = "startedAt"
	workloadSortCompletedAt       = "completedAt"

	workloadFacetJobType      = "job_type"
	workloadFacetKind         = "workload_kind"
	workloadFacetScheduleType = "schedule_type"
	workloadFacetStatus       = "status"
	workloadFacetOwner        = "owner"
	workloadFacetGPUResource  = "gpu_resource"

	workloadFacetCount = 6
)

var workloadModelBoosterListGVK = schema.GroupVersionKind{
	Group:   "workload.serving.volcano.sh",
	Version: "v1alpha1",
	Kind:    "ModelBoosterList",
}

// WorkloadResp is the common row returned by /vcjobs/workloads.  Existing job
// fields are deliberately preserved so the job table can render it directly,
// while workloadKind and detailPath make the Kthena rows unambiguous.
type WorkloadResp struct {
	WorkloadID   string `json:"workloadID"`
	WorkloadKind string `json:"workloadKind"`
	Scheduler    string `json:"scheduler"`
	DetailPath   string `json:"detailPath"`
	StatusDetail string `json:"statusDetail,omitempty"`
	Namespace    string `json:"namespace,omitempty"`
	Model        string `json:"model,omitempty"`

	Name                    string              `json:"name"`
	JobName                 string              `json:"jobName"`
	Owner                   string              `json:"owner"`
	UserInfo                model.UserInfo      `json:"userInfo"`
	JobType                 string              `json:"jobType"`
	ScheduleType            model.ScheduleType  `json:"scheduleType"`
	WaitingToleranceSeconds *int64              `json:"waitingToleranceSeconds,omitempty"`
	Queue                   string              `json:"queue"`
	Status                  string              `json:"status"`
	CreationTimestamp       metav1.Time         `json:"createdAt"`
	RunningTimestamp        metav1.Time         `json:"startedAt"`
	CompletedTimestamp      metav1.Time         `json:"completedAt"`
	Nodes                   []string            `json:"nodes"`
	Resources               corev1.ResourceList `json:"resources"`
	Locked                  bool                `json:"locked"`
	PermanentLocked         bool                `json:"permanentLocked"`
	LockedTimestamp         metav1.Time         `json:"lockedTimestamp"`
	BilledPointsTotal       float64             `json:"billedPointsTotal"`
}

// GetSelfWorkloads godoc
//
//	@Summary	Get current user's unified workloads
//	@Description	Lists persisted Volcano jobs and current-user Kthena ModelBoosters as a single pageable list.
//	@Tags		VolcanoJob
//	@Produce	json
//	@Security	Bearer
//	@Param		page			query	int		false	"Page number"
//	@Param		page_size		query	int		false	"Page size, 1-200"
//	@Param		sort			query	string	false	"Sort fields"
//	@Param		search			query	string	false	"Search workloads"
//	@Param		days			query	int		false	"Number of days to look back, -1 for all"
//	@Param		job_type		query	[]string	false	"Job types, including model-deployment" collectionFormat(multi)
//	@Param		workload_kind	query	[]string	false	"Workload kinds" collectionFormat(multi)
//	@Param		schedule_type	query	[]int		false	"Schedule types" collectionFormat(multi)
//	@Param		status			query	[]string	false	"Workload statuses" collectionFormat(multi)
//	@Param		node			query	string	false	"Node name"
//	@Success	200	{object}	resputil.Response[resputil.Page[WorkloadResp]]
//	@Router		/v1/vcjobs/workloads [get]
func (mgr *VolcanojobMgr) GetSelfWorkloads(c *gin.Context) {
	token := util.GetToken(c)
	mgr.listWorkloads(c, -1, jobListScope{UserID: &token.UserID, AccountID: &token.AccountID})
}

// GetSelfWorkloadFacets godoc
//
//	@Summary	Get current user's unified workload facets
//	@Tags		VolcanoJob
//	@Produce	json
//	@Security	Bearer
//	@Success	200	{object}	resputil.Response[resputil.FacetResponse]
//	@Router		/v1/vcjobs/workloads/facets [get]
func (mgr *VolcanojobMgr) GetSelfWorkloadFacets(c *gin.Context) {
	token := util.GetToken(c)
	mgr.listWorkloadFacets(c, -1, jobListScope{UserID: &token.UserID, AccountID: &token.AccountID}, false)
}

// The admin variants match the existing /vcjobs/all semantics.  They are also
// useful to administrative tables without changing the legacy APIs.
func (mgr *VolcanojobMgr) GetAllWorkloads(c *gin.Context) {
	mgr.listWorkloads(c, allJobsDefaultDays, jobListScope{})
}

func (mgr *VolcanojobMgr) GetAllWorkloadFacets(c *gin.Context) {
	mgr.listWorkloadFacets(c, allJobsDefaultDays, jobListScope{}, true)
}

func (mgr *VolcanojobMgr) GetUserWorkloads(c *gin.Context) {
	scope, ok := resolveJobUserScope(c)
	if !ok {
		return
	}
	mgr.listWorkloads(c, userJobsDefaultDays, scope)
}

func (mgr *VolcanojobMgr) GetUserWorkloadFacets(c *gin.Context) {
	scope, ok := resolveJobUserScope(c)
	if !ok {
		return
	}
	mgr.listWorkloadFacets(c, userJobsDefaultDays, scope, false)
}

func (mgr *VolcanojobMgr) listWorkloads(c *gin.Context, defaultDays int, scope jobListScope) {
	request, err := bindJobListQuery(c, true)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	workloads, err := mgr.findWorkloads(c.Request.Context(), scope, &request, defaultDays)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "list workloads failed"))
		return
	}

	total := len(workloads)
	start := request.offset()
	if start > total {
		start = total
	}
	end := start + request.PageSize
	if end > total {
		end = total
	}
	resputil.Success(c, resputil.NewPage(workloads[start:end], int64(total), request.Page, request.PageSize))
}

func (mgr *VolcanojobMgr) listWorkloadFacets(
	c *gin.Context,
	defaultDays int,
	scope jobListScope,
	includeOverview bool,
) {
	request, err := bindJobListQuery(c, false)
	if err != nil {
		resputil.HandleError(c, err)
		return
	}
	facets, err := mgr.findWorkloadFacets(c.Request.Context(), scope, &request, defaultDays, includeOverview)
	if err != nil {
		resputil.HandleError(c, bizerr.Internal.ServiceError.Wrap(err, "list workload facets failed"))
		return
	}
	resputil.Success(c, resputil.FacetResponse{Facets: facets})
}

func (mgr *VolcanojobMgr) findWorkloads(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) ([]WorkloadResp, error) {
	workloads := make([]WorkloadResp, 0)
	if includesWorkloadKind(request.WorkloadKinds, workloadKindVolcanoJob) && includesVolcanoJobType(request.JobTypes) {
		jobs, err := findWorkloadJobs(ctx, scope, request, defaultDays)
		if err != nil {
			return nil, err
		}
		for _, job := range jobs {
			workloads = append(workloads, workloadFromJob(job))
		}
	}
	if mgr.isKthenaInferenceEnabled(ctx) &&
		includesWorkloadKind(request.WorkloadKinds, workloadKindKthenaInference) &&
		includesKthenaJobType(request.JobTypes) {
		inference, err := mgr.findKthenaWorkloads(ctx, scope, request, defaultDays)
		if err != nil {
			return nil, err
		}
		workloads = append(workloads, inference...)
	}
	sortWorkloads(workloads, request.sorts)
	return workloads, nil
}

// isKthenaInferenceEnabled keeps unified workload surfaces consistent with the
// model-deployment feature gate. A missing ConfigService is deliberately
// treated as disabled: Kthena is an opt-in capability for existing installs.
func (mgr *VolcanojobMgr) isKthenaInferenceEnabled(ctx context.Context) bool {
	return mgr.configService != nil && mgr.configService.IsKthenaInferenceEnabled(ctx)
}

func findWorkloadJobs(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) ([]*model.Job, error) {
	jobRequest := *request
	jobRequest.JobTypes = regularJobTypes(request.JobTypes)
	return applyJobFilters(ctx, scope, &jobRequest, defaultDays).
		Preload(query.Job.User).
		Preload(query.Job.Account).
		Find()
}

func (mgr *VolcanojobMgr) findKthenaWorkloads(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
) ([]WorkloadResp, error) {
	// ModelBoosters do not expose a useful node until a runtime pod exists. A
	// node filter therefore must not return a false-positive inference row.
	if request.Node != nil || mgr.client == nil {
		return []WorkloadResp{}, nil
	}

	labels := client.MatchingLabels{workloadKthenaManagedByLabel: workloadKthenaManagedByValue}
	if scope.UserID != nil {
		labels[workloadKthenaUserIDLabel] = strconv.FormatUint(uint64(*scope.UserID), 10)
	}
	if scope.AccountID != nil {
		labels[workloadKthenaAccountIDLabel] = strconv.FormatUint(uint64(*scope.AccountID), 10)
	}

	list := &unstructured.UnstructuredList{}
	list.SetGroupVersionKind(workloadModelBoosterListGVK)
	namespace := mgr.workloadNamespace
	if namespace == "" {
		namespace = config.GetConfig().Namespaces.Job
	}
	if err := mgr.client.List(ctx, list, client.InNamespace(namespace), labels); err != nil {
		// Kthena is optional for an existing Crater installation. In that case a
		// unified list remains useful for normal Volcano jobs.
		if meta.IsNoMatchError(err) || apierrors.IsNotFound(err) {
			return []WorkloadResp{}, nil
		}
		return nil, err
	}

	result := make([]WorkloadResp, 0, len(list.Items))
	for i := range list.Items {
		workload := workloadFromModelBooster(&list.Items[i])
		if workloadMatchesQuery(&workload, request, defaultDays) {
			result = append(result, workload)
		}
	}
	return result, nil
}

func workloadFromJob(job *model.Job) WorkloadResp {
	response := convertJobResp([]*model.Job{job})[0]
	return WorkloadResp{
		WorkloadID:              workloadKindVolcanoJob + ":" + response.JobName,
		WorkloadKind:            workloadKindVolcanoJob,
		Scheduler:               VolcanoSchedulerName,
		DetailPath:              "/portal/jobs/detail/" + url.PathEscape(response.JobName),
		Name:                    response.Name,
		JobName:                 response.JobName,
		Owner:                   response.Owner,
		UserInfo:                response.UserInfo,
		JobType:                 response.JobType,
		ScheduleType:            response.ScheduleType,
		WaitingToleranceSeconds: response.WaitingToleranceSeconds,
		Queue:                   response.Queue,
		Status:                  response.Status,
		CreationTimestamp:       response.CreationTimestamp,
		RunningTimestamp:        response.RunningTimestamp,
		CompletedTimestamp:      response.CompletedTimestamp,
		Nodes:                   response.Nodes,
		Resources:               response.Resources,
		Locked:                  response.Locked,
		PermanentLocked:         response.PermanentLocked,
		LockedTimestamp:         response.LockedTimestamp,
		BilledPointsTotal:       response.BilledPointsTotal,
	}
}

func workloadFromModelBooster(obj *unstructured.Unstructured) WorkloadResp {
	backend, _, _ := unstructured.NestedMap(obj.Object, "spec", "backend")
	worker := firstKthenaWorker(backend)
	annotations := obj.GetAnnotations()
	owner := strings.TrimSpace(annotations[workloadKthenaUserAnnotation])
	queue := strings.TrimSpace(annotations[workloadKthenaAccountAnno])
	phase := kthenaModelBoosterPhase(obj)
	scheduler := strings.TrimSpace(stringFromMap(backend, "schedulerName"))
	if scheduler == "" {
		scheduler = VolcanoSchedulerName
	}
	modelName := servedModelName(backend, worker)

	return WorkloadResp{
		WorkloadID:        workloadKindKthenaInference + ":" + obj.GetName(),
		WorkloadKind:      workloadKindKthenaInference,
		Scheduler:         scheduler,
		DetailPath:        "/portal/inference-services/" + url.PathEscape(obj.GetName()),
		StatusDetail:      phase,
		Namespace:         obj.GetNamespace(),
		Model:             modelName,
		Name:              obj.GetName(),
		JobName:           obj.GetName(),
		Owner:             owner,
		UserInfo:          model.UserInfo{Username: owner},
		JobType:           workloadJobTypeModelDeploy,
		ScheduleType:      model.ScheduleTypeNormal,
		Queue:             queue,
		Status:            kthenaPhaseToJobStatus(phase),
		CreationTimestamp: metav1.NewTime(obj.GetCreationTimestamp().Time),
		Nodes:             []string{},
		Resources:         kthenaWorkerResources(backend, worker),
		LockedTimestamp:   metav1.NewTime(time.Time{}),
		BilledPointsTotal: 0,
	}
}

func firstKthenaWorker(backend map[string]any) map[string]any {
	workers, _ := backend["workers"].([]any)
	if len(workers) == 0 {
		return map[string]any{}
	}
	worker, _ := workers[0].(map[string]any)
	if worker == nil {
		return map[string]any{}
	}
	return worker
}

func servedModelName(backend, worker map[string]any) string {
	if configMap, ok := worker["config"].(map[string]any); ok {
		if served := strings.TrimSpace(stringFromMap(configMap, "served-model-name")); served != "" {
			return served
		}
	}
	return strings.TrimSpace(stringFromMap(backend, "modelURI"))
}

func kthenaModelBoosterPhase(obj *unstructured.Unstructured) string {
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")
	if phase = strings.TrimSpace(phase); phase != "" {
		return phase
	}
	conditions, _, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	for _, raw := range conditions {
		condition, _ := raw.(map[string]any)
		if strings.EqualFold(stringFromMap(condition, "type"), workloadStatusFailed) &&
			strings.EqualFold(stringFromMap(condition, "status"), workloadConditionTrue) {
			return workloadStatusFailed
		}
		if strings.EqualFold(stringFromMap(condition, "type"), workloadConditionActive) &&
			strings.EqualFold(stringFromMap(condition, "status"), workloadConditionTrue) {
			return workloadPhaseReady
		}
	}
	return workloadStatusPending
}

func kthenaPhaseToJobStatus(phase string) string {
	switch strings.ToLower(strings.TrimSpace(phase)) {
	case "ready", "active", "running":
		return workloadStatusRunning
	case "completed", "succeeded", "terminated":
		return workloadStatusCompleted
	case "failed", "error", "degraded":
		return workloadStatusFailed
	default:
		return workloadStatusPending
	}
}

func kthenaWorkerResources(backend, worker map[string]any) corev1.ResourceList {
	resourceMap, _ := worker["resources"].(map[string]any)
	requests, _ := resourceMap["requests"].(map[string]any)
	limits, _ := resourceMap["limits"].(map[string]any)
	// Crater writes CPU and memory to requests, but accelerator resources to
	// limits (the conventional Kubernetes representation).  Merge both maps
	// so the unified workload row shows the actual GPU allocation as well.
	resourceValues := make(map[string]any, len(requests)+len(limits))
	for name, raw := range limits {
		resourceValues[name] = raw
	}
	for name, raw := range requests {
		resourceValues[name] = raw
	}
	resources := make(corev1.ResourceList, len(resourceValues))
	multiplier := positiveKthenaInt(backend["replicas"]) *
		positiveKthenaInt(worker["replicas"]) * positiveKthenaInt(worker["pods"])
	for name, raw := range resourceValues {
		quantity, err := resource.ParseQuantity(strings.TrimSpace(valueString(raw)))
		if err != nil {
			continue
		}
		quantity.Mul(multiplier)
		resources[corev1.ResourceName(name)] = quantity
	}
	return resources
}

func positiveKthenaInt(value any) int64 {
	switch typed := value.(type) {
	case int64:
		if typed > 0 {
			return typed
		}
	case int:
		if typed > 0 {
			return int64(typed)
		}
	case float64:
		if typed > 0 {
			return int64(typed)
		}
	case string:
		if parsed, err := strconv.ParseInt(typed, 10, 64); err == nil && parsed > 0 {
			return parsed
		}
	}
	return 1
}

func stringFromMap(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	return valueString(values[key])
}

func valueString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func workloadMatchesQuery(workload *WorkloadResp, request *jobListQuery, defaultDays int) bool {
	if !includesWorkloadKind(request.WorkloadKinds, workload.WorkloadKind) ||
		(len(request.JobTypes) > 0 && !containsString(request.JobTypes, workload.JobType)) ||
		(len(request.ScheduleTypes) > 0 && !containsScheduleType(request.ScheduleTypes, workload.ScheduleType)) ||
		(len(request.Statuses) > 0 && !containsString(request.Statuses, workload.Status)) {
		return false
	}
	if days := request.days(defaultDays); days != -1 && workload.CreationTimestamp.Time.Before(time.Now().AddDate(0, 0, -days)) {
		return false
	}
	if request.Search == "" {
		return true
	}
	needle := strings.ToLower(request.Search)
	searchableValues := []string{
		workload.Name,
		workload.JobName,
		workload.Owner,
		workload.UserInfo.Nickname,
		workload.Queue,
		workload.Model,
	}
	for _, value := range searchableValues {
		if strings.Contains(strings.ToLower(value), needle) {
			return true
		}
	}
	return false
}

func includesWorkloadKind(kinds []string, kind string) bool {
	return len(kinds) == 0 || containsString(kinds, kind)
}

func includesVolcanoJobType(jobTypes []string) bool {
	if len(jobTypes) == 0 {
		return true
	}
	for _, jobType := range jobTypes {
		if jobType != workloadJobTypeModelDeploy {
			return true
		}
	}
	return false
}

func includesKthenaJobType(jobTypes []string) bool {
	return len(jobTypes) == 0 || containsString(jobTypes, workloadJobTypeModelDeploy)
}

func regularJobTypes(jobTypes []string) []string {
	if len(jobTypes) == 0 {
		return nil
	}
	result := make([]string, 0, len(jobTypes))
	for _, jobType := range jobTypes {
		if jobType != workloadJobTypeModelDeploy {
			result = append(result, jobType)
		}
	}
	return result
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsScheduleType(values []int, want model.ScheduleType) bool {
	for _, value := range values {
		if value == int(want) {
			return true
		}
	}
	return false
}

func sortWorkloads(workloads []WorkloadResp, sorts []jobSort) {
	sort.SliceStable(workloads, func(i, j int) bool {
		for _, item := range sorts {
			comparison := compareWorkloads(&workloads[i], &workloads[j], item.field)
			if comparison == 0 {
				continue
			}
			if item.descending {
				return comparison > 0
			}
			return comparison < 0
		}
		return workloads[i].WorkloadID < workloads[j].WorkloadID
	})
}

func compareWorkloads(left, right *WorkloadResp, field string) int {
	switch field {
	case workloadSortCreatedAt:
		return left.CreationTimestamp.Compare(right.CreationTimestamp.Time)
	case workloadSortStartedAt:
		return left.RunningTimestamp.Compare(right.RunningTimestamp.Time)
	case workloadSortCompletedAt:
		return left.CompletedTimestamp.Compare(right.CompletedTimestamp.Time)
	case workloadSortScheduleType:
		return int(left.ScheduleType) - int(right.ScheduleType)
	case workloadSortBilledPointsTotal:
		return compareFloat(left.BilledPointsTotal, right.BilledPointsTotal)
	case workloadSortName:
		return strings.Compare(left.Name, right.Name)
	case workloadSortJobName:
		return strings.Compare(left.JobName, right.JobName)
	case workloadSortOwner:
		return strings.Compare(left.Owner, right.Owner)
	case workloadSortQueue:
		return strings.Compare(left.Queue, right.Queue)
	case workloadSortJobType:
		return strings.Compare(left.JobType, right.JobType)
	case workloadFieldStatus:
		return strings.Compare(left.Status, right.Status)
	case workloadSortID:
		return strings.Compare(left.WorkloadID, right.WorkloadID)
	default:
		return 0
	}
}

func compareFloat(left, right float64) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func (mgr *VolcanojobMgr) findWorkloadFacets(
	ctx context.Context,
	scope jobListScope,
	request *jobListQuery,
	defaultDays int,
	includeOverview bool,
) (map[string][]resputil.FacetItem, error) {
	result := make(map[string][]resputil.FacetItem, workloadFacetCount)
	for _, facet := range []string{workloadFacetJobType, workloadFacetKind, workloadFacetScheduleType, workloadFacetStatus} {
		facetRequest := jobFacetQuery(request, facet)
		workloads, err := mgr.findWorkloads(ctx, scope, &facetRequest, defaultDays)
		if err != nil {
			return nil, err
		}
		switch facet {
		case workloadFacetJobType:
			result[facet] = workloadStringFacet(workloads, func(workload *WorkloadResp) string { return workload.JobType })
		case workloadFacetKind:
			result[facet] = workloadStringFacet(workloads, func(workload *WorkloadResp) string { return workload.WorkloadKind })
		case workloadFacetScheduleType:
			result[facet] = workloadStringFacet(workloads, func(workload *WorkloadResp) string {
				return strconv.Itoa(int(workload.ScheduleType))
			})
		case workloadFacetStatus:
			result[facet] = workloadStringFacet(workloads, func(workload *WorkloadResp) string { return workload.Status })
		}
	}
	if !includeOverview {
		return result, nil
	}
	runningRequest := *request
	runningRequest.Statuses = []string{workloadStatusRunning}
	running, err := mgr.findWorkloads(ctx, scope, &runningRequest, defaultDays)
	if err != nil {
		return nil, err
	}
	result[workloadFacetOwner] = workloadStringFacet(running, func(workload *WorkloadResp) string { return workload.Owner })
	result[workloadFacetGPUResource] = workloadGPUFacet(running)
	return result, nil
}

func workloadStringFacet(workloads []WorkloadResp, value func(*WorkloadResp) string) []resputil.FacetItem {
	counts := make(map[string]int64)
	for i := range workloads {
		if item := value(&workloads[i]); item != "" {
			counts[item]++
		}
	}
	return sortedFacetItems(counts)
}

func workloadGPUFacet(workloads []WorkloadResp) []resputil.FacetItem {
	counts := make(map[string]int64)
	for i := range workloads {
		for name, quantity := range workloads[i].Resources {
			if !strings.Contains(strings.ToLower(string(name)), "gpu") {
				continue
			}
			counts[strings.TrimPrefix(string(name), "nvidia.com/")] += quantity.Value()
		}
	}
	return sortedFacetItems(counts)
}

func sortedFacetItems(counts map[string]int64) []resputil.FacetItem {
	items := make([]resputil.FacetItem, 0, len(counts))
	for value, count := range counts {
		items = append(items, resputil.FacetItem{Value: value, Count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Count == items[j].Count {
			return items[i].Value < items[j].Value
		}
		return items[i].Count > items[j].Count
	})
	return items
}
