package handler

import (
	"sort"
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"

	"github.com/raids-lab/crater/internal/payload"
)

func isQuotaResourceName(name v1.ResourceName) bool {
	return name == v1.ResourceCPU || name == v1.ResourceMemory || strings.Contains(string(name), "/")
}

func newQuotaResourceBase(quantity resource.Quantity) *payload.ResourceBase {
	return ptr.To(payload.ResourceBase{
		Amount: quantity.Value(),
		Format: string(quantity.Format),
	})
}

func newAllocatedQuotaResources(allocated v1.ResourceList) map[v1.ResourceName]payload.ResourceResp {
	resources := make(map[v1.ResourceName]payload.ResourceResp)
	for name, quantity := range allocated {
		if !isQuotaResourceName(name) {
			continue
		}
		resources[name] = payload.ResourceResp{
			Label:     string(name),
			Allocated: newQuotaResourceBase(quantity),
		}
	}
	return resources
}

func applyQuotaResourceList(
	resources map[v1.ResourceName]payload.ResourceResp,
	values v1.ResourceList,
	apply func(*payload.ResourceResp, *payload.ResourceBase),
) {
	for name, quantity := range values {
		resourceResp, ok := resources[name]
		if !ok {
			continue
		}
		apply(&resourceResp, newQuotaResourceBase(quantity))
		resources[name] = resourceResp
	}
}

func setQuotaGuarantee(resp *payload.ResourceResp, base *payload.ResourceBase) {
	resp.Guarantee = base
}

func setQuotaDeserved(resp *payload.ResourceResp, base *payload.ResourceBase) {
	resp.Deserved = base
}

func setQuotaCapability(resp *payload.ResourceResp, base *payload.ResourceBase) {
	resp.Capability = base
}

func buildQuotaResp(resources map[v1.ResourceName]payload.ResourceResp) payload.QuotaResp {
	cpu := resources[v1.ResourceCPU]
	cpu.Label = "cpu"
	memory := resources[v1.ResourceMemory]
	memory.Label = "mem"

	var gpus []payload.ResourceResp
	for name, resourceResp := range resources {
		if !strings.Contains(string(name), "/") {
			continue
		}
		if split := strings.Split(string(name), "/"); len(split) == 2 {
			resourceResp.Label = split[1]
		}
		gpus = append(gpus, resourceResp)
	}
	sort.Slice(gpus, func(i, j int) bool {
		return gpus[i].Label < gpus[j].Label
	})

	return payload.QuotaResp{
		CPU:    cpu,
		Memory: memory,
		GPUs:   gpus,
	}
}
