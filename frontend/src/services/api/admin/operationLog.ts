import {
  type RemoteTableParams,
  buildRemoteSearchParams,
} from '@/components/query-table/remote-state'

import { apiV1Delete, apiV1Get } from '@/services/client'
import { IPage, IResponse, IWithPagination } from '@/services/types'

export type JsonPrimitive = string | number | boolean | null
export type JsonValue = JsonPrimitive | JsonObject | JsonValue[]
export type JsonObject = { [key: string]: JsonValue }

export interface IOperationLog {
  id: number
  created_at: string
  updated_at: string
  operator: string
  operator_role: string
  operation_type: string
  target: string
  details: JsonObject
  status: string
  error_message?: string
}

export type IOperationLogResponse = IResponse<IWithPagination<IOperationLog>>

export interface IGetOperationLogsParams {
  page?: number
  limit?: number
  operator?: string
  operation_type?: string
  target?: string
  start_time?: string
  end_time?: string
  search?: string
}

type OperationLogSearchParams = Record<string, string | number | boolean>

export const getOperationLogs = async (params: IGetOperationLogsParams) => {
  const searchParams = Object.fromEntries(
    Object.entries(params).filter(([, value]) => value !== undefined && value !== '')
  ) as OperationLogSearchParams

  return await apiV1Get<IOperationLogResponse>('admin/operation-logs', {
    searchParams,
  })
}

export const getOperationLogsPaged = async (
  params: RemoteTableParams,
  timeRange?: { start_time?: string; end_time?: string },
  signal?: AbortSignal
) => {
  const searchParams = buildRemoteSearchParams(params)
  if (timeRange?.start_time) searchParams.set('start_time', timeRange.start_time)
  if (timeRange?.end_time) searchParams.set('end_time', timeRange.end_time)

  return await apiV1Get<IResponse<IPage<IOperationLog>>>('admin/operation-logs/page', {
    searchParams,
    signal,
  })
}

export const clearOperationLogs = async () => {
  return await apiV1Delete<IResponse<Record<string, string>>>('admin/operation-logs')
}
