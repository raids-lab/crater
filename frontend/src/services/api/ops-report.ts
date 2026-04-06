import { apiV1Get } from '@/services/client'

// ─── Types ──────────────────────────────────────────────────────────────────

export interface OpsReportListItem {
  id: string
  report_type: string
  status: string
  trigger_source: string | null
  summary: Record<string, unknown>
  period_start: string | null
  period_end: string | null
  job_total: number
  job_success: number
  job_failed: number
  job_pending: number
  created_at: string
}

export interface OpsReportDetail extends OpsReportListItem {
  report_json: OpsReportJSON | null
}

export interface OpsReportJSON {
  executive_summary: string
  job_overview: {
    total: number
    success: number
    failed: number
    pending: number
    success_rate: number
    delta: { total: number; failed: number; pending: number }
  }
  failure_analysis: {
    categories: Array<{
      reason: string
      count: number
      top_job: { name: string; owner: string }
    }>
    top_affected_users: string[]
    patterns: string
  }
  success_analysis: {
    avg_duration_by_type: Record<string, number>
    resource_efficiency: {
      avg_cpu_ratio: number
      avg_gpu_ratio: number
      avg_memory_ratio: number
    }
  }
  resource_utilization: {
    cluster_gpu_avg: number
    cluster_cpu_avg: number
    cluster_memory_avg: number
    over_provisioned_count: number
    idle_gpu_jobs: number
    node_hotspots: string[]
  }
  recommendations: Array<{
    severity: 'high' | 'medium' | 'low'
    text: string
  }>
}

export interface OpsAuditItem {
  id: number
  report_id: string
  job_name: string
  username: string | null
  action_type: string
  severity: string
  category: string | null
  job_type: string | null
  owner: string | null
  namespace: string | null
  duration_seconds: number | null
  gpu_utilization: number | null
  gpu_requested: number | null
  gpu_actual_used: number | null
  resource_requested: Record<string, unknown> | null
  resource_actual: Record<string, unknown> | null
  exit_code: number | null
  failure_reason: string | null
  analysis_detail: Record<string, unknown> | null
  handled: boolean
  created_at: string
}

export interface OpsReportListResponse {
  total: number
  items: OpsReportListItem[]
}

export interface OpsAuditItemsResponse {
  total: number
  items: OpsAuditItem[]
}

// ─── API Functions ──────────────────────────────────────────────────────────

export const apiListOpsReports = (page = 1, pageSize = 20) =>
  apiV1Get<OpsReportListResponse>('admin/agent/ops-reports', {
    searchParams: { page, page_size: pageSize },
  })

export const apiGetLatestOpsReport = () =>
  apiV1Get<OpsReportDetail>('admin/agent/ops-reports/latest')

export const apiGetOpsReportDetail = (reportId: string) =>
  apiV1Get<OpsReportDetail>(`admin/agent/ops-reports/${reportId}`)

export const apiGetOpsReportItems = (
  reportId: string,
  params?: { category?: string; severity?: string; page?: number; page_size?: number }
) =>
  apiV1Get<OpsAuditItemsResponse>(`admin/agent/ops-reports/${reportId}/items`, {
    searchParams: params as Record<string, string | number>,
  })
