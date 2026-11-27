import { apiV1Get } from '@/services/client'
import { IResponse } from '@/services/types'

// StatisticsScope 表示资源统计的查询范围。
export type StatisticsScope = 'user' | 'account' | 'cluster'
// TimeStep 表示时间序列的聚合粒度。
export type TimeStep = 'day' | 'week'
// IStatisticsReq 描述资源统计查询条件。
export interface IStatisticsReq {
  startTime: string
  endTime: string
  step: TimeStep
  scope: StatisticsScope
  targetID?: number
}
// IResourceDetail 描述单种资源的用量和利用率。
export interface IResourceDetail {
  usage: number
  effectiveUsage?: number
  profiledUsage?: number
  utilization?: number | null
  coverage?: number | null
  label: string
  type: string
}
// ITimePointData 描述一个时间区间的资源用量。
export interface ITimePointData {
  timestamp: string
  status: 'complete' | 'partial' | 'pending'
  usage: Record<string, number>
  effectiveUsage?: Record<string, number>
}
// IStatisticsProgress 描述日统计的覆盖与回填进度。
export interface IStatisticsProgress {
  ready: boolean
  firstDate: string
  completeBefore: string
  coveredUntil: string | null
  updatedAt: string | null
  pendingDays: number
  backfillComplete: boolean
}
// IStatisticsResp 包含资源汇总、时间序列和进度。
export interface IStatisticsResp {
  totalUsage: Record<string, IResourceDetail>
  series: ITimePointData[]
  progress: IStatisticsProgress
}
// IStatisticsRankingReq 描述资源排名查询条件。
export interface IStatisticsRankingReq extends IStatisticsReq {
  dimension: 'user' | 'account'
  metric: 'usage' | 'effectiveUsage' | 'utilization'
  order: 'asc' | 'desc'
  resourceName?: string
  resourceType?: 'gpu'
  page: number
  pageSize: number
}
// IStatisticsRankingRow 描述单个用户或账户的排名。
export interface IStatisticsRankingRow extends Required<Omit<IResourceDetail, 'label' | 'type'>> {
  rank: number
  id: number
  name: string
  nickname: string
}
// IStatisticsRankings 包含排名结果和统计进度。
export interface IStatisticsRankings {
  rows: IStatisticsRankingRow[]
  total: number
  progress: IStatisticsProgress
}
function statisticsSearchParams(params: IStatisticsReq | IStatisticsRankingReq) {
  return Object.fromEntries(Object.entries(params).filter(([, value]) => value !== undefined))
}

// apiGetStatistics 查询用户或账户的资源统计。
export const apiGetStatistics = (params: IStatisticsReq) =>
  apiV1Get<IResponse<IStatisticsResp>>('statistics', {
    searchParams: statisticsSearchParams(params),
  })

// apiGetStatisticsRankings 查询账户内的资源排名。
export const apiGetStatisticsRankings = (params: IStatisticsRankingReq) =>
  apiV1Get<IResponse<IStatisticsRankings>>('statistics/rankings', {
    searchParams: statisticsSearchParams(params),
  })

// apiAdminGetStatistics 查询管理员可见的资源统计。
export const apiAdminGetStatistics = (params: IStatisticsReq) =>
  apiV1Get<IResponse<IStatisticsResp>>('admin/statistics', {
    searchParams: statisticsSearchParams(params),
  })

// apiAdminGetStatisticsRankings 查询管理员可见的资源排名。
export const apiAdminGetStatisticsRankings = (params: IStatisticsRankingReq) =>
  apiV1Get<IResponse<IStatisticsRankings>>('admin/statistics/rankings', {
    searchParams: statisticsSearchParams(params),
  })
