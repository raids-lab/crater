export type AgentActionKind =
  | 'choose-job-type'
  | 'choose-template'
  | 'assist-form'
  | 'find-platform-resource'
  | 'recreate-job'
  | 'diagnose-job'

export type AgentFieldValueType =
  | 'string'
  | 'number'
  | 'boolean'
  | 'string-array'
  | 'option-array'
  | 'volume-mount-array'
  | 'forward-array'

export interface AgentFormFieldDescriptor {
  path: string
  label: string
  valueType: AgentFieldValueType
  description?: string
  includeCurrentValue?: boolean
}

export interface AgentFormCapability {
  id: string
  schemaVersion: number
  title: string
  route: string
}

export interface AgentActionContext {
  action: AgentActionKind
  title: string
  prompt: string
  route: string
  capabilityId?: string
  resource?: AgentResourceContext
}

export interface AgentEntryRequest {
  id: string
  initialMessage: string
}

export interface AgentResourceContext {
  kind: 'job'
  route: string
  jobName: string
  status?: string
  jobType?: string
}

export interface AgentFormPatch {
  path: string
  value: unknown
  reason: string
}

export interface AgentFormRuntime {
  capability: AgentFormCapability
  fields: readonly AgentFormFieldDescriptor[]
  snapshot: () => Record<string, unknown>
  apply: (patches: readonly AgentFormPatch[]) => Promise<string[]>
}

export interface AgentPageContext {
  route: string
  action: AgentActionKind
  title: string
  prompt: string
  form?: {
    capabilityId: string
    schemaVersion: number
    title: string
    fields: readonly AgentFormFieldDescriptor[]
    currentValues: Record<string, unknown>
  }
  resource?: Omit<AgentResourceContext, 'route'>
}
