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
        const changedPaths: FieldPath<TValues>[] = []
        for (const patch of patches) {
          const field = allowedFields.get(patch.path)
          if (!field || !matchesValueType(patch.value, field.valueType)) continue
          const path = patch.path as FieldPath<TValues>
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
    case 'json':
      return value !== undefined
  }
}
