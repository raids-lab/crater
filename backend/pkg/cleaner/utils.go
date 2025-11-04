package cleaner

import (
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

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
