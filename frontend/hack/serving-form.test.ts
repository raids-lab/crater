import assert from 'node:assert/strict'
import { test } from 'node:test'

import {
  defaultValues,
  formSchema,
} from '../src/routes/portal/inference-services/-components/serving-form-schema'
import {
  fromRequest,
  restoreNetworkProfile,
  toRequest,
} from '../src/routes/portal/inference-services/-components/serving-serialization'

test('clone restores node controls and preserves independent advanced selectors', () => {
  const req = toRequest(defaultValues())
  req.roles[0].entry.selectors = [
    { key: 'kubernetes.io/hostname', operator: 'NotIn', values: ['gpu-a'] },
    { key: 'rack', operator: 'In', values: ['r1'] },
  ]
  const form = fromRequest(req)
  const profile = form.roles[0].entry
  assert.equal(profile.nodeSelector.enable, true)
  assert.equal(profile.nodeSelector.mode, 'exclude')
  assert.deepEqual(toRequest(form).roles[0].entry.selectors, req.roles[0].entry.selectors)
  profile.nodeSelector.enable = false
  assert.deepEqual(toRequest(form).roles[0].entry.selectors, [req.roles[0].entry.selectors[1]])
  profile.selectors = []
  assert.deepEqual(toRequest(form).roles[0].entry.selectors, [])
})

test('registered RDMA quantities round trip and disabling or replacing removes old request', () => {
  const req = toRequest(defaultValues())
  req.roles[0].entry.extendedResources = { 'example.net/fabric': '2', 'example.net/other': '3' }
  const form = fromRequest(req)
  form.roles[0].entry = restoreNetworkProfile(form.roles[0].entry, ['example.net/fabric'])
  const profile = form.roles[0].entry
  assert.equal(profile.resource.network.enabled, true)
  assert.equal(profile.networkCount, 2)
  assert.deepEqual(
    toRequest(form).roles[0].entry.extendedResources,
    req.roles[0].entry.extendedResources
  )
  profile.resource.network.enabled = false
  assert.deepEqual(toRequest(form).roles[0].entry.extendedResources, { 'example.net/other': '3' })
  assert.equal(restoreNetworkProfile(profile, ['example.net/fabric']), profile)
  profile.resource.network = { enabled: true, model: 'example.net/replacement' }
  assert.deepEqual(toRequest(form).roles[0].entry.extendedResources, {
    'example.net/other': '3',
    'example.net/replacement': '2',
  })
})

test('all profile fields survive template serialization', () => {
  const form = defaultValues()
  form.name = 'regression'
  form.platformModelId = 1
  form.roles[0].entry.resource.cpu = 0.5
  form.roles[0].entry.secretEnvs = [{ name: 'HF_AUTH_TOKEN', secret: 'hf', key: 'token' }]
  const request = toRequest(formSchema.parse(form))
  assert.deepEqual(toRequest(formSchema.parse(fromRequest(request))), request)
})
