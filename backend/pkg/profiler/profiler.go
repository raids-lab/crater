package profiler

import (
	"context"
	"sync"
	"time"

	"github.com/aisystem/ai-protal/pkg/crclient"
	tasksvc "github.com/aisystem/ai-protal/pkg/db/task"
	"github.com/aisystem/ai-protal/pkg/models"
	"github.com/aisystem/ai-protal/pkg/monitor"
	"github.com/aisystem/ai-protal/pkg/util/queue"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Profiler struct {
	mutex            sync.Mutex
	taskQueue        queue.Queue                   //
	taskDB           tasksvc.DBService             // update profiling status
	prometheusClient *monitor.PrometheusClient     // get monitor data
	podControl       *crclient.ProfilingPodControl // get pod status
	profilingTimeout time.Duration                 // profiling timeout
}

func NewProfiler(mgr manager.Manager, prometheusClient *monitor.PrometheusClient, profileTimeout int) *Profiler {
	return &Profiler{
		mutex:            sync.Mutex{}, // todo: add lock to taskQueue
		taskQueue:        queue.New(keyFunc, fifoOrdering),
		taskDB:           tasksvc.NewDBService(),
		profilingTimeout: time.Duration(profileTimeout) * time.Second, //todo: configuraion
		podControl:       &crclient.ProfilingPodControl{mgr.GetClient()},
		prometheusClient: prometheusClient,
	}
}

func (p *Profiler) SubmitProfileTask(taskID uint) {
	task, err := p.taskDB.GetByID(taskID)
	if err != nil {
		logrus.Errorf("profiling task not found, taskID: %v", taskID)
		return
	}
	if task.ProfileStatus == models.UnProfiled {
		logrus.Infof("submit profiling task, user:%v, taskName:%v, taskID: %v", task.UserName, task.TaskName, taskID)
		p.taskDB.UpdateProfilingStat(task.ID, models.ProfileQueued, "", "")
		p.taskQueue.PushIfNotPresent(task)
	}
}
func (p *Profiler) DeleteProfilePodFromTask(taskID uint) {
	task, err := p.taskDB.GetByID(taskID)
	if err != nil {
		logrus.Errorf("profiling task not found, taskID: %v", taskID)
		return
	}
	p.taskQueue.Delete(task)
	p.podControl.DeleteProfilePodFromTask(task)
}

func (p *Profiler) Start(ctx context.Context) {
	go p.run(ctx)
}

func (p *Profiler) run(ctx context.Context) {
	ticker := time.Tick(5 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker:

			// create profiling pod
			// todo: check resource free
			// todo: check task status
			if p.taskQueue.Len() > 0 {
				t := p.taskQueue.Top()
				if t == nil {
					continue
				}
				task := t.(*models.AITask)
				// 1. create pod
				// 2. update task status

				err := p.podControl.CreateProfilePodFromTask(task)
				if err != nil {
					logrus.Errorf("create profiling pod failed, taskID:%v, taskName:%v, err:%v", task.ID, task.TaskName, err)
					p.taskDB.UpdateProfilingStat(task.ID, models.ProfileFailed, "", "")
				} else {
					logrus.Infof("create profiling pod success, taskID:%v, taskName:%v", task.ID, task.TaskName)
				}
				p.taskQueue.Delete(task)
			}

			// check profiling pod status
			podList, err := p.podControl.ListProflingPods()
			if err != nil {
				logrus.Errorf("list profiling pods failed: %v", err)
			}
			for _, pod := range podList {

				if pod.Status.Phase == corev1.PodPending {
					continue
				}
				// todo:
				// pod.Status.ContainerStatuses[0].State.Running.StartedAt?
				// pod.Status.StartTime
				if pod.Status.Phase == corev1.PodRunning && time.Now().Sub(pod.Status.StartTime.Time) < p.profilingTimeout {
					// p.taskDB.UpdateProfilingStat(task.ID, models.ProfileFailed, "", "")
					// todo: pod running-> update profiling stat
					continue
				}
				if pod.Status.Phase == corev1.PodUnknown {
					logrus.Errorf("profiling pod status unknow, pod: %v/%v", pod.Namespace, pod.Name)
					p.podControl.Delete(context.Background(), &pod)
					continue
				}

				taskID, err := p.podControl.GetTaskIDFromPod(&pod)
				if err != nil {
					logrus.Error(err)
					continue
				}

				jobStatus := ""
				if pod.Status.Phase == corev1.PodFailed {
					jobStatus = models.TaskFailedStatus
				} else if pod.Status.Phase == corev1.PodSucceeded {
					jobStatus = models.TaskSucceededStatus
				}
				podUtil, err := p.prometheusClient.QueryPodUtilMetric(pod.Namespace, pod.Name)
				if err != nil {
					logrus.Errorf("profile query pod util failed, taskID:%v, pod:%v/%v, err:%v", taskID, pod.Namespace, pod.Name, err)
					p.taskDB.UpdateProfilingStat(taskID, models.ProfileFailed, "", jobStatus)
				} else {
					p.taskDB.UpdateProfilingStat(taskID, models.ProfileFinish, monitor.PodUtilToJSON(podUtil), jobStatus)
					// todo: error handle
					logrus.Infof("profile query pod util success, taskID:%v, pod:%v/%v, status:%v", taskID, pod.Namespace, pod.Name, jobStatus)
				}
				p.podControl.Delete(context.Background(), &pod)
			}
		}
	}
}
