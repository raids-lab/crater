import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useNavigate, useSearch } from '@tanstack/react-router'
import { useEffect, useRef, useState } from 'react'
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

import { FormSchema, defaultRole, defaultValues, formSchema } from './serving-form-schema'
import { fromRequest, toRequest } from './serving-serialization'
import { useServingTemplates } from './use-serving-templates'

export function useServingForm() {
  const { t } = useTranslation()
  const keys = useInferenceKeys()
  const queryClient = useQueryClient()
  const navigate = useNavigate({ from: '/portal/inference-services/new' })
  const { clone, edit } = useSearch({ from: '/portal/inference-services/new' })
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: defaultValues(),
  })
  const [step, setStep] = useState(0)
  const [preview, setPreview] = useState<ServingPreflight | null>(null)
  const [pendingCreate, setPendingCreate] = useState<FormSchema | null>(null)
  const previewFingerprint = useRef('')
  const values = form.watch()
  const scopeFingerprint = JSON.stringify(keys.scope)
  const previousScope = useRef(scopeFingerprint)
  useEffect(() => {
    if (previousScope.current !== scopeFingerprint) {
      previousScope.current = scopeFingerprint
      form.reset(defaultValues())
      setStep(0)
      setPreview(null)
      setPendingCreate(null)
    }
  }, [scopeFingerprint, form])
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
    queryKey: keys.detail(edit ?? clone),
    queryFn: () => apiGetKthenaService((edit ?? clone)!).then((r) => r.data),
    enabled: !!(edit ?? clone),
  })
  const cloned = useRef('')
  useEffect(() => {
    if (cloneQuery.data && cloned.current !== scopeFingerprint + cloneQuery.data.name) {
      cloned.current = scopeFingerprint + cloneQuery.data.name
      form.reset(
        fromRequest({
          ...cloneQuery.data,
          name: edit ? cloneQuery.data.name : `${cloneQuery.data.name}-copy`.slice(0, 63),
          replicas: Math.max(1, cloneQuery.data.desiredReplicas),
        })
      )
    }
  }, [cloneQuery.data, form, scopeFingerprint, edit])
  const templates = useServingTemplates(form, keys.scope)
  const preflight = useMutation({
    mutationFn: (req: CreateKthenaReq) => apiPreviewServing(req, edit ? 'update' : 'create'),
    onSuccess: (r, req) => {
      previewFingerprint.current = JSON.stringify(req)
      setPreview(r.data)
      setStep(3)
    },
    onError: showErrorToast,
  })
  const create = useMutation({
    mutationFn: (v: FormSchema) =>
      edit
        ? apiUpdateServing(edit, toRequest(v)).then(() => undefined)
        : apiCreateKthenaService(toRequest(v)).then(() => undefined),
    onSuccess: async (_, v) => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      toast.success(t('kthena.form.createSuccess', { name: v.name }))
      setPendingCreate(null)
      void navigate({ to: '/portal/inference-services/$name', params: { name: edit ?? v.name } })
    },
    onError: showErrorToast,
  })
  const previewCurrent =
    preview !== null && previewFingerprint.current === JSON.stringify(toRequest(values))
  const onSubmit = (v: FormSchema) => {
    if (step < 3 || !previewCurrent) {
      preflight.mutate(toRequest(v))
      return
    }
    setPendingCreate(v)
  }
  const next = async () => {
    const paths =
      step === 0
        ? (['name', 'platformModelId', 'modelURI', 'modelRevision', 'servedModel'] as const)
        : step === 1
          ? (['backendType', 'layout', 'replicas', 'port'] as const)
          : (['roles'] as const)
    if (await form.trigger([...paths])) {
      if (step === 2) {
        await form.handleSubmit((v) => preflight.mutate(toRequest(v)))()
      } else setStep(step + 1)
    }
  }
  const changeTopology = (engine: FormSchema['backendType'], layout: FormSchema['layout']) => {
    form.setValue('backendType', engine)
    form.setValue('layout', layout)
    form.setValue(
      'roles',
      layout === 'combined'
        ? [defaultRole('server', engine)]
        : [defaultRole('prefill', engine), defaultRole('decode', engine)],
      { shouldDirty: true }
    )
    setPreview(null)
  }
  return {
    t,
    clone,
    edit,
    form,
    values,
    step,
    setStep,
    next,
    preview,
    previewCurrent,
    preflight,
    modelQuery,
    nodeQuery,
    capabilities,
    cloneQuery,
    changeTopology,
    pendingCreate,
    setPendingCreate,
    createService: create.mutate,
    isPending: create.isPending,
    onSubmit,
    ...templates,
  }
}
export type ServingFormState = ReturnType<typeof useServingForm>
