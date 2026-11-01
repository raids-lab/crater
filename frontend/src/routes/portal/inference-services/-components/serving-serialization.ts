import type { CreateKthenaReq, ServingPodProfile } from '@/services/api/inference'

import { quantityValue } from '@/hooks/inference/format'

import { NodeSelectorMode, buildNodeSelectors } from '@/utils/form'

import { type FormSchema, type ProfileForm, defaultProfile } from './serving-form-schema'

function toProfile(p: ProfileForm): ServingPodProfile {
  const extendedResources = { ...p.extendedResources }
  if (p.resource.network.enabled && p.resource.network.model)
    extendedResources[p.resource.network.model] = String(p.networkCount)
  return {
    secretEnv: Object.fromEntries(
      p.secretEnvs.map((e) => [e.name, { name: e.secret, key: e.key }])
    ),
    image: p.imageSource === 'platform' ? p.platformImage.imageLink! : p.image,
    cpu: String(p.resource.cpu),
    memory: `${p.resource.memory}Gi`,
    gpu: String(p.resource.gpu.count),
    gpuModel: p.resource.gpu.model || 'nvidia.com/gpu',
    extendedResources,
    env: Object.fromEntries(p.envs.map((e) => [e.name, e.value])),
    imageArchs: p.imageSource === 'platform' ? p.platformImage.archs : p.imageArchs,
    selectors: [...(buildNodeSelectors(p.nodeSelector) ?? []), ...p.selectors],
    tolerations: p.tolerations,
  }
}
export function toRequest(v: FormSchema): CreateKthenaReq {
  return {
    name: v.name,
    modelSource: v.modelSource,
    modelRevision: v.modelSource === 'external' ? v.modelRevision : '',
    platformModelId: v.modelSource === 'platform' ? v.platformModelId : undefined,
    modelURI: v.modelSource === 'external' ? v.modelURI : undefined,
    servedModel: v.servedModel,
    backendType: v.backendType,
    layout: v.layout,
    replicas: v.replicas,
    port: v.port,
    roles: v.roles.map((r) => ({
      name: r.name,
      instances: r.instances,
      execution: r.execution,
      nodesPerInstance: r.nodesPerInstance,
      tensorParallel: r.tensorParallel,
      pipelineParallel: r.pipelineParallel,
      entry: toProfile(r.entry),
      worker: r.execution === 'ray' && r.worker ? toProfile(r.worker) : undefined,
      engineOptions: Object.fromEntries(r.configItems.map((c) => [c.key, c.value])),
    })),
  }
}
function fromProfile(p: ServingPodProfile, engine: FormSchema['backendType']): ProfileForm {
  const selectors = [...(p.selectors ?? [])]
  const hostIndex = selectors.findIndex(
    (s) => s.key === 'kubernetes.io/hostname' && ['In', 'NotIn'].includes(s.operator)
  )
  const host = hostIndex >= 0 ? selectors.splice(hostIndex, 1)[0] : undefined
  return {
    ...defaultProfile(engine),
    nodeSelector: {
      enable: !!host,
      mode: host?.operator === 'NotIn' ? NodeSelectorMode.Exclude : NodeSelectorMode.Include,
      nodes: host?.values ?? [],
    },
    secretEnvs: Object.entries(p.secretEnv ?? {}).map(([name, ref]) => ({
      name,
      secret: ref.name,
      key: ref.key,
    })),
    image: p.image,
    imageArchs: p.imageArchs ?? [],
    selectors,
    tolerations: p.tolerations ?? [],
    extendedResources: p.extendedResources ?? {},
    envs: Object.entries(p.env ?? {}).map(([name, value]) => ({ name, value })),
    resource: {
      ...defaultProfile(engine).resource,
      cpu: quantityValue(p.cpu),
      memory: quantityValue(p.memory) / 1024 ** 3,
      gpu: { count: quantityValue(p.gpu), model: p.gpuModel },
    },
  }
}
export function fromRequest(req: CreateKthenaReq): FormSchema {
  return {
    ...req,
    modelRevision: req.modelRevision ?? '',
    modelURI: req.modelURI ?? '',
    servedModel: req.servedModel ?? '',
    roles: req.roles.map((r) => ({
      ...r,
      entry: fromProfile(r.entry, req.backendType),
      worker: r.worker ? fromProfile(r.worker, req.backendType) : undefined,
      configItems: Object.entries(r.engineOptions ?? {}).map(([key, value]) => ({ key, value })),
    })),
  }
}
export function resourceSummary(v: FormSchema) {
  const resources: Record<string, number> = {}
  let pods = 0
  for (const r of v.roles) {
    pods += v.replicas * r.instances * r.nodesPerInstance
    const entry = toProfile(r.entry),
      worker = toProfile(r.worker ?? r.entry)
    for (const [p, count] of [
      [entry, 1],
      [worker, r.nodesPerInstance - 1],
    ] as const) {
      const multiplier = v.replicas * r.instances * count
      const profile = { cpu: p.cpu, memory: p.memory, ...p.extendedResources, [p.gpuModel]: p.gpu }
      for (const [k, q] of Object.entries(profile)) {
        resources[k] = (resources[k] ?? 0) + quantityValue(q) * multiplier
      }
    }
  }
  return { pods, resources }
}

// Restore a registered network allocation into the shared RDMA control. Unknown
// resources remain in the visible extended-resource editor.
export function restoreNetworkProfile(p: ProfileForm, networkNames: string[]): ProfileForm {
  if (p.resource.network.model !== undefined) return p
  const name = networkNames.find((n) => n in p.extendedResources)
  if (!name) return p
  const extendedResources = { ...p.extendedResources }
  const networkCount = quantityValue(extendedResources[name])
  delete extendedResources[name]
  return {
    ...p,
    extendedResources,
    networkCount,
    resource: { ...p.resource, network: { enabled: true, model: name } },
  }
}
