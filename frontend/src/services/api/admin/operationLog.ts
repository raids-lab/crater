import { apiV1Delete, apiV1Get } from '@/services/client'
import { IResponse, IWithPagination } from '@/services/types'

export interface IOperationLog {
  id: number
  created_at: string
  updated_at: string
  operator: string
  operator_role: string
  operation_type: string
  target: string
  details: Record<string, any>
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
}

export const getOperationLogs = async (params: IGetOperationLogsParams) => {
  return await apiV1Get<IOperationLogResponse>('admin/operation-logs', {
    searchParams: params as any,
  })
}

export const clearOperationLogs = async () => {
  return await apiV1Delete<IResponse<null>>('admin/operation-logs')
}
