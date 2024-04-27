package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/raids-lab/crater/pkg/config"
	resputil "github.com/raids-lab/crater/pkg/server/response"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	bus "volcano.sh/apis/pkg/apis/bus/v1alpha1"
)

type VolcanojobMgr struct {
	client.Client
}

func NewVolcanojobMgr(cl client.Client) Manager {
	return &VolcanojobMgr{
		Client: cl,
	}
}

func (mgr *VolcanojobMgr) RegisterPublic(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) RegisterProtected(g *gin.RouterGroup) {
	g.POST("", mgr.CreateVolcanoJob)
}

func (mgr *VolcanojobMgr) RegisterAdmin(_ *gin.RouterGroup) {}

func (mgr *VolcanojobMgr) CreateVolcanoJob(c *gin.Context) {
	job := batch.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "jupyter-lyl-test-1",
			Namespace: config.GetConfig().Workspace.Namespace,
		},
		Spec: batch.JobSpec{
			MinAvailable:  1,
			SchedulerName: "volcano",
			Policies: []batch.LifecyclePolicy{
				{
					Action: bus.RestartJobAction,
					Event:  bus.PodEvictedEvent,
				},
			},
			Tasks: []batch.TaskSpec{
				{
					Name:     "jupyter-lyl-test",
					Replicas: 1,
					Template: v1.PodTemplateSpec{
						Spec: v1.PodSpec{
							Containers: []v1.Container{
								{
									Name:  "jupyter-notebook",
									Image: "jupyter/base-notebook:ubuntu-22.04",
									Command: []string{
										"/bin/bash",
										"-c",
										"start.sh jupyter lab --allow-root --NotebookApp.base_url=/jupyter/21-23-72/",
									},
									WorkingDir: "/home/jovyan",
								},
							},
						},
					},
				},
			},
		},
	}
	err := mgr.Create(c, &job)
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	resputil.Success(c, job)
}
