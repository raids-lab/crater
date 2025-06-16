package operations

import (
	"fmt"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/raids-lab/crater/internal/resputil"
	"github.com/raids-lab/crater/pkg/logutils"
)

const (
	CRONJOBNAMESPACE  = "crater"
	CRAONJOBLABELKEY  = "crater.raids-lab.io/component"
	CRONJOBLABELVALUE = "cronjob"
)

type CronjobConfigs struct {
	Name     string            `json:"name"`
	Schedule string            `json:"schedule"`
	Suspend  bool              `json:"suspend"`
	Configs  map[string]string `json:"configs"`
}

// UpdateCronjobConfig godoc
//
//	@Summary		Update cronjob config
//	@Description	Update one cronjob config
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Param			use	body		CronjobConfigs			true	"CronjobConfigs"
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/operations/cronjob [put]
func (mgr *OperationsMgr) UpdateCronjobConfig(c *gin.Context) {
	var req CronjobConfigs
	if err := c.ShouldBindJSON(&req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	fmt.Println(req)
	if err := mgr.updateCronjobConfig(c, req); err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, "Successfully update cronjob config")
}

func (mgr *OperationsMgr) updateCronjobConfig(c *gin.Context, cronjobConfigs CronjobConfigs) error {
	namespace := CRONJOBNAMESPACE
	cronjob, err := mgr.kubeClient.BatchV1().CronJobs(namespace).Get(c, cronjobConfigs.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get cronjob %s: %w", cronjobConfigs.Name, err)
	}

	// 更新 schedule
	cronjob.Spec.Schedule = cronjobConfigs.Schedule

	// 更新 suspend 字段
	*cronjob.Spec.Suspend = cronjobConfigs.Suspend

	// CronJob 确保只有一个 container
	if len(cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers) == 0 {
		return fmt.Errorf("cronjob %s has no container", cronjobConfigs.Name)
	}
	container := &cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers[0]

	// 检查并更新 env 变量，确保传入的 env 必须已经存在
	for key, newVal := range cronjobConfigs.Configs {
		found := false
		for idx, env := range container.Env {
			if env.Name == key {
				container.Env[idx].Value = newVal
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("container %s missing env key: %s", container.Name, key)
		}
	}

	// 更新 CronJob 对象
	_, err = mgr.kubeClient.BatchV1().CronJobs(namespace).Update(c, cronjob, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update cronjob %s: %w", cronjobConfigs.Name, err)
	}
	return nil
}

// GetCronjobConfigs godoc
//
//	@Summary		Get all cronjob configs
//	@Description	Get all cronjob configs
//	@Tags			Operations
//	@Accept			json
//	@Produce		json
//	@Security		Bearer
//	@Success		200	{object}	resputil.Response[any]	"Success"
//	@Failure		400	{object}	resputil.Response[any]	"Request parameter error"
//	@Failure		500	{object}	resputil.Response[any]	"Other errors"
//	@Router			/v1/operations/cronjob [get]
func (mgr *OperationsMgr) GetCronjobConfigs(c *gin.Context) {
	configs, err := mgr.getCronjobConfigs(c)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, configs)
}

func (mgr *OperationsMgr) getCronjobConfigs(c *gin.Context) ([]CronjobConfigs, error) {
	// get all cronjobs in namespace crater, label crater.raids-lab.io/component=cronjob
	// apiVersion: batch/v1, kind: CronJob
	cronjobConfigList := make([]CronjobConfigs, 0)
	namespace := CRONJOBNAMESPACE
	labelSelector := fmt.Sprintf("%s=%s", CRAONJOBLABELKEY, CRONJOBLABELVALUE)
	cronjobs, err := mgr.kubeClient.BatchV1().CronJobs(namespace).List(c, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		logutils.Log.Errorf("Failed to get cronjobs: %v", err)
		return nil, err
	}
	for i := range cronjobs.Items {
		cronjob := &cronjobs.Items[i]
		configs := make(map[string]string)

		// cronjob 确保只有一个 container
		containers := cronjob.Spec.JobTemplate.Spec.Template.Spec.Containers
		if len(containers) != 1 {
			continue
		}
		container := containers[0]
		for _, env := range container.Env {
			configs[env.Name] = env.Value
		}
		cronjobConfigList = append(cronjobConfigList, CronjobConfigs{
			Name:     cronjob.Name,
			Configs:  configs,
			Suspend:  *cronjob.Spec.Suspend,
			Schedule: cronjob.Spec.Schedule,
		})
	}
	return cronjobConfigList, nil
}
