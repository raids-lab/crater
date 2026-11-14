import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
import type { FormEvent } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { apiGetNodes } from '@/services/api/cluster'
import { apiGetDataset } from '@/services/api/dataset'
import {
  CreateKthenaReq,
  ServingPreflight,
  apiCreateKthenaService,
  apiGetKthenaService,
  apiPreviewServing,
  apiServingCapabilities,
  apiUpdateServing,
} from '@/services/api/inference'

import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { showErrorToast } from '@/utils/toast'

import {
  FormSchema,
  defaultRole,
  defaultValues,
  formSubmitSchema,
  selectedServingImage,
  servingFormErrorMap,
} from './serving-form-schema'
import { fromRequest, toRequest } from './serving-serialization'
import { useServingTemplates } from './use-serving-templates'

const fieldsByStep: (keyof FormSchema)[][] = [
  ['name', 'modelSource', 'platformModelId', 'modelURI', 'modelRevision'],
  ['backendType', 'layout', 'replicas', 'port', 'servedModel'],
  ['roles'],
]

type PreviewRequest = {
  req: CreateKthenaReq
  mode: 'create' | 'update'
  context: string
  version: number
}

const requestFingerprint = (values: FormSchema) => JSON.stringify(toRequest(values))

export function useServingForm() {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const queryClient = useQueryClient()
  const navigate = useNavigate({ from: '/portal/inference-services/new' })
  const { clone, edit } = useSearch({ from: '/portal/inference-services/new' })
  const source = edit ?? clone
  const context = JSON.stringify([keys.scope, edit, clone])
  const activeContext = useRef(context)
  const validationVersion = useRef(0)
  const topologyDrafts = useRef(new Map<string, FormSchema['roles']>())
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSubmitSchema, { errorMap: servingFormErrorMap }),
    defaultValues: defaultValues(),
  })
  const [step, setStep] = useState(0)
  const [furthestStep, setFurthestStep] = useState(0)
  const [stepError, setStepError] = useState<string | null>(null)
  const [validationStep, setValidationStep] = useState<number | null>(null)
  const [preview, setPreview] = useState<{
    data: ServingPreflight
    fingerprint: string
    context: string
  } | null>(null)
  const [pendingCreate, setPendingCreate] = useState<FormSchema | null>(null)
  const [loadedSource, setLoadedSource] = useState<string>()
  const values = form.watch()
  const formDefaults = form.formState.defaultValues
  useEffect(() => {
    topologyDrafts.current.clear()
  }, [formDefaults])
  useEffect(() => {
    if (activeContext.current !== context) {
      activeContext.current = context
      validationVersion.current++
      form.reset(defaultValues())
      setStep(0)
      setFurthestStep(0)
      setStepError(null)
      setValidationStep(null)
      setPreview(null)
      setPendingCreate(null)
      setLoadedSource(undefined)
    }
  }, [context, form])
  const modelQuery = useQuery({
    queryKey: ['dataset', 'my-models', ...keys.scope],
    queryFn: apiGetDataset,
    select: (r) => r.data.filter((d) => d.type === 'model'),
  })
  const nodeQuery = useQuery({
    queryKey: ['cluster', 'nodes', 'brief', ...keys.scope],
    queryFn: apiGetNodes,
    select: (r) => r.data.map((n) => ({ value: n.name, label: n.name })),
  })
  const capabilities = useQuery({
    queryKey: ['kthena', 'capabilities', ...keys.scope],
    queryFn: () => apiServingCapabilities().then((r) => r.data),
  })
  const cloneQuery = useQuery({
    queryKey: keys.detail(source),
    queryFn: () => apiGetKthenaService(source!).then((r) => r.data),
    enabled: !!source,
  })
  useEffect(() => {
    if (cloneQuery.data && cloneQuery.data.name === source && loadedSource !== context) {
      form.reset(
        fromRequest({
          ...cloneQuery.data,
          name: edit ? source : `${source.slice(0, 58).replace(/-+$/, '')}-copy`,
          replicas: Math.max(1, cloneQuery.data.desiredReplicas),
        })
      )
      setLoadedSource(context)
    }
  }, [cloneQuery.data, form, context, source, loadedSource, edit])
  const sourceLoading = !!source && loadedSource !== context && !cloneQuery.isError
  const sourceError = source && loadedSource !== context ? cloneQuery.error : null
  const templates = useServingTemplates(form, keys.scope)
  const isCurrentPreviewRequest = ({ context, req, version }: PreviewRequest) =>
    activeContext.current === context &&
    validationVersion.current === version &&
    requestFingerprint(form.getValues()) === JSON.stringify(req)
  const preflight = useMutation({
    mutationFn: ({ req, mode }: PreviewRequest) => apiPreviewServing(req, mode),
    onSuccess: (r, request) => {
      if (!isCurrentPreviewRequest(request)) return
      setPreview({
        data: r.data,
        fingerprint: JSON.stringify(request.req),
        context: request.context,
      })
      setStep(3)
      setFurthestStep(3)
    },
    onError: (error, request) => {
      if (isCurrentPreviewRequest(request)) showErrorToast(error)
    },
  })
  const create = useMutation({
    mutationFn: ({ req, target }: { req: CreateKthenaReq; target?: string; context: string }) =>
      target
        ? apiUpdateServing(target, req).then(() => undefined)
        : apiCreateKthenaService(req).then(() => undefined),
    onSuccess: async (_, { req, target, context }) => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      if (activeContext.current !== context) return
      toast.success(t('kthena.form.createSuccess', { name: req.name }))
      setPendingCreate(null)
      void navigate({
        to: '/portal/inference-services/$name',
        params: { name: target ?? req.name },
      })
    },
    onError: (error, { context }) => {
      if (activeContext.current === context) showErrorToast(error)
    },
  })
  const matchesPreview = (v: FormSchema) =>
    preview?.context === context && preview.fingerprint === requestFingerprint(v)
  const previewCurrent = matchesPreview(values)
  const revealError = (index: number, message: string | null) => {
    setStep(index)
    setValidationStep(index)
    setStepError(message)
  }
  const checkCapabilities = (v: FormSchema) => {
    const profiles = capabilities.data
    if (!profiles?.length || profiles.some((profile) => profile.evidence === 'unknown')) return true
    for (const [index, role] of v.roles.entries()) {
      const profile = profiles.find(
        (item) =>
          item.engine === v.backendType &&
          item.layout === v.layout &&
          item.execution === role.execution
      )
      if (!profile?.enabled) {
        const message = t('kthena.form.validation.profileUnavailable')
        form.setError(`roles.${index}.execution`, { type: 'capability', message })
        revealError(2, message)
        return false
      }
      for (const kind of ['entry', 'worker'] as const) {
        const pod = role[kind]
        if (!pod || (kind === 'worker' && role.execution !== 'ray')) continue
        if (profile.images.includes(selectedServingImage(pod) ?? '')) continue
        const message = t('kthena.topology.unapprovedImage')
        const imageField = pod.imageSource === 'platform' ? 'platformImage' : 'image'
        form.setError(`roles.${index}.${kind}.${imageField}`, { type: 'capability', message })
        revealError(2, message)
        return false
      }
    }
    return true
  }
  const goToStep = async (target: number, submit = false) => {
    if (sourceLoading || sourceError || create.isPending || target < 0 || target > 3) return
    const version = ++validationVersion.current
    if (target < step) {
      setStep(target)
      return
    }
    const valid = await form.trigger(fieldsByStep.slice(0, target).flat())
    if (activeContext.current !== context || validationVersion.current !== version) return
    const draft = form.getValues()
    if (!valid) {
      const invalidStep = fieldsByStep.findIndex((fields) =>
        fields.some((field) => form.getFieldState(field).invalid)
      )
      const index = invalidStep < 0 ? Math.max(0, target - 1) : invalidStep
      const result = formSubmitSchema.safeParse(draft, { errorMap: servingFormErrorMap })
      const issue = result.success
        ? undefined
        : result.error.issues.find(({ path }) =>
            fieldsByStep[index].includes(path[0] as keyof FormSchema)
          )
      revealError(index, issue?.message ?? null)
      return
    }
    setValidationStep(null)
    setStepError(null)
    if (target === 3) {
      if (!checkCapabilities(draft)) return
      if (!matchesPreview(draft)) {
        preflight.mutate({
          req: toRequest(draft),
          mode: edit ? 'update' : 'create',
          context,
          version,
        })
        return
      }
      if (submit) setPendingCreate(structuredClone(draft))
    }
    setStep(target)
    setFurthestStep((previous) => Math.max(previous, target))
  }
  const next = () => goToStep(step + 1)
  const submitForm = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (preflight.isPending || create.isPending) return
    if (step < 3) void next()
    else void goToStep(3, true)
  }
  const confirmCreate = () => {
    if (!pendingCreate || create.isPending) return
    if (!matchesPreview(pendingCreate) || !matchesPreview(form.getValues())) {
      setPendingCreate(null)
      toast.error(t('kthena.topology.stale'))
      return
    }
    create.mutate({ req: toRequest(pendingCreate), target: edit, context })
  }
  const changeTopology = (engine: FormSchema['backendType'], layout: FormSchema['layout']) => {
    const current = form.getValues()
    if (current.backendType === engine && current.layout === layout) return
    validationVersion.current++
    const previousTopology = `${current.backendType}/${current.layout}`
    const nextTopology = `${engine}/${layout}`
    topologyDrafts.current.set(previousTopology, structuredClone(current.roles))
    const names = layout === 'combined' ? (['server'] as const) : (['prefill', 'decode'] as const)
    const roles =
      topologyDrafts.current.get(nextTopology) ??
      names.map((name) => {
        const prior = current.roles.find((role) => role.name === name) ?? current.roles[0]
        if (prior && current.backendType === engine) return { ...prior, name }
        const role = defaultRole(name, engine)
        if (!prior) return role
        return {
          ...role,
          instances: prior.instances,
          entry: {
            ...prior.entry,
            imageSource: 'manual' as const,
            image: role.entry.image,
            imageArchs: [],
          },
        }
      })
    form.setValue('backendType', engine, { shouldDirty: true })
    form.setValue('layout', layout, { shouldDirty: true })
    form.setValue('roles', structuredClone(roles), { shouldDirty: true })
    setPreview(null)
  }
  return {
    t,
    clone,
    edit,
    form,
    values,
    step,
    furthestStep,
    goToStep,
    validationStep,
    stepError: validationStep === step ? stepError : null,
    hasStepError: validationStep === step,
    next,
    preview: preview?.context === context ? preview.data : null,
    previewCurrent,
    preflight,
    modelQuery,
    nodeQuery,
    capabilities,
    cloneQuery,
    sourceLoading,
    sourceError,
    changeTopology,
    pendingCreate,
    setPendingCreate,
    confirmCreate,
    isPending: create.isPending,
    submitForm,
    ...templates,
  }
}
export type ServingFormState = ReturnType<typeof useServingForm>
