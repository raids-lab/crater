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

export const runtimeImages = {
  vLLM: 'ghcr.io/volcano-sh/vllm-openai:v0.10.0-cu128-nixl-v0.4.1-lmcache-0.3.2',
  SGLang: 'lmsysorg/sglang:v0.4.10.post2',
}
const uniqueItems = (items: Array<{ name: string }>) =>
  new Set(items.map((i) => i.name)).size === items.length
const profileSchema = z
  .object({
    secretEnvs: z
      .array(
        z.object({ name: z.string().min(1), secret: z.string().min(1), key: z.string().min(1) })
      )
      .refine(uniqueItems, () => ({ message: t('kthena.topology.duplicate') })),
    imageSource: z.enum(['platform', 'manual']),
    image: z.string(),
    platformImage: z.object({ imageLink: z.string().optional(), archs: z.array(z.string()) }),
    imageArchs: z.array(z.string()),
    resource: resourceSchema.extend({ cpu: z.number().positive(), memory: z.number().positive() }),
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
    if (!(p.imageSource === 'platform' ? p.platformImage.imageLink : p.image))
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        path: [p.imageSource === 'platform' ? 'platformImage' : 'image'],
        message: t('kthena.form.validation.imageRequired'),
      })
  })
const roleSchema = z.object({
  name: z.enum(['server', 'prefill', 'decode']),
  instances: z.number().int().min(1).max(10000),
  execution: z.enum(['single', 'ray']),
  nodesPerInstance: z.number().int().min(1).max(128),
  tensorParallel: z.number().int().min(1),
  pipelineParallel: z.number().int().min(1),
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
    name: z
      .string()
      .min(1)
      .max(63)
      .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/),
    modelSource: z.enum(['platform', 'external']),
    platformModelId: z.number().optional(),
    modelRevision: z.string(),
    modelURI: z.string(),
    servedModel: z.string(),
    backendType: z.enum(['vLLM', 'SGLang']),
    layout: z.enum(['combined', 'disaggregated']),
    replicas: z.number().int().min(1).max(10000),
    port: z.number().int().min(1).max(65535),
    roles: z.array(roleSchema).min(1).max(2),
  })
  .superRefine((v, ctx) => {
    const error = (path: (string | number)[], message: string) =>
      ctx.addIssue({ code: z.ZodIssueCode.custom, path, message })
    if (v.modelSource === 'platform' && !v.platformModelId)
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
      if (
        r.execution === 'ray' &&
        (v.backendType !== 'vLLM' ||
          r.nodesPerInstance < 2 ||
          r.tensorParallel !== r.entry.resource.gpu.count ||
          r.pipelineParallel !== r.nodesPerInstance)
      )
        error(['roles', i, 'nodesPerInstance'], t('kthena.topology.rayConstraint'))
      if (
        r.execution === 'single' &&
        (r.nodesPerInstance !== 1 ||
          r.pipelineParallel !== 1 ||
          r.tensorParallel !== Math.max(1, r.entry.resource.gpu.count))
      )
        error(['roles', i, 'tensorParallel'], t('kthena.topology.singleConstraint'))
    })
  })
export type FormSchema = z.infer<typeof formSchema>
export type ProfileForm = FormSchema['roles'][number]['entry']
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
