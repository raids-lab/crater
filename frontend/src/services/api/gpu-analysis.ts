// 请将此文件保存或替换为 src/services/api/gpu-analysis.ts
import { apiV1Get, apiV1Post, apiV1Put } from '@/services/client'

import { IResponse } from '../types'
// 新增结束

/**
 * 对应于 Go 中 `model.GpuAnalysis` 与 `model.Job` 连接后的数据结构。
 * 此接口定义了列表接口返回的单条分析记录的结构。
 */
// src/services/api/gpu-analysis.ts
import { JobPhase } from './vcjob'

/**
 * Corresponds to the Go model `model.ReviewStatus`.
 * This enum defines the review status of an analysis record.
 * Backend definition: 1: Pending (待处理), 2: Confirmed (已确认占卡), 3: Ignored (已忽略)
 * Note: The backend only accepts Confirmed (2) or Ignored (3) for updates.
 */
export enum ReviewStatus {
  Pending = 1,
  Confirmed = 2,
  Ignored = 3,
}

// 新增：为来自关联 Job 的新字段定义类型
/**
 * 代表作业的类型，对应 Go 中的 `model.JobType`。
 * 例如: 'jupyter', 'pytorch', 'custom' 等。
 */
export type IJobType = string

/**
 * 代表 Kubernetes 的 ResourceList，对应 Go 中的 `v1.ResourceList`。
 * 它是一个从资源名称 (例如 'cpu', 'memory', 'nvidia.com/gpu') 到其数量的映射。
 */
export type IResourceList = Record<string, string>

// 从 vcjob API 文件中导入 JobPhase 类型

export interface IGpuAnalysis {
  ID: number
  CreatedAt: string // ISO 8601 date string
  DeletedAt?: string | null

  // 核心关联信息
  JobID: number
  JobName: string
  UserID: number
  UserName: string
  UserNickname: string

  // Kubernetes 信息
  PodName: string
  Namespace: string

  // LLM 分析结果
  Phase1Score: number
  Phase2Score: number
  Phase1LLMReason: string
  Phase2LLMReason: string
  LLMVersion: string

  // 采集的原始数据
  Command: string
  HistoricalMetrics: string

  // 管理状态
  ReviewStatus: ReviewStatus

  // 新增：为列表视图添加的关联 Job 字段
  /** 作业的类型 (例如 'pytorch', 'jupyter')。 */
  Name: string
  JobType: IJobType
  /** 作业所请求的资源。 */
  Resources: IResourceList
  /** 作业运行所在的节点。 */
  Nodes: string[]
  /** 新增：作业的当前状态 */
  status: JobPhase
  /**
   * 新增：作业的锁定时间。
   * 注意：为了简化，我们可以只检查这个时间戳是否存在且是否在未来，来判断作业是否被锁定。
   * JobNameCell 组件内部可能会处理这个逻辑。
   */
  lockedTimestamp?: string
}

/**
 * 从管理端点获取所有 GPU 分析记录的列表。
 * GET /v1/admin/gpu-analysis
 * 现在，响应中包含了来自关联 Job 的额外详细信息。
 */
export const apiAdminListGpuAnalyses = () =>
  apiV1Get<IResponse<IGpuAnalysis[]>>('admin/gpu-analysis')

/**
 * 更新特定分析记录的管理员审核状态。
 * PUT /v1/admin/gpu-analysis/{id}/review
 * @param id 分析记录的 ID。
 * @param reviewStatus 管理员的决定，例如 ReviewStatus.Confirmed 或 ReviewStatus.Ignored。
 */
export const apiAdminUpdateGpuAnalysisReviewStatus = (id: number, reviewStatus: ReviewStatus) =>
  apiV1Put<IResponse<string>>(`admin/gpu-analysis/${id}/review`, { reviewStatus })

/**
 * 触发对单个 pod 的分析。
 * POST /v1/admin/gpu-analysis/trigger/pod
 * @param namespace pod 的命名空间。
 * @param podName pod 的名称。
 * 注意: 此端点的响应是原始的 `GpuAnalysis` 模型，不包含额外的 Job 字段。
 * 不过，复用 IGpuAnalysis 接口是可接受的，因为额外的字段只会是 `undefined`，不会影响程序运行。
 */
export const apiAdminTriggerPodAnalysis = (namespace: string, podName: string) =>
  apiV1Post<IResponse<IGpuAnalysis>>('admin/gpu-analysis/trigger/pod', {
    namespace,
    podName,
  })

/**
 * 触发对单个 job 的分析。
 * POST /v1/admin/gpu-analysis/trigger/job
 * @param jobName job 的名称。
 */
export const apiAdminTriggerJobAnalysis = (jobName: string) =>
  apiV1Post<IResponse<IGpuAnalysis>>('admin/gpu-analysis/trigger/job', {
    jobName,
  })

/**
 * 触发对所有当前运行中的 job 的异步分析。
 * POST /v1/admin/gpu-analysis/trigger/all-jobs
 */
export const apiAdminTriggerAllJobsAnalysis = () =>
  apiV1Post<IResponse<{ queuedJobs: number; message: string }>>(
    'admin/gpu-analysis/trigger/all-jobs'
  )

/**
 * 确认占卡并停止作业。
 * POST /v1/admin/gpu-analysis/{id}/confirm-stop
 * 将分析记录标记为“已确认”，并立即停止（删除）对应的 Volcano Job。
 * @param id 分析记录ID
 */
export const apiAdminConfirmAndStopJob = (id: number) =>
  apiV1Post<IResponse<string>>(`admin/gpu-analysis/${id}/confirm-stop`)
