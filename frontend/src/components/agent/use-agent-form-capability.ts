import { useQuery } from '@tanstack/react-query'
import { useLocation } from '@tanstack/react-router'
import { useSetAtom } from 'jotai'
import { useEffect, useRef } from 'react'
import type { FieldPath, FieldPathValue, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import type {
  AgentFormCapability,
  AgentFormFieldDescriptor,
  AgentFormOption,
  AgentFormOptionSet,
  AgentFormPatch,
  AgentFormRuntime,
} from '@/components/agent/agent-context'

import { type Resource, apiResourceList } from '@/services/api/resource'
import { queryBaseImages } from '@/services/query/image'

import { atomAgentFormRuntime } from '@/utils/store'

interface UseAgentFormCapabilityOptions<TValues extends FieldValues> {
  capability: AgentFormCapability
  fields: readonly AgentFormFieldDescriptor[]
  form: UseFormReturn<TValues>
}

export function defineAgentFormFields<TValues extends FieldValues>() {
  return <TPath extends FieldPath<TValues>>(
    fields: readonly (AgentFormFieldDescriptor & { path: TPath })[]
  ) => fields
}

export function useAgentFormCapability<TValues extends FieldValues>({
  capability,
  fields,
  form,
}: UseAgentFormCapabilityOptions<TValues>) {
  const setRuntime = useSetAtom(atomAgentFormRuntime)
  const { t } = useTranslation()
  const pathname = useLocation({ select: (location) => location.pathname })
  const isJobForm = capability.id.startsWith('job.')
  const { data: resourceResponse, isError: resourceError } = useQuery({
    queryKey: ['resources', 'list'],
    queryFn: () => apiResourceList(true),
    enabled: isJobForm && fields.some((field) => field.path.endsWith('.resource.gpu.model')),
  })
  const { data: images, isError: imageError } = useQuery({
    ...queryBaseImages(),
    enabled: isJobForm && fields.some((field) => field.path.endsWith('.image.imageLink')),
  })
  const resourcesRef = useRef<Resource[] | undefined>(undefined)
  const imagesRef = useRef<typeof images>(undefined)
  const resourceErrorRef = useRef(false)
  const imageErrorRef = useRef(false)
  resourcesRef.current = resourceResponse?.data
  imagesRef.current = images
  resourceErrorRef.current = resourceError
  imageErrorRef.current = imageError

  useEffect(() => {
    const allowedFields = new Map(fields.map((field) => [field.path, field]))
    const gpuFields = fields.filter((field) => field.path.endsWith('.resource.gpu.model'))
    const imageFields = fields.filter((field) => field.path.endsWith('.image.imageLink'))
    const runtime: AgentFormRuntime = {
      instanceId: crypto.randomUUID(),
      capability: { ...capability, route: pathname },
      fields: fields.map((field) => ({ ...field, label: t(field.label) })),
      snapshot: () =>
        Object.fromEntries(
          fields
            .filter((field) => field.includeCurrentValue)
            .map((field) => [field.path, form.getValues(field.path as FieldPath<TValues>)])
        ),
      optionsReady: () =>
        (gpuFields.length === 0 ||
          resourcesRef.current !== undefined ||
          resourceErrorRef.current) &&
        (imageFields.length === 0 || imagesRef.current !== undefined || imageErrorRef.current),
      options: () => {
        const options: Record<string, AgentFormOptionSet> = {}
        const gpuItems = resourcesRef.current
          ?.filter((resource) => resource.amountSingleMax > 0 && resource.name.length <= 512)
          .map((resource) => ({
            value: resource.name,
            label: (resource.label || resource.name).slice(0, 512),
          }))
        const imageItems = imagesRef.current
          ?.filter((image) => image.value.length <= 512)
          .map((image) => ({
            value: image.value,
            label: (image.selectedLabel || image.value).slice(0, 256),
            tags: image.tags?.slice(0, 8).map((tag) => tag.slice(0, 64)),
          }))
          .sort((a, b) => imageOptionScore(b) - imageOptionScore(a))
        const perFieldBudget = Math.floor(
          (200 * 1024) / Math.max(1, gpuFields.length + imageFields.length)
        )
        for (const field of gpuFields) {
          if (gpuItems) options[field.path] = boundedOptions(gpuItems, perFieldBudget)
        }
        for (const field of imageFields) {
          if (imageItems) options[field.path] = boundedOptions(imageItems, perFieldBudget)
        }
        return options
      },
      apply: async (patches: readonly AgentFormPatch[]) => {
        const validatedPatches = patches.map((patch) => {
          const field = allowedFields.get(patch.path)
          if (!field || !matchesValueType(patch.value, field.valueType)) return undefined
          return { patch, path: patch.path as FieldPath<TValues> }
        })
        if (validatedPatches.some((patch) => patch === undefined)) return []

        const changes: { path: FieldPath<TValues>; value: unknown }[] = []
        for (const validated of validatedPatches) {
          if (!validated) continue
          const { patch, path } = validated
          if (path.endsWith('.resource.gpu.model')) {
            const options = resourcesRef.current
              ?.filter((resource) => resource.amountSingleMax > 0)
              .map((resource) => ({ value: resource.name, label: resource.label }))
            const selected = resolveUniqueOption(options, patch.value as string, true)
            if (!selected) throw new Error(t('craterAgent.formOptionUnavailable'))
            changes.push({ path, value: selected.value })
          } else if (path.endsWith('.image.imageLink')) {
            const selected = resolveUniqueOption(imagesRef.current, patch.value as string)
            if (!selected) throw new Error(t('craterAgent.formOptionUnavailable'))
            changes.push({
              path: path.slice(0, -'.imageLink'.length) as FieldPath<TValues>,
              value: {
                imageLink: selected.detail?.imageLink ?? selected.value,
                archs: selected.detail?.archs ?? [],
              },
            })
          } else {
            changes.push({ path, value: patch.value })
          }
        }

        const changedPaths: FieldPath<TValues>[] = []
        const setField = (path: FieldPath<TValues>, value: unknown) => {
          form.setValue(path, value as FieldPathValue<TValues, typeof path>, {
            shouldDirty: true,
            shouldTouch: true,
          })
          changedPaths.push(path)
        }
        for (const { path, value } of changes) {
          if (!path.endsWith('.resource.gpu.model') || form.getValues(path) === value) continue
          const resourcePath = path.slice(0, -'.gpu.model'.length)
          setField(`${resourcePath}.network.enabled` as FieldPath<TValues>, false)
          setField(`${resourcePath}.network.model` as FieldPath<TValues>, undefined)
          setField(`${resourcePath}.vgpu.enabled` as FieldPath<TValues>, false)
          setField(`${resourcePath}.vgpu.models` as FieldPath<TValues>, undefined)
        }
        for (const { path, value } of changes) {
          setField(path, value)
        }
        if (changedPaths.length > 0) await form.trigger([...new Set(changedPaths)])
        return changedPaths
      },
    }
    setRuntime(runtime)
    return () => setRuntime((current) => (current === runtime ? undefined : current))
  }, [capability, fields, form, setRuntime, t, pathname])
}

function imageOptionScore(option: AgentFormOption): number {
  const text = [option.value, option.label, ...(option.tags ?? [])].join(' ').toLowerCase()
  return (
    (/(python|jupyter)/.test(text) ? 4 : 0) +
    (/(basic|base|基础|通用)/.test(text) ? 2 : 0) +
    (/(cuda|pytorch|torch)/.test(text) ? 1 : 0)
  )
}

function boundedOptions(items: AgentFormOption[], byteBudget: number): AgentFormOptionSet {
  const selected: AgentFormOption[] = []
  const encoder = new TextEncoder()
  let usedBytes = 0
  for (const item of items) {
    if (selected.length >= 512) break
    const itemBytes = encoder.encode(JSON.stringify(item)).length
    if (usedBytes + itemBytes > byteBudget) break
    selected.push(item)
    usedBytes += itemBytes
  }
  return { items: selected, total: items.length }
}

function resolveUniqueOption<T extends { value: string; label: string; selectedLabel?: string }>(
  options: readonly T[] | undefined,
  requested: string,
  allowQualifiedSuffix = false
): T | undefined {
  const normalized = requested.trim().toLowerCase()
  if (!normalized) return undefined
  const matches = options?.filter((option) =>
    [
      option.value,
      option.label,
      option.selectedLabel,
      allowQualifiedSuffix && option.value.split('/').at(-1),
    ]
      .filter((candidate): candidate is string => typeof candidate === 'string')
      .some((candidate) => candidate.toLowerCase() === normalized)
  )
  if (matches?.length === 1) return matches[0]
  if (!allowQualifiedSuffix || matches?.length) return undefined
  const normalizedGpu = normalizeGpuName(requested)
  if (!normalizedGpu) return undefined
  const aliases = options?.filter((option) =>
    [option.value.split('/').at(-1), option.label]
      .filter((candidate): candidate is string => typeof candidate === 'string')
      .some((candidate) => normalizeGpuName(candidate) === normalizedGpu)
  )
  return aliases?.length === 1 ? aliases[0] : undefined
}

function normalizeGpuName(value: string): string {
  return value
    .toLowerCase()
    .replace(/nvidia|tesla|gpu|gib|gb/g, '')
    .replace(/[^a-z0-9]/g, '')
}

function matchesValueType(value: unknown, valueType: AgentFormFieldDescriptor['valueType']) {
  switch (valueType) {
    case 'string':
      return typeof value === 'string'
    case 'number':
      return typeof value === 'number' && Number.isFinite(value)
    case 'boolean':
      return typeof value === 'boolean'
    case 'string-array':
      return Array.isArray(value) && value.every((item) => typeof item === 'string')
    case 'option-array':
      return Array.isArray(value) && value.every(isAgentOption)
    case 'volume-mount-array':
      return Array.isArray(value) && value.every(isAgentVolumeMount)
    case 'forward-array':
      return Array.isArray(value) && value.every(isAgentForward)
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function isAgentOption(value: unknown) {
  return isRecord(value) && typeof value.value === 'string'
}

function isAgentVolumeMount(value: unknown) {
  return (
    isRecord(value) &&
    typeof value.type === 'number' &&
    Number.isInteger(value.type) &&
    (value.type === 1 || value.type === 2) &&
    typeof value.subPath === 'string' &&
    typeof value.mountPath === 'string' &&
    (value.datasetID === undefined ||
      (typeof value.datasetID === 'number' &&
        Number.isInteger(value.datasetID) &&
        value.datasetID >= 0))
  )
}

function isAgentForward(value: unknown) {
  return (
    isRecord(value) &&
    typeof value.name === 'string' &&
    typeof value.port === 'number' &&
    Number.isInteger(value.port) &&
    value.port >= 1 &&
    value.port <= 65_535
  )
}
