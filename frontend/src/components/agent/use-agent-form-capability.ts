import { useSetAtom } from 'jotai'
import { useEffect } from 'react'
import type { FieldPath, FieldPathValue, FieldValues, UseFormReturn } from 'react-hook-form'

import type {
  AgentFormCapability,
  AgentFormFieldDescriptor,
  AgentFormPatch,
  AgentFormRuntime,
} from '@/components/agent/agent-context'

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

  useEffect(() => {
    const allowedFields = new Map(fields.map((field) => [field.path, field]))
    const runtime: AgentFormRuntime = {
      capability,
      fields,
      snapshot: () =>
        Object.fromEntries(
          fields
            .filter((field) => field.includeCurrentValue)
            .map((field) => [field.path, form.getValues(field.path as FieldPath<TValues>)])
        ),
      apply: async (patches: readonly AgentFormPatch[]) => {
        const validatedPatches = patches.map((patch) => {
          const field = allowedFields.get(patch.path)
          if (!field || !matchesValueType(patch.value, field.valueType)) return undefined
          return { patch, path: patch.path as FieldPath<TValues> }
        })
        if (validatedPatches.some((patch) => patch === undefined)) return []

        const changedPaths: FieldPath<TValues>[] = []
        for (const validated of validatedPatches) {
          if (!validated) continue
          const { patch, path } = validated
          form.setValue(path, patch.value as FieldPathValue<TValues, typeof path>, {
            shouldDirty: true,
            shouldTouch: true,
          })
          changedPaths.push(path)
        }
        if (changedPaths.length > 0) await form.trigger(changedPaths)
        return changedPaths
      },
    }
    setRuntime(runtime)
    return () => setRuntime((current) => (current === runtime ? undefined : current))
  }, [capability, fields, form, setRuntime])
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
