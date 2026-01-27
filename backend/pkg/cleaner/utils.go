package cleaner

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/monitor"
)

func NewCleanerClients(
	cli client.Client,
	kubeClient kubernetes.Interface,
	promClient monitor.PrometheusInterface,
) *Clients {
	return &Clients{
		Client:     cli,
		KubeClient: kubeClient,
		PromClient: promClient,
	}
}

func isJobscheduled(c context.Context, clients *Clients, jobName string) bool {
	namespace := config.GetConfig().Namespaces.Job
	pods, err := clients.KubeClient.CoreV1().Pods(namespace).List(c, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("volcano.sh/job-name=%s", jobName),
	})
	if err != nil {
		klog.Errorf("Failed to get pods: %v", err)
		return false
	}

	if len(pods.Items) == 0 {
		return false
	}

	result := true
	for i := range pods.Items {
		pod := &pods.Items[i]
		thisPodScheduled := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type != "PodScheduled" {
				continue
			}
			if condition.Status == "True" {
				thisPodScheduled = true
				break
			}
			if condition.Status == "False" {
				return false
			}
		}
		result = result && thisPodScheduled
	}
	return result
}
