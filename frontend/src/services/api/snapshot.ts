import type { INodeBriefInfo } from '@/services/api/cluster'
import { apiV1Get } from '@/services/client'
import type { IResponse } from '@/services/types'

export interface SnapshotPage<T> {
  items: T[]
  next_cursor: string | null
  has_more: boolean
  snapshot_id: string
  resource_version?: string
  observed_at: string
}

export interface SnapshotQuery {
  limit?: number
  cursor?: string
  selector?: string
  resource_version?: string
}

export interface SnapshotPod {
  name: string
  namespace: string
  nodeName?: string
  ip: string
  status: string
  resources: Record<string, string>
  requestResources?: Record<string, string>
  createTime: string
  startTime?: string
}

export const snapshotParams = (params: SnapshotQuery) => {
  const searchParams = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== '') searchParams.set(key, String(value))
  }
  return searchParams
}

export const apiGetNodeSnapshot = (params: SnapshotQuery = {}) =>
  apiV1Get<IResponse<SnapshotPage<INodeBriefInfo>>>('resources/snapshot', {
    searchParams: snapshotParams(params),
  })

export const apiGetJobPodSnapshot = (jobName: string, params: SnapshotQuery = {}) =>
  apiV1Get<IResponse<SnapshotPage<SnapshotPod>>>(`vcjobs/${jobName}/pods/snapshot`, {
    searchParams: snapshotParams(params),
  })
