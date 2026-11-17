import { t } from 'i18next'
import { GaugeIcon } from 'lucide-react'
import { z } from 'zod'

import {
  NodeSelectorMode,
  defaultResource,
  envsSchema,
  nodeSelectorSchema,
  resourceSchema,
} from '@/utils/form'

export const servingFormErrorMap: z.ZodErrorMap = (issue, { defaultError }) => ({
  message:
    issue.code === z.ZodIssueCode.invalid_type && issue.path.slice(-2).join('.') === 'gpu.count'
      ? t('kthena.form.validation.gpuCount')
      : defaultError,
})

export const runtimeImages = {
  vLLM: 'ghcr.io/volcano-sh/vllm-openai:v0.10.0-cu128-nixl-v0.4.1-lmcache-0.3.2',
  SGLang: 'lmsysorg/sglang:v0.4.10.post2',
}
const uniqueItems = (items: Array<{ name: string }>) =>
  new Set(items.map((i) => i.name)).size === items.length
const requiredText = (label: string) =>
  z
    .string({
      errorMap: () => ({
        message: t('kthena.form.validation.requiredField', { field: t(label) }),
      }),
    })
    .min(1)
const boundedInteger = (min: number, max: number) =>
  z
    .number({
      errorMap: () => ({ message: t('kthena.form.validation.integerRange', { min, max }) }),
    })
    .int()
    .min(min)
    .max(max)
const positiveNumber = () =>
  z
    .number({
      errorMap: () => ({ message: t('kthena.form.validation.positiveNumber') }),
    })
    .positive()
const serviceName = z
  .string()
  .trim()
  .superRefine((name, ctx) => {
    if (!name || name.length > 63 || !/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(name))
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: t(
          name ? 'kthena.form.validation.namePattern' : 'kthena.form.validation.nameRequired'
        ),
      })
  })
export const selectedServingImage = (profile: ProfileForm) =>
  (profile.imageSource === 'platform' ? profile.platformImage.imageLink : profile.image)?.trim()

const profileSchema = z
  .object({
    secretEnvs: z
      .array(
        z.object({
          name: requiredText('kthena.topology.envName'),
          secret: requiredText('kthena.topology.secretName'),
          key: requiredText('kthena.topology.secretKey'),
        })
      )
      .refine(uniqueItems, () => ({ message: t('kthena.topology.duplicate') })),
    imageSource: z.enum(['platform', 'manual']),
    image: z.string(),
    platformImage: z.object({ imageLink: z.string().optional(), archs: z.array(z.string()) }),
    imageArchs: z.array(z.string()),
    resource: resourceSchema.extend({ cpu: positiveNumber(), memory: positiveNumber() }),
    envs: envsSchema.refine(uniqueItems, () => ({ message: t('kthena.topology.duplicate') })),
    selectors: z.array(
      z.object({ key: z.string(), operator: z.string(), values: z.array(z.string()).optional() })
    ),
    tolerations: z.array(
      z.object({
        key: z.string().optional(),
        operator: z.string().optional(),
        value: z.string().optional(),
        effect: z.string().optional(),
        tolerationSeconds: z.number().optional(),
      })
    ),
    nodeSelector: nodeSelectorSchema,
    extendedResources: z.record(z.string().min(1)),
    networkCount: z.number().int().positive(),
  })
  .superRefine((p, ctx) => {
    if (!(p.imageSource === 'platform' ? p.platformImage.imageLink : p.image)?.trim())
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: [p.imageSource === 'platform' ? 'platformImage' : 'image'],
        message: t('kthena.form.validation.imageRequired'),
      })
  })
const roleSchema = z.object({
  name: z.enum(['server', 'prefill', 'decode']),
  instances: boundedInteger(1, 10000),
  execution: z.enum(['single', 'ray']),
  nodesPerInstance: boundedInteger(1, 128),
  // Existing configuration exports may contain these fields. The request derives
  // both values from the GPU and node counts.
  tensorParallel: z.number().nullish(),
  pipelineParallel: z.number().nullish(),
  entry: profileSchema,
  worker: profileSchema.optional(),
  configItems: z
    .array(z.object({ key: z.string().regex(/^[a-z][a-z0-9-]*$/), value: z.string() }))
    .refine(
      (items) => new Set(items.map((i) => i.key)).size === items.length,
      () => ({ message: t('kthena.topology.duplicate') })
    ),
})
export const formSchema = z
  .object({
    name: serviceName,
    modelSource: z.enum(['platform', 'external']),
    platformModelId: z.number().int().optional(),
    modelRevision: z.string().trim(),
    modelURI: z.string().trim(),
    servedModel: z.string().trim(),
    backendType: z.enum(['vLLM', 'SGLang']),
    layout: z.enum(['combined', 'disaggregated']),
    replicas: boundedInteger(1, 10000),
    port: boundedInteger(1, 65535),
    roles: z.array(roleSchema).min(1).max(2),
  })
  .superRefine((v, ctx) => {
    const error = (path: (string | number)[], message: string) =>
      ctx.addIssue({ code: z.ZodIssueCode.custom, path, message })
    if (v.modelSource === 'platform' && !(v.platformModelId && v.platformModelId > 0))
      error(['platformModelId'], t('kthena.form.validation.platformModelRequired'))
    if (v.modelSource === 'external' && !/^(hf|ms|s3):\/\/.+/.test(v.modelURI))
      error(['modelURI'], t('kthena.topology.uri'))
    const expected = v.layout === 'combined' ? ['server'] : ['prefill', 'decode']
    if (
      v.roles.length !== expected.length ||
      expected.some((n) => !v.roles.some((r) => r.name === n))
    )
      error(['roles'], t('kthena.topology.invalidRoles'))
    v.roles.forEach((r, i) => {
      if (r.execution === 'ray' && v.backendType !== 'vLLM')
        error(['roles', i, 'execution'], t('kthena.topology.rayConstraint'))
      if (r.execution === 'ray' && r.nodesPerInstance < 2)
        error(['roles', i, 'nodesPerInstance'], t('kthena.topology.rayConstraint'))
    })
  })
export type FormSchema = z.infer<typeof formSchema>
export type ProfileForm = FormSchema['roles'][number]['entry']
type RoleForm = FormSchema['roles'][number]

export function servingParallelism(role: RoleForm) {
  return {
    tensorParallel: Math.max(1, role.entry.resource.gpu.count),
    pipelineParallel: role.execution === 'ray' ? role.nodesPerInstance : 1,
  }
}

const pdSharedOptions = [
  'dtype',
  'kv-cache-dtype',
  'tokenizer',
  'revision',
  'tokenizer-revision',
  'quantization',
  'trust-remote-code',
]

// Keep existing imports editable even when new checks require changes before deployment.
export const formSubmitSchema = formSchema.superRefine((v, ctx) => {
  const error = (path: (string | number)[], message: string) =>
    ctx.addIssue({ code: z.ZodIssueCode.custom, path, message })
  const distributed = v.layout === 'disaggregated' || v.roles.some((r) => r.execution === 'ray')
  if (v.modelSource === 'external') {
    if (v.modelURI.startsWith('s3://') && (v.modelRevision || distributed)) {
      error(['modelURI'], t('kthena.form.validation.s3RequiresPlatform'))
    } else if (distributed && !/^[a-f0-9]{40}$/i.test(v.modelRevision)) {
      error(['modelRevision'], t('kthena.form.validation.distributedRevisionRequired'))
    }
  }
  v.roles.forEach((role, i) => {
    if (role.execution === 'ray') {
      const gpu = role.entry.resource.gpu
      if (gpu.count < 1)
        error(
          ['roles', i, 'entry', 'resource', 'gpu', 'count'],
          t('kthena.form.validation.rayGpuRequired')
        )
      const worker = role.worker
      if (
        worker &&
        (worker.resource.gpu.count !== gpu.count ||
          (worker.resource.gpu.model || 'nvidia.com/gpu') !== (gpu.model || 'nvidia.com/gpu') ||
          selectedServingImage(worker) !== selectedServingImage(role.entry))
      )
        error(['roles', i, 'worker', 'root'], t('kthena.form.validation.rayWorkerMismatch'))
    }
    if (v.layout === 'disaggregated' && role.entry.resource.gpu.count < 1)
      error(
        ['roles', i, 'entry', 'resource', 'gpu', 'count'],
        t('kthena.form.validation.pdGpuRequired')
      )
  })
  if (v.layout === 'disaggregated' && v.roles.length === 2) {
    const [first, second] = v.roles
    if (selectedServingImage(first.entry) !== selectedServingImage(second.entry)) {
      error(
        ['roles', 1, 'entry', second.entry.imageSource === 'platform' ? 'platformImage' : 'image'],
        t('kthena.form.validation.pdImageMismatch')
      )
    }
    for (const option of pdSharedOptions) {
      const getOption = (role: RoleForm) =>
        role.configItems.find((item) => item.key === option)?.value || ''
      if (getOption(first) !== getOption(second)) {
        error(
          ['roles', 1, 'configItems'],
          t('kthena.form.validation.pdOptionsMismatch', { option })
        )
        break
      }
    }
  }
})

export const defaultProfile = (engine: 'vLLM' | 'SGLang'): ProfileForm => ({
  secretEnvs: [],
  imageSource: 'manual',
  image: runtimeImages[engine],
  platformImage: { imageLink: '', archs: [] },
  imageArchs: [],
  resource: { ...defaultResource, cpu: 4, memory: 16, gpu: { count: 1, model: 'nvidia.com/gpu' } },
  envs: [],
  selectors: [],
  tolerations: [],
  nodeSelector: { enable: false, mode: NodeSelectorMode.Include, nodes: [] },
  extendedResources: {},
  networkCount: 1,
})
export const defaultRole = (
  name: 'server' | 'prefill' | 'decode',
  engine: 'vLLM' | 'SGLang'
): FormSchema['roles'][number] => ({
  name,
  instances: 1,
  execution: 'single',
  nodesPerInstance: 1,
  tensorParallel: 1,
  pipelineParallel: 1,
  entry: defaultProfile(engine),
  configItems: [],
})
export const defaultValues = (): FormSchema => ({
  name: '',
  modelSource: 'platform',
  modelRevision: '',
  modelURI: '',
  servedModel: '',
  backendType: 'vLLM',
  layout: 'combined',
  replicas: 1,
  port: 8000,
  roles: [defaultRole('server', 'vLLM')],
})
export type DeploymentPreset = {
  key: string
  title: string
  description: string
  icon: typeof GaugeIcon
  values: Partial<FormSchema>
}
export const deploymentPresets: DeploymentPreset[] = (['vLLM', 'SGLang'] as const).map(
  (engine) => ({
    key: engine,
    title: engine,
    description: t('kthena.topology.preset'),
    icon: GaugeIcon,
    values: { backendType: engine, layout: 'combined', roles: [defaultRole('server', engine)] },
  })
)
