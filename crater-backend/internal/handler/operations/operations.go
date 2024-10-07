package operations

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/handler"
	"github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/crclient"
	"github.com/samber/lo"
	"gorm.io/datatypes"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
)

type OperationsMgr struct {
	name       string
	nodeClient *crclient.NodeClient
	client.Client
	kubeClient kubernetes.Interface
	mu         sync.Mutex // Add a mutex to protect the ingress creation
}

func NewOperationsMgr(nodeClient *crclient.NodeClient, cl client.Client, kc kubernetes.Interface) handler.Manager {
	return &OperationsMgr{
		name:       "operations",
		nodeClient: nodeClient,
		Client:     cl,
		kubeClient: kc,
	}
}

func (mgr *OperationsMgr) GetName() string {
	return mgr.name
}

func (mgr *OperationsMgr) RegisterPublic(g *gin.RouterGroup) {
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

func (mgr *OperationsMgr) RegisterProtected(_ *gin.RouterGroup) {
}

func (mgr *OperationsMgr) RegisterAdmin(g *gin.RouterGroup) {
	g.POST("/whitelist", mgr.AddJobWhiteList)
	g.GET("/whitelist", mgr.GetWhiteList)
	g.DELETE("/job", mgr.DeleteUnUsedJobList)
}

type JobFreRequest struct {
	TimeRange int `form:"timeRange" binding:"required"`
	Util      int `form:"util" binding:"required"`
}

func (mgr *OperationsMgr) getJobWhiteList(c *gin.Context) ([]string, error) {
	var cleanList []string
	wlDB := query.Whitelist
	data, err := wlDB.WithContext(c).Find()
	if err != nil {
		return nil, err
	}
	for _, item := range data {
		cleanList = append(cleanList, item.Name)
	}
	return cleanList, nil
}

func (mgr *OperationsMgr) deleteJobByName(c *gin.Context, jobName string) error {
	job := &batch.Job{}
	namespace := config.GetConfig().Workspace.Namespace
	if err := mgr.Get(c, client.ObjectKey{Name: jobName, Namespace: namespace}, job); err != nil {
		return err
	}
	baseURL := job.Labels[vcjob.LabelKeyBaseURL]

	j := query.Job
	if _, err := j.WithContext(c).Where(j.JobName.Eq(jobName)).Updates(model.Job{
		Status:             model.Freed,
		CompletedTimestamp: time.Now(),
		Nodes:              datatypes.NewJSONType([]string{}),
	}); err != nil {
		return err
	}

	if err := mgr.Delete(c, job); err != nil {
		return err
	}

	svc := &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
		},
	}
	if err := mgr.Delete(context.Background(), svc); err != nil {
		return err
	}

	mgr.mu.Lock()
	defer mgr.mu.Unlock()
	ingressClient := mgr.kubeClient.NetworkingV1().Ingresses(namespace)

	ingress, err := ingressClient.Get(c, config.GetConfig().Workspace.IngressName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	newPaths := []networkingv1.HTTPIngressPath{}
	for _, path := range ingress.Spec.Rules[0].HTTP.Paths {
		if !strings.Contains(path.Path, baseURL) {
			newPaths = append(newPaths, path)
		}
	}
	ingress.Spec.Rules[0].HTTP.Paths = newPaths

	_, err = ingressClient.Update(context.Background(), ingress, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	return nil
}

// GetWhiteList godoc
// @Summary Get job white list
// @Description get job white list
// @Tags Operations
// @Accept json
// @Produce json
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/whitelist [get]
func (mgr *OperationsMgr) GetWhiteList(c *gin.Context) {
	whiteList, err := mgr.getJobWhiteList(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, whiteList)
}

var newEntries struct {
	Entries []string `json:"white_list"`
}

// AddJobWhiteList godoc
// @Summary Add job white list
// @Description add job white list
// @Tags Operations
// @Accept json
// @Produce json
// @param newEntries body []string true "white list"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/whitelist [post]
func (mgr *OperationsMgr) AddJobWhiteList(c *gin.Context) {
	if err := c.ShouldBindJSON(&newEntries); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.InvalidRequest)
		return
	}
	wlDB := query.Whitelist
	lists := []*model.Whitelist{}
	for _, job := range newEntries.Entries {
		lists = append(lists, &model.Whitelist{Name: job})
	}
	err := wlDB.WithContext(c).CreateInBatches(lists, 2)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "White list updated successfully")
}

// DeleteUnUsedJobList godoc
// @Summary Delete not using gpu job list
// @Description check job list and delete not using gpu job
// @Tags Operations
// @Accept json
// @Produce json
// @Security Bearer
// @Param use query JobFreRequest true "timeRange util"
// @Success 200 {object} resputil.Response[any] "Success"
// @Failure 400 {object} resputil.Response[any] "Request parameter error"
// @Failure 500 {object} resputil.Response[any] "Other errors"
// @Router /v1/operations/job [delete]
func (mgr *OperationsMgr) DeleteUnUsedJobList(c *gin.Context) {
	var req JobFreRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		resputil.HTTPError(c, http.StatusBadRequest, err.Error(), resputil.InvalidRequest)
		return
	}

	unUsedJobs := mgr.nodeClient.GetLeastUsedGPUJobs(req.TimeRange, req.Util)
	whiteList, _ := mgr.getJobWhiteList(c)
	deleteJobList := []string{}

	for _, job := range unUsedJobs {
		if lo.Contains(whiteList, job) {
			fmt.Printf("Job %s is in the white list\n", job)
			continue
		}
		if err := mgr.deleteJobByName(c, job); err != nil {
			fmt.Printf("Delete job %s failed\n", job)
			fmt.Println(err)
		}
		fmt.Printf("Delete job %s successfully\n", job)
		deleteJobList = append(deleteJobList, job)
	}
	response := map[string][]string{
		"delete_job_list": deleteJobList,
	}
	resputil.Success(c, response)
}
