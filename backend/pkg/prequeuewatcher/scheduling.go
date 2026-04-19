package prequeuewatcher

import (
	"context"
	"fmt"
	"sort"

	"github.com/samber/lo"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
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

func nodeMatchesPodSchedulingConstraints(node *v1.Node, podSpec *v1.PodSpec) bool {
	if node == nil || podSpec == nil || node.Spec.Unschedulable || !isNodeReady(node) {
		return false
	}

	for key, value := range podSpec.NodeSelector {
		if node.Labels[key] != value {
			return false
		}
	}

	if podSpec.Affinity != nil && podSpec.Affinity.NodeAffinity != nil {
		required := podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution
		if required != nil && !matchNodeSelector(node, required) {
			return false
		}
	}

	_, hasUntoleratedTaint := corev1helpers.FindMatchingUntoleratedTaint(
		node.Spec.Taints,
		podSpec.Tolerations,
		func(taint *v1.Taint) bool {
			return taint.Effect == v1.TaintEffectNoSchedule || taint.Effect == v1.TaintEffectNoExecute
		},
	)

	return !hasUntoleratedTaint
}

func isNodeReady(node *v1.Node) bool {
	for i := range node.Status.Conditions {
		condition := node.Status.Conditions[i]
		if condition.Type == v1.NodeReady {
			return condition.Status == v1.ConditionTrue
		}
	}
	return false
}

func matchNodeSelector(node *v1.Node, selector *v1.NodeSelector) bool {
	if selector == nil || len(selector.NodeSelectorTerms) == 0 {
		return false
	}

	for i := range selector.NodeSelectorTerms {
		if matchNodeSelectorTerm(node, &selector.NodeSelectorTerms[i]) {
			return true
		}
	}
	return false
}

func matchNodeSelectorTerm(node *v1.Node, term *v1.NodeSelectorTerm) bool {
	if node == nil || term == nil {
		return false
	}
	for i := range term.MatchExpressions {
		if !matchNodeSelectorRequirement(node.Labels, &term.MatchExpressions[i]) {
			return false
		}
	}
	return len(term.MatchFields) == 0
}

//nolint:gocyclo // Kubernetes selector operators are clearer as one switch.
func matchNodeSelectorRequirement(labels map[string]string, requirement *v1.NodeSelectorRequirement) bool {
	if requirement == nil {
		return false
	}

	value, exists := labels[requirement.Key]
	switch requirement.Operator {
	case v1.NodeSelectorOpIn:
		if !exists {
			return false
		}
		for _, expected := range requirement.Values {
			if value == expected {
				return true
			}
		}
		return false
	case v1.NodeSelectorOpNotIn:
		if !exists {
			return true
		}
		for _, unexpected := range requirement.Values {
			if value == unexpected {
				return false
			}
		}
		return true
	case v1.NodeSelectorOpExists:
		return exists
	case v1.NodeSelectorOpDoesNotExist:
		return !exists
	case v1.NodeSelectorOpGt, v1.NodeSelectorOpLt:
		if !exists || len(requirement.Values) != 1 {
			return false
		}
		current, err := resource.ParseQuantity(value)
		if err != nil {
			return false
		}
		target, err := resource.ParseQuantity(requirement.Values[0])
		if err != nil {
			return false
		}
		if requirement.Operator == v1.NodeSelectorOpGt {
			return current.Cmp(target) > 0
		}
		return current.Cmp(target) < 0
	default:
		return false
	}
}
