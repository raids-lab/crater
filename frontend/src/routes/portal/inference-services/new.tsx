import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtomValue } from 'jotai'
import {
  BookmarkPlusIcon,
  BracesIcon,
  CirclePlusIcon,
  CpuIcon,
  GaugeIcon,
  HardDriveIcon,
  Loader2Icon,
  PencilIcon,
  RocketIcon,
  Settings2Icon,
  Trash2Icon,
  VariableIcon,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

import AccordionCard from '@/components/form/accordion-card'
import Combobox, { ComboboxItem } from '@/components/form/combobox'
import { EnvFormCard } from '@/components/form/env-form-field'
import FormLabelMust from '@/components/form/form-label-must'
import { ImageFormField } from '@/components/form/image-form-field'
import { ResourceFormFields } from '@/components/form/resource-form-field'
import CardTitle from '@/components/label/card-title'
import PageTitle from '@/components/layout/page-title'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui-custom/alert-dialog'

import { apiGetNodes } from '@/services/api/cluster'
import { IDataset, apiGetDataset } from '@/services/api/dataset'
import {
  KthenaInferenceTemplate,
  KthenaInferenceTemplateConfig,
  apiCreateKthenaInferenceTemplate,
  apiCreateKthenaService,
  apiDeleteKthenaInferenceTemplate,
  apiGetKthenaService,
  apiListKthenaInferenceTemplates,
  apiUpdateKthenaInferenceTemplate,
} from '@/services/api/inference'
import { JobType } from '@/services/api/vcjob'

import {
  NodeSelectorMode,
  buildNodeSelectors,
  defaultResource,
  envsSchema,
  nodeSelectorSchema,
  resourceSchema,
} from '@/utils/form'
import { atomUserContext, atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

export const Route = createFileRoute('/portal/inference-services/new')({
  validateSearch: (search: Record<string, unknown>) => ({
    clone: typeof search.clone === 'string' ? search.clone : undefined,
  }),
  component: NewKthenaServicePage,
  loader: () => ({ crumb: t('kthena.form.title') }),
})

const configItemsSchema = z.array(
  z.object({
    key: z.string().min(1, t('kthena.form.validation.configKeyRequired')),
    value: z.string(),
  })
)

const formSchema = z.object({
  name: z
    .string()
    .min(1)
    .max(63)
    .regex(/^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, t('kthena.form.validation.namePattern')),
  modelSource: z.enum(['platform', 'external']).default('platform'),
  platformModelId: z.number().int().nonnegative().optional(),
  modelURI: z.string().optional(),
  servedModel: z.string().optional(),
  backendType: z.literal('vLLM').default('vLLM'),
  cacheURI: z.string().default('hostpath:///tmp/cache'),
  replicas: z.number().int().min(1).max(1000000).default(1),
  imageSource: z.enum(['platform', 'manual']).default('manual'),
  image: z.string().optional(),
  platformImage: z
    .object({
      imageLink: z.string().optional(),
      archs: z.array(z.string()).default([]),
    })
    .optional(),
  resource: resourceSchema,
  envs: envsSchema,
  configItems: configItemsSchema,
  nodeSelector: nodeSelectorSchema,
})

type FormSchema = z.infer<typeof formSchema>

type DeploymentPreset = {
  key: string
  title: string
  description: string
  icon: typeof CpuIcon
  values: Partial<FormSchema>
}

const deploymentPresets: DeploymentPreset[] = [
  {
    key: 'cpu-vllm',
    title: t('kthena.form.preset.cpu.title'),
    description: t('kthena.form.preset.cpu.description'),
    icon: CpuIcon,
    values: {
      modelSource: 'external',
      modelURI: 'hf://Qwen/Qwen2.5-0.5B-Instruct',
      servedModel: 'Qwen2.5-0.5B-Instruct',
      backendType: 'vLLM',
      imageSource: 'manual',
      image: 'public.ecr.aws/q9t5s3a7/vllm-cpu-release-repo:latest',
      resource: { ...defaultResource, cpu: 2, memory: 4 },
      replicas: 1,
      envs: [{ name: 'HF_ENDPOINT', value: 'https://hf-mirror.com' }],
      configItems: [
        { key: 'max-model-len', value: '32768' },
        { key: 'max-num-batched-tokens', value: '65536' },
        { key: 'block-size', value: '128' },
      ],
    },
  },
  {
    key: 'gpu-vllm',
    title: 'GPU vLLM',
    description: t('kthena.form.preset.gpu.description'),
    icon: GaugeIcon,
    values: {
      modelSource: 'external',
      modelURI: 'hf://Qwen/Qwen3-8B',
      servedModel: 'Qwen3-8B',
      backendType: 'vLLM',
      imageSource: 'manual',
      image: 'ghcr.io/volcano-sh/vllm-openai:v0.10.0-cu128-nixl-v0.4.1-lmcache-0.3.2',
      resource: {
        ...defaultResource,
        cpu: 4,
        memory: 16,
        gpu: { count: 1, model: 'nvidia.com/gpu' },
      },
      replicas: 1,
      envs: [
        { name: 'HF_ENDPOINT', value: 'https://hf-mirror.com' },
        { name: 'NCCL_IB_DISABLE', value: '0' },
      ],
      configItems: [
        { key: 'max-model-len', value: '32768' },
        { key: 'max-num-batched-tokens', value: '65536' },
        { key: 'tensor-parallel-size', value: '1' },
      ],
    },
  },
]

const runtimeImages: Record<string, string> = {
  vLLM: 'ghcr.io/volcano-sh/vllm-openai:v0.10.0-cu128-nixl-v0.4.1-lmcache-0.3.2',
}

const runtimeDescriptionKeys: Record<string, string> = {
  vLLM: 'kthena.form.runtime.vllm',
}

const parseMemoryGi = (value: string) => {
  const trimmed = value.trim()
  if (trimmed.endsWith('Gi')) {
    return parseInt(trimmed.slice(0, -2), 10) || 4
  }
  if (trimmed.endsWith('Mi')) {
    return Math.max(1, Math.ceil((parseInt(trimmed.slice(0, -2), 10) || 1024) / 1024))
  }
  return parseInt(trimmed, 10) || 4
}

const mapToItems = (value?: Record<string, string>) =>
  Object.entries(value ?? {}).map(([name, envValue]) => ({ name, value: envValue }))

const mapToConfigItems = (value?: Record<string, string>) =>
  Object.entries(value ?? {}).map(([key, configValue]) => ({ key, value: configValue }))

function NewKthenaServicePage() {
  const { t } = useTranslation()
  const navigate = Route.useNavigate()
  const { clone } = Route.useSearch()
  const queryClient = useQueryClient()
  const user = useAtomValue(atomUserInfo)
  const accountContext = useAtomValue(atomUserContext)
  const userTemplateQueryKey = [
    'kthena',
    'inference-templates',
    user?.id,
    accountContext?.space,
  ] as const
  const [advancedOpen, setAdvancedOpen] = useState(false)
  const [envOpen, setEnvOpen] = useState(false)
  const [guideOpen, setGuideOpen] = useState(false)
  const [schedulingOpen, setSchedulingOpen] = useState(false)
  const [activePresetKey, setActivePresetKey] = useState<string | null>(null)
  const [templateDialogOpen, setTemplateDialogOpen] = useState(false)
  const [editingTemplate, setEditingTemplate] = useState<KthenaInferenceTemplate | null>(null)
  const [templateToDelete, setTemplateToDelete] = useState<KthenaInferenceTemplate | null>(null)
  const [templateName, setTemplateName] = useState('')
  const [templateDescription, setTemplateDescription] = useState('')
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      modelSource: 'platform',
      platformModelId: undefined,
      modelURI: '',
      servedModel: '',
      backendType: 'vLLM',
      cacheURI: 'hostpath:///tmp/cache',
      replicas: 1,
      imageSource: 'manual',
      image: 'public.ecr.aws/q9t5s3a7/vllm-cpu-release-repo:latest',
      platformImage: {
        imageLink: '',
        archs: [],
      },
      resource: { ...defaultResource, cpu: 2, memory: 4 },
      envs: [{ name: 'HF_ENDPOINT', value: 'https://hf-mirror.com' }],
      configItems: [
        { key: 'max-model-len', value: '32768' },
        { key: 'max-num-batched-tokens', value: '65536' },
        { key: 'block-size', value: '128' },
      ],
      nodeSelector: {
        enable: false,
        mode: NodeSelectorMode.Include,
        nodes: [],
      },
    },
  })
  const modelSource = form.watch('modelSource')
  const platformModelId = form.watch('platformModelId')
  const imageSource = form.watch('imageSource')
  const serviceName = form.watch('name')
  const servedModel = form.watch('servedModel')
  const modelURI = form.watch('modelURI')
  const backendType = form.watch('backendType')
  const workerImage = form.watch('image')
  const platformImage = form.watch('platformImage')
  const resourceCPU = form.watch('resource.cpu')
  const resourceMemory = form.watch('resource.memory')
  const gpuCount = form.watch('resource.gpu.count')
  const gpuModel = form.watch('resource.gpu.model')
  const replicas = form.watch('replicas')
  const nodeSelectorEnabled = form.watch('nodeSelector.enable')
  const {
    fields: configFields,
    append: appendConfig,
    remove: removeConfig,
  } = useFieldArray({
    control: form.control,
    name: 'configItems',
  })

  const { data: modelItems = [] } = useQuery({
    queryKey: ['dataset', 'my-models'],
    queryFn: () => apiGetDataset(),
    select: (res) =>
      res.data
        .filter((item) => item.type === 'model')
        .sort((a, b) => a.name.localeCompare(b.name))
        .map(
          (item) =>
            ({
              value: String(item.id),
              label: item.name,
              selectedLabel: item.name,
              tags: item.extra?.tag ?? [],
              detail: item,
            }) as ComboboxItem<IDataset>
        ),
  })

  const selectedModel = useMemo(
    () => modelItems.find((item) => item.value === String(platformModelId))?.detail,
    [modelItems, platformModelId]
  )
  const { data: nodeItems = [] } = useQuery({
    queryKey: ['cluster', 'nodes', 'brief'],
    queryFn: () => apiGetNodes(),
    select: (res) =>
      res.data
        .map(
          (node) =>
            ({
              value: node.name,
              label: node.name,
            }) as ComboboxItem<{ name: string }>
        )
        .sort((a, b) => a.label.localeCompare(b.label)),
  })
  const { data: cloneSource } = useQuery({
    queryKey: ['kthena/inference-services', clone],
    queryFn: () => apiGetKthenaService(clone ?? '').then((res) => res.data),
    enabled: !!clone,
  })
  const { data: userTemplates = [], isLoading: isUserTemplatesLoading } = useQuery({
    queryKey: userTemplateQueryKey,
    queryFn: () => apiListKthenaInferenceTemplates().then((res) => res.data),
    enabled: Boolean(user?.id),
  })

  useEffect(() => {
    if (!cloneSource) return
    const resource = {
      ...defaultResource,
      cpu: parseInt(cloneSource.workerCPU || '2', 10) || 2,
      memory: parseMemoryGi(cloneSource.workerMemory || '4Gi'),
      gpu: {
        count: parseInt(cloneSource.workerGPU || '0', 10) || 0,
        model: cloneSource.workerGPUModel || undefined,
      },
    }
    form.reset({
      name: `${cloneSource.name}-copy`.slice(0, 63),
      modelSource: cloneSource.modelSource || 'external',
      platformModelId: cloneSource.platformModelId || undefined,
      modelURI: cloneSource.modelSource === 'external' ? cloneSource.modelURI : '',
      servedModel: cloneSource.servedModel,
      backendType: 'vLLM',
      cacheURI: cloneSource.cacheURI || 'hostpath:///tmp/cache',
      replicas: cloneSource.replicas || 1,
      imageSource: 'manual',
      image: cloneSource.workerImage,
      platformImage: {
        imageLink: '',
        archs: [],
      },
      resource,
      envs: mapToItems(cloneSource.env),
      configItems: mapToConfigItems(cloneSource.workerConfig),
      nodeSelector: {
        enable: false,
        mode: NodeSelectorMode.Include,
        nodes: [],
      },
    })
    setAdvancedOpen(true)
    setEnvOpen(Object.keys(cloneSource.env ?? {}).length > 0)
    setActivePresetKey(null)
  }, [cloneSource, form])

  const { mutate: createService, isPending } = useMutation({
    mutationFn: (values: FormSchema) =>
      apiCreateKthenaService({
        name: values.name,
        modelSource: values.modelSource,
        platformModelId: values.modelSource === 'platform' ? values.platformModelId : undefined,
        modelURI: values.modelSource === 'external' ? values.modelURI : undefined,
        servedModel: values.servedModel,
        backendType: values.backendType,
        cacheURI: values.cacheURI,
        replicas: values.replicas,
        env: Object.fromEntries(values.envs.map((item) => [item.name, item.value])),
        worker: {
          image:
            values.imageSource === 'platform'
              ? (values.platformImage?.imageLink ?? '')
              : (values.image ?? ''),
          replicas: 1,
          pods: 1,
          cpu: String(values.resource.cpu),
          memory: `${values.resource.memory}Gi`,
          gpu: String(values.resource.gpu.count),
          gpuModel: values.resource.gpu.model,
          config: Object.fromEntries(values.configItems.map((item) => [item.key, item.value])),
        },
        selectors: buildNodeSelectors(values.nodeSelector),
      }),
    onSuccess: async (_, values) => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      toast.success(t('kthena.form.createSuccess', { name: values.name }))
      navigate({ to: '/portal/inference-services' })
    },
    onError: showErrorToast,
  })

  const { mutate: saveUserTemplate, isPending: isSavingUserTemplate } = useMutation({
    mutationFn: ({
      templateID,
      data,
    }: {
      templateID?: number
      data: { name: string; description: string; config: KthenaInferenceTemplateConfig }
    }) => {
      const payload = data
      return templateID
        ? apiUpdateKthenaInferenceTemplate(templateID, payload).then((res) => res.data)
        : apiCreateKthenaInferenceTemplate(payload).then((res) => res.data)
    },
    onSuccess: async (template) => {
      await queryClient.invalidateQueries({ queryKey: userTemplateQueryKey })
      setActivePresetKey(`user:${template.id}`)
      setTemplateDialogOpen(false)
      setEditingTemplate(null)
      toast.success(t('kthena.form.template.saved'))
    },
    onError: showErrorToast,
  })

  const { mutate: deleteUserTemplate, isPending: isDeletingUserTemplate } = useMutation({
    mutationFn: (templateID: number) => apiDeleteKthenaInferenceTemplate(templateID),
    onSuccess: async (_, templateID) => {
      await queryClient.invalidateQueries({ queryKey: userTemplateQueryKey })
      if (activePresetKey === `user:${templateID}`) {
        setActivePresetKey(null)
      }
      setTemplateToDelete(null)
      toast.success(t('kthena.form.template.deleted'))
    },
    onError: showErrorToast,
  })

  const onSubmit = (values: FormSchema) => {
    const parsed = formSchema.parse(values)
    if (parsed.modelSource === 'platform' && !parsed.platformModelId) {
      form.setError('platformModelId', {
        message: t('kthena.form.validation.platformModelRequired'),
      })
      return
    }
    if (parsed.modelSource === 'external' && !parsed.modelURI) {
      form.setError('modelURI', { message: t('kthena.form.validation.modelURIRequired') })
      return
    }
    if (parsed.imageSource === 'platform' && !parsed.platformImage?.imageLink) {
      form.setError('platformImage', { message: t('kthena.form.validation.platformImageRequired') })
      return
    }
    if (parsed.imageSource === 'manual' && !parsed.image) {
      form.setError('image', { message: t('kthena.form.validation.imageRequired') })
      return
    }
    createService(parsed)
  }

  const applyPreset = (preset: DeploymentPreset) => {
    if (preset.values.modelSource === 'external') {
      form.setValue('platformModelId', undefined, { shouldDirty: true })
    }
    if (preset.values.imageSource === 'manual') {
      form.setValue('platformImage', { imageLink: '', archs: [] }, { shouldDirty: true })
    }
    Object.entries(preset.values).forEach(([key, value]) => {
      form.setValue(key as keyof FormSchema, value as never, {
        shouldDirty: true,
        shouldValidate: true,
      })
    })
    setActivePresetKey(preset.key)
  }

  const buildUserTemplateConfig = (): KthenaInferenceTemplateConfig => {
    const values = form.getValues()
    return {
      modelSource: values.modelSource,
      platformModelId: values.platformModelId,
      modelURI: values.modelURI,
      servedModel: values.servedModel,
      backendType: values.backendType,
      cacheURI: values.cacheURI,
      imageSource: values.imageSource,
      image: values.image,
      platformImage: values.platformImage
        ? {
            imageLink: values.platformImage.imageLink,
            archs: [...(values.platformImage.archs ?? [])],
          }
        : undefined,
      resource: {
        cpu: values.resource.cpu,
        memory: values.resource.memory,
        gpu: {
          count: values.resource.gpu.count,
          model: values.resource.gpu.model,
        },
      },
      replicas: values.replicas,
      envs: values.envs.map((item) => ({ ...item })),
      configItems: values.configItems.map((item) => ({ ...item })),
      nodeSelector: {
        enable: values.nodeSelector.enable,
        mode: values.nodeSelector.mode,
        nodes: [...(values.nodeSelector.nodes ?? [])],
      },
    }
  }

  const applyUserTemplate = (template: KthenaInferenceTemplate) => {
    const config = template.config
    const current = form.getValues()
    const templateMode =
      config.nodeSelector?.mode === NodeSelectorMode.Exclude
        ? NodeSelectorMode.Exclude
        : NodeSelectorMode.Include
    form.reset({
      ...current,
      modelSource: config.modelSource ?? current.modelSource,
      platformModelId:
        config.modelSource === 'external'
          ? undefined
          : (config.platformModelId ?? current.platformModelId),
      modelURI: config.modelURI ?? current.modelURI,
      servedModel: config.servedModel ?? current.servedModel,
      backendType: config.backendType ?? current.backendType,
      cacheURI: config.cacheURI ?? current.cacheURI,
      imageSource: config.imageSource ?? current.imageSource,
      image: config.image ?? current.image,
      platformImage: config.platformImage
        ? {
            imageLink: config.platformImage.imageLink ?? '',
            archs: [...(config.platformImage.archs ?? [])],
          }
        : current.platformImage,
      resource: {
        ...current.resource,
        ...config.resource,
        gpu: {
          ...current.resource.gpu,
          ...config.resource?.gpu,
        },
      },
      replicas: config.replicas ?? current.replicas,
      envs: config.envs?.map((item) => ({ ...item })) ?? current.envs,
      configItems: config.configItems?.map((item) => ({ ...item })) ?? current.configItems,
      nodeSelector: config.nodeSelector
        ? {
            enable: config.nodeSelector.enable ?? false,
            mode: templateMode,
            nodes: [...(config.nodeSelector.nodes ?? [])],
          }
        : current.nodeSelector,
    })
    setActivePresetKey(`user:${template.id}`)
  }

  const openTemplateDialog = (template?: KthenaInferenceTemplate) => {
    setEditingTemplate(template ?? null)
    setTemplateName(template?.name ?? '')
    setTemplateDescription(template?.description ?? '')
    setTemplateDialogOpen(true)
  }

  const submitUserTemplate = () => {
    const name = templateName.trim()
    if (!name) {
      toast.error(t('kthena.form.template.nameRequired'))
      return
    }
    saveUserTemplate({
      templateID: editingTemplate?.id,
      data: {
        name,
        description: templateDescription.trim(),
        config: buildUserTemplateConfig(),
      },
    })
  }

  const activeUserTemplate = userTemplates.find(
    (template) => activePresetKey === `user:${template.id}`
  )

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-1 flex-col gap-4">
        <PageTitle
          title={t('kthena.form.title')}
          description={
            clone
              ? t('kthena.form.cloneDescription', { name: clone })
              : t('kthena.form.description')
          }
        >
          <Button type="submit" disabled={isPending}>
            <RocketIcon className="size-4" />
            {isPending ? t('kthena.actions.requesting') : t('kthena.form.submit')}
          </Button>
        </PageTitle>

        <section className="space-y-2" aria-labelledby="runtime-template-heading">
          <div className="flex flex-wrap items-center justify-between gap-2 px-1">
            <div className="flex items-center gap-2">
              <span className="bg-primary/10 text-primary flex size-7 items-center justify-center rounded-md">
                <RocketIcon className="size-4" />
              </span>
              <h2 id="runtime-template-heading" className="text-sm font-semibold">
                {t('kthena.form.sections.presets')}
              </h2>
            </div>
            <div className="flex items-center gap-1.5">
              {activeUserTemplate && (
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  className="h-8"
                  onClick={() => openTemplateDialog(activeUserTemplate)}
                >
                  <PencilIcon className="size-3.5" />
                  {t('kthena.form.template.update')}
                </Button>
              )}
              <Button
                type="button"
                variant="outline"
                size="sm"
                className="h-8"
                onClick={() => openTemplateDialog()}
              >
                <BookmarkPlusIcon className="size-3.5" />
                {t('kthena.form.template.save')}
              </Button>
            </div>
          </div>

          <div
            role="radiogroup"
            aria-label={t('kthena.form.sections.presets')}
            className="bg-muted/25 grid gap-1.5 rounded-xl border p-1.5 sm:grid-cols-2"
          >
            {deploymentPresets.map((preset) => {
              const Icon = preset.icon
              const selected = activePresetKey === preset.key
              return (
                <button
                  key={preset.key}
                  type="button"
                  role="radio"
                  aria-checked={selected}
                  onClick={() => applyPreset(preset)}
                  className={[
                    'flex min-h-20 items-center gap-3 rounded-lg border px-3 py-2.5 text-left transition-all',
                    selected
                      ? 'border-primary/30 bg-background ring-primary/15 shadow-sm ring-1'
                      : 'hover:bg-background/80 border-transparent',
                  ].join(' ')}
                >
                  <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md">
                    <Icon className="size-4" />
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium">{preset.title}</span>
                      {selected && (
                        <Badge className="h-5 px-1.5 text-[10px]">
                          {t('kthena.form.summary.applied')}
                        </Badge>
                      )}
                    </div>
                    <p className="text-muted-foreground mt-0.5 text-xs leading-4">
                      {preset.description}
                    </p>
                  </div>
                  <div className="flex shrink-0 gap-1">
                    <Badge variant="secondary">vLLM</Badge>
                    <Badge variant="secondary">{preset.key === 'gpu-vllm' ? 'GPU' : 'CPU'}</Badge>
                  </div>
                </button>
              )
            })}
          </div>

          {(isUserTemplatesLoading || userTemplates.length > 0) && (
            <div className="space-y-1.5 pt-1">
              <div className="text-muted-foreground flex items-center gap-2 px-1 text-xs font-medium">
                {t('kthena.form.template.mine')}
                {isUserTemplatesLoading ? (
                  <Loader2Icon className="size-3 animate-spin" />
                ) : (
                  <Badge variant="secondary" className="h-5 px-1.5 text-[10px]">
                    {userTemplates.length}
                  </Badge>
                )}
              </div>
              {!isUserTemplatesLoading && (
                <div
                  role="radiogroup"
                  aria-label={t('kthena.form.template.mine')}
                  className="bg-muted/25 grid gap-1.5 rounded-xl border p-1.5 sm:grid-cols-2"
                >
                  {userTemplates.map((template) => {
                    const selected = activePresetKey === `user:${template.id}`
                    const isGPU = (template.config.resource?.gpu?.count ?? 0) > 0
                    return (
                      <div
                        key={template.id}
                        className={[
                          'flex min-w-0 items-stretch overflow-hidden rounded-lg border transition-all',
                          selected
                            ? 'border-primary/30 bg-background ring-primary/15 shadow-sm ring-1'
                            : 'hover:bg-background/80 border-transparent',
                        ].join(' ')}
                      >
                        <button
                          type="button"
                          role="radio"
                          aria-checked={selected}
                          className="flex min-w-0 flex-1 items-center gap-3 px-3 py-2.5 text-left"
                          onClick={() => applyUserTemplate(template)}
                        >
                          <div className="bg-primary/10 text-primary flex size-8 shrink-0 items-center justify-center rounded-md">
                            <BookmarkPlusIcon className="size-4" />
                          </div>
                          <div className="min-w-0 flex-1">
                            <div className="flex items-center gap-2">
                              <span className="truncate text-sm font-medium">{template.name}</span>
                              {selected && (
                                <Badge className="h-5 px-1.5 text-[10px]">
                                  {t('kthena.form.summary.applied')}
                                </Badge>
                              )}
                            </div>
                            <p className="text-muted-foreground mt-0.5 truncate text-xs leading-4">
                              {template.description || t('kthena.form.template.noDescription')}
                            </p>
                          </div>
                          <div className="flex shrink-0 gap-1">
                            <Badge variant="secondary">vLLM</Badge>
                            <Badge variant="secondary">{isGPU ? 'GPU' : 'CPU'}</Badge>
                          </div>
                        </button>
                        <div className="bg-muted/20 flex shrink-0 flex-col justify-center border-l p-1">
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="size-7"
                            aria-label={t('kthena.form.template.update')}
                            title={t('kthena.form.template.update')}
                            onClick={() => openTemplateDialog(template)}
                          >
                            <PencilIcon className="size-3.5" />
                          </Button>
                          <Button
                            type="button"
                            variant="ghost"
                            size="icon"
                            className="text-muted-foreground hover:text-destructive size-7"
                            aria-label={t('kthena.form.template.delete')}
                            title={t('kthena.form.template.delete')}
                            onClick={() => setTemplateToDelete(template)}
                          >
                            <Trash2Icon className="size-3.5" />
                          </Button>
                        </div>
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )}
        </section>

        <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_24rem]">
          <div className="flex min-w-0 flex-col gap-4">
            <div className="grid items-stretch gap-4 lg:grid-cols-2">
              <Card>
                <CardHeader>
                  <CardTitle icon={HardDriveIcon}>
                    {t('kthena.form.sections.modelSource')}
                  </CardTitle>
                </CardHeader>
                <CardContent className="grid gap-5">
                  <FormField
                    control={form.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>
                          {t('kthena.form.fields.serviceName')}
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Input {...field} placeholder="qwen-demo" />
                        </FormControl>
                        <FormDescription>
                          {t('kthena.form.descriptions.serviceName')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="modelSource"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('kthena.form.fields.source')}</FormLabel>
                        <Select value={field.value} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger className="w-full">
                              <SelectValue />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="platform">
                              {t('kthena.form.source.platform')}
                            </SelectItem>
                            <SelectItem value="external">
                              {t('kthena.form.source.external')}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                        <FormDescription>{t('kthena.form.descriptions.source')}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  {modelSource === 'platform' ? (
                    <FormField
                      control={form.control}
                      name="platformModelId"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('kthena.form.fields.platformModel')}
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Combobox
                              items={modelItems}
                              current={field.value ? String(field.value) : ''}
                              handleSelect={(value) => {
                                const id = Number(value)
                                field.onChange(id)
                                const model = modelItems.find(
                                  (item) => item.value === value
                                )?.detail
                                if (model) {
                                  form.setValue('servedModel', model.name, { shouldDirty: true })
                                }
                              }}
                              renderLabel={(item) => (
                                <div className="flex min-w-0 flex-col">
                                  <span className="truncate">{item.label}</span>
                                  <span className="text-muted-foreground truncate text-xs">
                                    {item.detail?.url}
                                  </span>
                                </div>
                              )}
                              formTitle={t('kthena.form.selectPlatformModel')}
                            />
                          </FormControl>
                          <FormDescription>
                            {selectedModel
                              ? t('kthena.form.descriptions.selectedModel', {
                                  url: selectedModel.url,
                                })
                              : t('kthena.form.descriptions.platformModel')}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  ) : (
                    <FormField
                      control={form.control}
                      name="modelURI"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('kthena.form.fields.modelURI')}
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input
                              {...field}
                              className="font-mono"
                              placeholder="hf://Qwen/Qwen2.5-0.5B-Instruct"
                            />
                          </FormControl>
                          <FormDescription>
                            {t('kthena.form.descriptions.modelURI')}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
                  <FormField
                    control={form.control}
                    name="servedModel"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('kthena.form.fields.servedModel')}</FormLabel>
                        <FormControl>
                          <Input {...field} className="font-mono" />
                        </FormControl>
                        <FormDescription>
                          {t('kthena.form.descriptions.servedModel')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                </CardContent>
              </Card>

              <Card>
                <CardHeader>
                  <CardTitle icon={RocketIcon}>{t('kthena.form.sections.runtime')}</CardTitle>
                </CardHeader>
                <CardContent className="grid gap-5">
                  <FormField
                    control={form.control}
                    name="backendType"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('kthena.form.fields.backend')}</FormLabel>
                        <Select
                          value={field.value}
                          onValueChange={(value) => {
                            field.onChange(value)
                            const image = runtimeImages[value]
                            if (image) {
                              form.setValue('imageSource', 'manual', {
                                shouldDirty: true,
                                shouldValidate: true,
                              })
                              form.setValue('image', image, {
                                shouldDirty: true,
                                shouldValidate: true,
                              })
                            }
                          }}
                        >
                          <FormControl>
                            <SelectTrigger className="w-full">
                              <SelectValue />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="vLLM">vLLM</SelectItem>
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          {t(
                            runtimeDescriptionKeys[field.value ?? ''] ??
                              'kthena.form.runtime.default'
                          )}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="imageSource"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('kthena.form.fields.imageSource')}</FormLabel>
                        <Select value={field.value} onValueChange={field.onChange}>
                          <FormControl>
                            <SelectTrigger className="w-full">
                              <SelectValue />
                            </SelectTrigger>
                          </FormControl>
                          <SelectContent>
                            <SelectItem value="platform">
                              {t('kthena.form.imageSource.platform')}
                            </SelectItem>
                            <SelectItem value="manual">
                              {t('kthena.form.imageSource.manual')}
                            </SelectItem>
                          </SelectContent>
                        </Select>
                        <FormDescription>
                          {t('kthena.form.descriptions.imageSource')}
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  {imageSource === 'platform' ? (
                    <ImageFormField
                      form={form}
                      name="platformImage"
                      jobType={JobType.Custom}
                      label={t('kthena.form.fields.workerImage')}
                    />
                  ) : (
                    <FormField
                      control={form.control}
                      name="image"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('kthena.form.fields.workerImage')}
                            <FormLabelMust />
                          </FormLabel>
                          <FormControl>
                            <Input {...field} className="font-mono" />
                          </FormControl>
                          <FormDescription>
                            {t('kthena.form.descriptions.workerImage')}
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  )}
                </CardContent>
              </Card>
            </div>

            <Card>
              <CardHeader>
                <CardTitle icon={Settings2Icon}>{t('kthena.form.sections.resources')}</CardTitle>
              </CardHeader>
              <CardContent className="grid gap-5">
                <ResourceFormFields
                  form={form}
                  cpuPath="resource.cpu"
                  memoryPath="resource.memory"
                  gpuCountPath="resource.gpu.count"
                  gpuModelPath="resource.gpu.model"
                />
                {gpuCount > 0 && (
                  <div className="text-muted-foreground rounded-md border p-3 text-sm">
                    {t('kthena.form.descriptions.gpuModel')}
                  </div>
                )}
                <FormField
                  control={form.control}
                  name="replicas"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>{t('kthena.form.fields.replicas')}</FormLabel>
                      <FormControl>
                        <Input
                          {...field}
                          type="number"
                          min={1}
                          max={1000000}
                          onChange={(event) => field.onChange(event.target.valueAsNumber)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>
          </div>

          <aside className="flex min-w-0 flex-col gap-4 xl:sticky xl:top-4">
            <Card className="border-primary/15 overflow-hidden shadow-sm">
              <CardHeader className="bg-muted/20 border-b py-4">
                <div className="flex items-center justify-between gap-3">
                  <CardTitle icon={RocketIcon}>{t('kthena.form.sections.summary')}</CardTitle>
                  <Badge variant={activePresetKey ? 'default' : 'secondary'}>
                    {activePresetKey
                      ? t('kthena.form.summary.applied')
                      : t('kthena.form.summary.custom')}
                  </Badge>
                </div>
              </CardHeader>
              <CardContent className="grid gap-4 p-4">
                <DeploymentSummaryRow
                  label={t('kthena.form.fields.serviceName')}
                  value={serviceName || '-'}
                />
                <DeploymentSummaryRow
                  label={t('kthena.form.fields.servedModel')}
                  value={
                    modelSource === 'platform'
                      ? selectedModel?.name || servedModel || '-'
                      : servedModel || modelURI || '-'
                  }
                  mono
                />
                <DeploymentSummaryRow label={t('kthena.form.fields.backend')} value={backendType} />
                <DeploymentSummaryRow
                  label={t('kthena.form.fields.workerImage')}
                  value={
                    imageSource === 'platform'
                      ? platformImage?.imageLink || '-'
                      : workerImage || '-'
                  }
                  mono
                />
                <div className="grid gap-2 border-t pt-4">
                  <span className="text-muted-foreground text-xs font-medium">
                    {t('kthena.form.sections.resources')}
                  </span>
                  <div className="flex flex-wrap gap-1.5">
                    <Badge variant="secondary">{resourceCPU || 0}c</Badge>
                    <Badge variant="secondary">{resourceMemory || 0}Gi</Badge>
                    {gpuCount > 0 && (
                      <Badge variant="secondary">
                        {gpuModel || 'GPU'}: {gpuCount}
                      </Badge>
                    )}
                    <Badge variant="secondary">
                      {t('kthena.form.fields.replicas')}: {replicas || 1}
                    </Badge>
                  </div>
                </div>
              </CardContent>
            </Card>

            <AccordionCard
              cardTitle={t('kthena.form.sections.scheduling')}
              icon={Settings2Icon}
              open={schedulingOpen}
              setOpen={setSchedulingOpen}
            >
              <div className="mt-3 grid gap-4">
                <FormField
                  control={form.control}
                  name="nodeSelector.enable"
                  render={({ field }) => (
                    <FormItem className="flex flex-row items-center justify-between space-y-0">
                      <div className="min-w-0 pr-3">
                        <FormLabel>{t('kthena.form.fields.pinNode')}</FormLabel>
                        <FormDescription>{t('kthena.form.descriptions.pinNode')}</FormDescription>
                      </div>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={(enabled) => {
                            field.onChange(enabled)
                            if (enabled) setSchedulingOpen(true)
                          }}
                        />
                      </FormControl>
                    </FormItem>
                  )}
                />
                {nodeSelectorEnabled && (
                  <FormField
                    control={form.control}
                    name="nodeSelector.nodes"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>
                          {t('kthena.form.fields.node')}
                          <FormLabelMust />
                        </FormLabel>
                        <FormControl>
                          <Combobox
                            items={nodeItems}
                            current={field.value?.[0] ?? ''}
                            handleSelect={(value) => field.onChange(value ? [value] : [])}
                            formTitle={t('kthena.form.selectNode')}
                          />
                        </FormControl>
                        <FormDescription>{t('kthena.form.descriptions.node')}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
              </div>
            </AccordionCard>

            <EnvFormCard
              form={form}
              open={envOpen}
              setOpen={setEnvOpen}
              cardTitle={t('kthena.form.sections.env')}
            />
            <AccordionCard
              cardTitle={t('kthena.form.sections.advanced')}
              icon={BracesIcon}
              open={advancedOpen}
              setOpen={setAdvancedOpen}
            >
              <div className="mt-3 grid gap-4">
                {modelSource === 'external' && (
                  <FormField
                    control={form.control}
                    name="cacheURI"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>{t('kthena.form.fields.cacheURI')}</FormLabel>
                        <FormControl>
                          <Input {...field} className="font-mono" />
                        </FormControl>
                        <FormDescription>{t('kthena.form.descriptions.cacheURI')}</FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                <div className="grid gap-3">
                  {configFields.map((field, index) => (
                    <div
                      key={field.id}
                      className="grid items-start gap-2 md:grid-cols-[1fr_1fr_auto]"
                    >
                      <FormField
                        control={form.control}
                        name={`configItems.${index}.key`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel hidden={index > 0}>
                              {t('kthena.form.fields.configKey')}
                            </FormLabel>
                            <FormControl>
                              <Input {...field} className="font-mono" />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={form.control}
                        name={`configItems.${index}.value`}
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel hidden={index > 0}>
                              {t('kthena.form.fields.configValue')}
                            </FormLabel>
                            <FormControl>
                              <Input {...field} className="font-mono" />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button
                        type="button"
                        variant="outline"
                        size="icon"
                        className={index > 0 ? '' : 'md:mt-6'}
                        onClick={() => removeConfig(index)}
                      >
                        <Trash2Icon className="size-4" />
                      </Button>
                    </div>
                  ))}
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={() => appendConfig({ key: '', value: '' })}
                  >
                    <CirclePlusIcon className="size-4" />
                    {t('kthena.form.actions.addConfig')}
                  </Button>
                </div>
                <FormDescription>{t('kthena.form.descriptions.workerConfig')}</FormDescription>
              </div>
            </AccordionCard>
            <AccordionCard
              cardTitle={t('kthena.form.sections.guide')}
              icon={VariableIcon}
              open={guideOpen}
              setOpen={setGuideOpen}
            >
              <div className="text-muted-foreground mt-3 space-y-3 text-sm leading-6">
                <p>{t('kthena.form.guide.model')}</p>
                <p>{t('kthena.form.guide.resources')}</p>
                <p>{t('kthena.form.guide.env')}</p>
                <p>{t('kthena.form.guide.advanced')}</p>
                <p>{t('kthena.form.guide.cache')}</p>
              </div>
            </AccordionCard>
          </aside>
        </div>
      </form>

      <Dialog
        open={templateDialogOpen}
        onOpenChange={(open) => {
          setTemplateDialogOpen(open)
          if (!open) setEditingTemplate(null)
        }}
      >
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle>
              {editingTemplate
                ? t('kthena.form.template.updateTitle')
                : t('kthena.form.template.saveTitle')}
            </DialogTitle>
            <DialogDescription>{t('kthena.form.template.dialogDescription')}</DialogDescription>
          </DialogHeader>
          <div className="grid gap-4 py-1">
            <div className="grid gap-2">
              <label htmlFor="kthena-template-name" className="text-sm font-medium">
                {t('kthena.form.template.name')}
              </label>
              <Input
                id="kthena-template-name"
                value={templateName}
                maxLength={64}
                autoFocus
                onChange={(event) => setTemplateName(event.target.value)}
              />
            </div>
            <div className="grid gap-2">
              <label htmlFor="kthena-template-description" className="text-sm font-medium">
                {t('kthena.form.template.description')}
              </label>
              <Input
                id="kthena-template-description"
                value={templateDescription}
                maxLength={512}
                onChange={(event) => setTemplateDescription(event.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              disabled={isSavingUserTemplate}
              onClick={() => setTemplateDialogOpen(false)}
            >
              {t('kthena.actions.cancel')}
            </Button>
            <Button type="button" disabled={isSavingUserTemplate} onClick={submitUserTemplate}>
              {isSavingUserTemplate && <Loader2Icon className="size-4 animate-spin" />}
              {editingTemplate ? t('kthena.form.template.update') : t('kthena.form.template.save')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog
        open={Boolean(templateToDelete)}
        onOpenChange={(open) => {
          if (!open) setTemplateToDelete(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.form.template.deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.form.template.deleteDescription', { name: templateToDelete?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeletingUserTemplate}>
              {t('kthena.actions.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              variant="destructive"
              disabled={isDeletingUserTemplate || !templateToDelete}
              onClick={() => {
                if (templateToDelete) deleteUserTemplate(templateToDelete.id)
              }}
            >
              {isDeletingUserTemplate && <Loader2Icon className="size-4 animate-spin" />}
              {t('kthena.form.template.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </Form>
  )
}

function DeploymentSummaryRow({
  label,
  value,
  mono = false,
}: {
  label: string
  value: string
  mono?: boolean
}) {
  return (
    <div className="grid gap-1.5">
      <span className="text-muted-foreground text-xs font-medium">{label}</span>
      <span
        className={['min-w-0 truncate text-sm font-medium', mono ? 'font-mono text-xs' : ''].join(
          ' '
        )}
        title={value}
      >
        {value}
      </span>
    </div>
  )
}
