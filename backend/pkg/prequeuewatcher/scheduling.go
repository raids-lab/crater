package prequeuewatcher

import (
	"context"
	"fmt"
	"sort"

	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/indexer"
	"github.com/raids-lab/crater/pkg/utils"
)

const volcanoJobNameLabelKey = "volcano.sh/job-name"

func (w *PrequeueWatcher) getAssignedNodes(
	ctx context.Context,
	record *model.Job,
) ([]string, error) {
	if record == nil {
		return nil, nil
	}
	if nodes := record.Nodes.Data(); len(nodes) > 0 {
		filtered := lo.Filter(nodes, func(node string, _ int) bool {
			return node != ""
		})
		nodes = lo.Uniq(filtered)
		sort.Strings(nodes)
		return nodes, nil
	}

	podList := &v1.PodList{}
	if err := w.k8sClient.List(
		ctx,
		podList,
		client.InNamespace(config.GetConfig().Namespaces.Job),
		client.MatchingLabels{volcanoJobNameLabelKey: record.JobName},
	); err != nil {
		return nil, err
	}

	nodes := make([]string, 0, len(podList.Items))
	for i := range podList.Items {
		nodeName := podList.Items[i].Spec.NodeName
		if nodeName == "" {
			continue
		}
		nodes = append(nodes, nodeName)
	}
	nodes = lo.Uniq(nodes)
	sort.Strings(nodes)
	return nodes, nil
}

func (w *PrequeueWatcher) getNodeAvailableResources(ctx context.Context, node *v1.Node) (v1.ResourceList, error) {
	podList := &v1.PodList{}
	if err := w.k8sClient.List(ctx, podList, indexer.MatchingPodsByNodeName(node.Name)); err != nil {
		return nil, err
	}

	used := make(v1.ResourceList)
	for i := range podList.Items {
		used = utils.SumResources(used, utils.CalculateRequsetsByContainers(podList.Items[i].Spec.Containers))
	}

	return utils.SubtractResource(node.Status.Allocatable, used), nil
}

func getSingleNodeJobRequirements(record *model.Job) (*singleNodeJobRequirements, error) {
	if record == nil || record.Attributes.Data() == nil || len(record.Attributes.Data().Spec.Tasks) != 1 {
		return nil, fmt.Errorf("job does not have a single task template")
	}

	task := record.Attributes.Data().Spec.Tasks[0]
	if task.Replicas != 1 {
		return nil, fmt.Errorf("job is not single node")
	}

	spec := task.Template.Spec.DeepCopy()
	return &singleNodeJobRequirements{
		podSpec:  spec,
		requests: utils.CalculateRequsetsByContainers(spec.Containers),
	}, nil
}
