// src/services/api/statistics.ts
import { apiV1Get } from '@/services/client'
import { IResponse } from '@/services/types'

export type StatisticsScope = 'user' | 'account' | 'cluster'
export type TimeStep = 'day' | 'week'

export interface IStatisticsReq {
  startTime: string // RFC3339
  endTime: string // RFC3339
  step: TimeStep
  scope: StatisticsScope
  targetID?: number
}

export interface IResourceDetail {
  usage: number
  label: string
  type: string
}

export interface ITimePointData {
  timestamp: string
  usage: Record<string, number>
}

export interface IStatisticsResp {
  totalUsage: Record<string, IResourceDetail>
  series: ITimePointData[]
}

/**
 * 获取资源统计信息
 */
export const apiGetStatistics = (params: IStatisticsReq) =>
  apiV1Get<IResponse<IStatisticsResp>>('statistics', {
    // 修正点：必须通过 searchParams 传递查询参数
    // 使用 as any 或 as Record<...> 避免类型定义的细微不兼容
    searchParams: params as unknown as Record<string, string | number | boolean>,
  })
