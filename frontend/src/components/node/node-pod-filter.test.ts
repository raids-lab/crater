// Copyright 2026 The Crater Project Team, RAIDS-Lab
import assert from 'node:assert/strict'
import test from 'node:test'

import { getNodePodTypeFilter } from './node-pod-filter.ts'

const owners = {
  volcano: { apiVersion: 'batch.volcano.sh/v1alpha1', kind: 'Job' },
  colocate: { apiVersion: 'aisystem.github.com/v1alpha1', kind: 'AIJob' },
  sparse: { apiVersion: 'kube-gpu-sparse.kube-gpu-sparse.io/v1', kind: 'RecommendDLJob' },
}

for (const scheduler of ['volcano', 'colocate', 'sparse'] as const) {
  test(`${scheduler} defaults to its backend owner type and keeps all scheduler filters available`, () => {
    const filter = getNodePodTypeFilter(scheduler)
    const owner = owners[scheduler]
    assert.deepEqual(filter.defaultValues, [`${owner.apiVersion}/${owner.kind}`])
    assert.deepEqual(
      filter.option.map((option) => option.value).sort(),
      Object.values(owners)
        .map((owner) => `${owner.apiVersion}/${owner.kind}`)
        .sort()
    )
    assert.ok(
      filter.option.some((option) => option.label === 'EMIAS' && option.value.endsWith('/AIJob'))
    )
  })
}
