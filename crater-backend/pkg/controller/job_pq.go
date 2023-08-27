package controller

import (
	"container/heap"

	aijobapi "github.com/aisys/ai-task-controller/pkg/apis/aijob/v1alpha1"
)

type JobPQ []*aijobapi.AIJob

func (pq JobPQ) Len() int {
	return len(pq)
}

// todo: modify this compare function
func (pq JobPQ) Less(i, j int) bool {
	return pq[i].CreationTimestamp.Before(&pq[j].CreationTimestamp)
}

func (pq JobPQ) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *JobPQ) Push(x interface{}) {
	*pq = append(*pq, x.(*aijobapi.AIJob))
}

func (pq *JobPQ) Pop() interface{} {
	old := *pq
	n := len(old)
	x := old[n-1]
	*pq = old[0 : n-1]
	return x
}

func (pq *JobPQ) PushJob(job *aijobapi.AIJob) {
	heap.Push(pq, job)
}

func (pq *JobPQ) PopJob() *aijobapi.AIJob {
	if len(*pq) == 0 {
		return nil
	}
	return heap.Pop(pq).(*aijobapi.AIJob)
}
