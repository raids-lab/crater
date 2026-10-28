// Copyright 2026 The Crater Project Team, RAIDS-Lab

// These values match the owner API version/kind returned by node Pod snapshots.
const podTypes = {
  volcano: { value: 'batch.volcano.sh/v1alpha1/Job', label: 'BASE' },
  colocate: { value: 'aisystem.github.com/v1alpha1/AIJob', label: 'EMIAS' },
  sparse: { value: 'kube-gpu-sparse.kube-gpu-sparse.io/v1/RecommendDLJob', label: 'SPARSE' },
}

export function getNodePodTypeFilter(scheduler: keyof typeof podTypes) {
  return {
    option: Object.values(podTypes),
    defaultValues: [podTypes[scheduler].value],
  }
}
