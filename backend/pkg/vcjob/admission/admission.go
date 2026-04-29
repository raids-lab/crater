package admission

import (
	"context"
	"fmt"
	"sort"

	v1 "k8s.io/api/core/v1"
	nodeutil "k8s.io/component-helpers/node/util"
	corev1helpers "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/component-helpers/scheduling/corev1/nodeaffinity"
	"sigs.k8s.io/controller-runtime/pkg/client"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/pkg/utils"
)

const (
	FailureReason = "AdmissionFailed"
)

type Result struct {
	Accepted bool
	Reason   string
}

type placementTask struct {
	name       string
	replicas   int32
	requests   v1.ResourceList
	candidates []int
	score      float64
}

func CheckJobAdmission(ctx context.Context, k8sClient client.Client, job *batch.Job) (*Result, error) {
	if job == nil {
		return nil, fmt.Errorf("invalid job: nil")
	}
	if len(job.Spec.Tasks) == 0 {
		return nil, fmt.Errorf("invalid job spec: no tasks defined")
	}
	nodeList := &v1.NodeList{}
	if err := k8sClient.List(ctx, nodeList); err != nil {
		return nil, err
	}
	remaining := make([]v1.ResourceList, len(nodeList.Items))
	for i := range nodeList.Items {
		remaining[i] = utils.CopyResources(nodeList.Items[i].Status.Allocatable)
	}

	tasks := buildPlacementTasks(nodeList.Items, job)
	sort.Slice(tasks, func(i, j int) bool { return tasks[i].score > tasks[j].score })
	for _, task := range tasks {
		for replica := int32(0); replica < task.replicas; replica++ {
			nodeIndex := bestFitNode(nodeList.Items, remaining, task)
			if nodeIndex < 0 {
				return &Result{
					Accepted: false,
					Reason: fmt.Sprintf(
						"cannot place replica %d/%d of task %s with requests %s",
						replica+1,
						task.replicas,
						task.name,
						utils.ResourceListSummary(task.requests),
					),
				}, nil
			}
			remaining[nodeIndex] = utils.SubtractResource(remaining[nodeIndex], task.requests)
		}
	}

	return &Result{Accepted: true}, nil
}

func buildPlacementTasks(nodes []v1.Node, job *batch.Job) []*placementTask {
	maxResourceMap := make(v1.ResourceList, 0)
	tasks := make([]*placementTask, 0, len(job.Spec.Tasks))
	for i := range job.Spec.Tasks {
		task := &job.Spec.Tasks[i]
		if task.Replicas <= 0 {
			continue
		}
		filteredNodeIndices := make([]int, 0)
		podSpec := &task.Template.Spec
		clear(maxResourceMap)
		resource := utils.CalculateRequsetsByContainers(task.Template.Spec.Containers)
		for nodeIndex := range nodes {
			node := &nodes[nodeIndex]
			if NodeMatchesPodSchedulingConstraints(node, podSpec) &&
				utils.ResourceListCovers(node.Status.Allocatable, resource) {
				filteredNodeIndices = append(filteredNodeIndices, nodeIndex)
			}
		}
		for _, nodeIndex := range filteredNodeIndices {
			node := &nodes[nodeIndex]
			for name, quantity := range node.Status.Allocatable {
				if existing, ok := maxResourceMap[name]; !ok || quantity.Cmp(existing) > 0 {
					maxResourceMap[name] = quantity
				}
			}
		}
		score := 0.0
		for name, quantity := range resource {
			if maxQuantity, ok := maxResourceMap[name]; ok && maxQuantity.Value() > 0 {
				s := float64(quantity.Value()) / float64(maxQuantity.Value())
				score += s
			}
		}
		p := &placementTask{
			name:       task.Name,
			replicas:   task.Replicas,
			requests:   resource,
			candidates: filteredNodeIndices,
			score:      score,
		}
		tasks = append(tasks, p)
	}
	return tasks
}

func bestFitNode(nodes []v1.Node, remaining []v1.ResourceList, task *placementTask) int {
	bestNodeIndex := -1
	bestScore := 0.0
	for _, nodeIndex := range task.candidates {
		if !utils.ResourceListCovers(remaining[nodeIndex], task.requests) {
			continue
		}
		next := utils.SubtractResource(remaining[nodeIndex], task.requests)
		score := 0.0
		for name, quantity := range task.requests {
			if quantity.Sign() <= 0 {
				continue
			}
			capacity := nodes[nodeIndex].Status.Allocatable[name]
			if capacity.Value() <= 0 {
				continue
			}
			left := next[name]
			score += float64(capacity.Value()-left.Value()) / float64(capacity.Value())
		}
		if bestNodeIndex < 0 || score > bestScore {
			bestNodeIndex = nodeIndex
			bestScore = score
		}
	}
	return bestNodeIndex
}

func NodeMatchesPodSchedulingConstraints(node *v1.Node, podSpec *v1.PodSpec) bool {
	if node == nil || podSpec == nil || node.Spec.Unschedulable {
		return false
	}

	_, ready := nodeutil.GetNodeCondition(&node.Status, v1.NodeReady)
	if ready == nil || ready.Status != v1.ConditionTrue {
		return false
	}

	matchesAffinity, err := nodeaffinity.GetRequiredNodeAffinity(&v1.Pod{Spec: *podSpec}).Match(node)
	if err != nil || !matchesAffinity {
		return false
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
