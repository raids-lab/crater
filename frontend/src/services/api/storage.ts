import { apiGet, apiPost, apiPut } from '@/services/client'
import { IResponse } from '@/services/types'

export interface UserSpace {
  user: string
  size: number
  quota: number
  unit: string
  formatted: string
  quota_formatted: string
  is_expanded: boolean
  jobs_frozen: boolean
  shrink_stage?: string
  original_quota?: number
  original_quota_formatted?: string
}

export interface PagedUserSpaces {
  items: UserSpace[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export const apiAdminGetUserSpaces = (
  page: number = 1,
  pageSize: number = 10
): Promise<IResponse<PagedUserSpaces>> =>
  apiGet<IResponse<PagedUserSpaces>>(
    `v1/admin/storage/user-spaces?page=${page}&pageSize=${pageSize}`
  )

export interface SetQuotaRequest {
  quota: number
}

export interface SetQuotaResponse {
  user: string
  quota: number
  unit: string
  quota_formatted: string
  ceph_quota_set: boolean
  ceph_quota_error: string | null
}

export const apiAdminSetUserSpaceQuota = (
  user: string,
  quota: number
): Promise<IResponse<SetQuotaResponse>> =>
  apiPut<IResponse<SetQuotaResponse>>(`v1/admin/storage/user-spaces/${user}/quota`, { quota })

export interface AutoScaleRequest {
  min_quota: number
  max_quota: number
  scale_up_ratio: number
  scale_down_ratio: number
}

export interface AutoScaleResponse {
  user: string
  current_usage: number
  new_quota: number
  unit: string
  current_usage_formatted: string
  new_quota_formatted: string
  ceph_quota_set: boolean
  ceph_quota_error: string | null
}

export const apiAdminAutoScaleUserSpaceQuota = (
  user: string,
  params: AutoScaleRequest
): Promise<IResponse<AutoScaleResponse>> =>
  apiPost<IResponse<AutoScaleResponse>>(`v1/admin/storage/user-spaces/${user}/autoscale`, params)

export interface LLMDecisionResponse {
  allow_expand: boolean
  expand_bytes: number
  freeze_new_jobs: boolean
  reason: string
}

export interface LLMJobStatus {
  status: 'pending' | 'done' | 'error'
  result?: LLMDecisionResponse
  error?: string
  constraint_adjusted?: boolean
  constraint_blocked?: boolean
}

export interface ApplyExpansionRequest {
  expand_bytes: number
  freeze_new_jobs?: boolean
  decision_job_id?: string
}

export interface ApplyExpansionResponse {
  user: string
  original_quota: number
  new_quota: number
  original_quota_formatted: string
  new_quota_formatted: string
}

export interface RevertExpansionResponse {
  user: string
  reverted_quota: number
  reverted_quota_formatted: string
  jobs_unfrozen: boolean
}

export const apiAdminApplyExpansion = (
  user: string,
  expandBytes: number,
  freezeNewJobs: boolean,
  decisionJobId?: string
): Promise<IResponse<ApplyExpansionResponse>> =>
  apiPost<IResponse<ApplyExpansionResponse>>(
    `v1/admin/storage/user-spaces/${user}/apply-expansion`,
    {
      expand_bytes: expandBytes,
      freeze_new_jobs: freezeNewJobs,
      decision_job_id: decisionJobId,
    }
  )

export const apiAdminRevertExpansion = (
  user: string
): Promise<IResponse<RevertExpansionResponse>> =>
  apiPost<IResponse<RevertExpansionResponse>>(
    `v1/admin/storage/user-spaces/${user}/revert-expansion`
  )

export const apiAdminUnfreezeJobs = (
  user: string
): Promise<IResponse<{ user: string; jobs_frozen: boolean }>> =>
  apiPost<IResponse<{ user: string; jobs_frozen: boolean }>>(
    `v1/admin/storage/user-spaces/${user}/unfreeze-jobs`
  )

export const apiAdminFreezeJobs = (
  user: string,
  decisionJobId?: string
): Promise<IResponse<{ user: string; jobs_frozen: boolean }>> =>
  apiPost<IResponse<{ user: string; jobs_frozen: boolean }>>(
    `v1/admin/storage/user-spaces/${user}/freeze-jobs`,
    { decision_job_id: decisionJobId }
  )

export const apiAdminStartLLMDecision = (user: string): Promise<IResponse<{ job_id: string }>> =>
  apiPost<IResponse<{ job_id: string }>>(`v1/admin/storage/user-spaces/${user}/llm-decision`)

export const apiAdminGetLLMDecisionStatus = (
  user: string,
  jobId: string
): Promise<IResponse<LLMJobStatus>> =>
  apiGet<IResponse<LLMJobStatus>>(`v1/admin/storage/user-spaces/${user}/llm-decision/${jobId}`)

export type StorageDecisionSource = 'manual' | 'patrol' | 'replay'
export type StorageDecisionStatus = 'pending' | 'running' | 'done' | 'error'

export interface StorageDecisionRecordSummary {
  job_id: string
  username: string
  source: StorageDecisionSource
  status: StorageDecisionStatus
  trigger_reason: string
  raw_allow_expand: boolean
  raw_expand_bytes: number
  raw_freeze_new_jobs: boolean
  final_allow_expand: boolean
  final_expand_bytes: number
  final_freeze_new_jobs: boolean
  constraint_adjusted: boolean
  constraint_blocked: boolean
  applied_action: string
  error_message: string
  constraint_version: string
  latency_ms: number
  created_at: string
  updated_at: string
}

export interface StorageDecisionPage {
  items: StorageDecisionRecordSummary[]
  total: number
  page: number
  page_size: number
  total_pages: number
}

export interface DecisionSnapshot {
  username: string
  user_id: number
  current_usage_bytes: number
  current_quota_bytes: number
  theoretical_quota_bytes: number
  usage_ratio: number
  growth_rate_bytes_per_hour?: number
  platform_total_bytes: number
  platform_used_bytes: number
  platform_available_bytes: number
  is_currently_expanded: boolean
  jobs_frozen: boolean
  shrink_stage?: string
  last_expand_at?: string
  recent_history?: Array<{
    recorded_at: string
    usage_bytes: number
  }>
}

export interface ConstraintEvaluation {
  policy_version: string
  adjusted: boolean
  blocked: boolean
  violations: string[]
  adjustments: string[]
}

export interface StorageDecisionRecordDetail extends StorageDecisionRecordSummary {
  current_shrink_stage?: string
  snapshot?: DecisionSnapshot
  raw_decision?: LLMDecisionResponse
  final_decision?: LLMDecisionResponse
  constraint_result?: ConstraintEvaluation
}

export interface ReplayRecord {
  job_id: string
  username: string
  stored_adjusted: boolean
  stored_blocked: boolean
  replay_adjusted: boolean
  replay_blocked: boolean
  stored_allow_expand: boolean
  replay_allow_expand: boolean
  stored_expand_bytes: number
  replay_expand_bytes: number
  stored_freeze: boolean
  replay_freeze: boolean
  evaluation: ConstraintEvaluation
}

export interface ReplaySummary {
  total_cases: number
  changed_cases: number
  blocked_cases: number
  clamped_cases: number
  freeze_escalations: number
  policy_version: string
  records?: ReplayRecord[]
}

export interface ReplayDecisionRequest {
  limit?: number
  max_expand_ratio?: number
  max_expand_bytes?: number
  min_platform_reserved_ratio?: number
  min_platform_reserved_bytes?: number
  expansion_cooldown_hours?: number
  force_freeze_when_over_quota?: boolean
}

export const apiAdminGetStorageDecisions = (
  page: number = 1,
  pageSize: number = 20,
  filters?: { user?: string; status?: string; source?: string }
): Promise<IResponse<StorageDecisionPage>> => {
  const params = new URLSearchParams({
    page: String(page),
    pageSize: String(pageSize),
  })
  if (filters?.user) params.set('user', filters.user)
  if (filters?.status) params.set('status', filters.status)
  if (filters?.source) params.set('source', filters.source)
  return apiGet<IResponse<StorageDecisionPage>>(`v1/admin/storage/decisions?${params.toString()}`)
}

export const apiAdminGetStorageDecision = (
  jobId: string
): Promise<IResponse<StorageDecisionRecordDetail>> =>
  apiGet<IResponse<StorageDecisionRecordDetail>>(`v1/admin/storage/decisions/${jobId}`)

export const apiAdminReplayStorageDecisions = (
  payload: ReplayDecisionRequest
): Promise<IResponse<ReplaySummary>> =>
  apiPost<IResponse<ReplaySummary>>('v1/admin/storage/decisions/replay', payload)

export const apiAdminRunAutoShrink = (): Promise<IResponse<{ message: string }>> =>
  apiPost<IResponse<{ message: string }>>('v1/admin/storage/auto-shrink')

export interface MetadataScanJob {
  id: number
  scanId: string
  workspaceType: 'user' | 'account' | 'public'
  workspaceName: string
  logicalPath: string
  snapshotName?: string
  materializedSnapshotName?: string
  scanRoot?: string
  triggerSource: string
  scanMode: 'full' | 'daily_refresh'
  baseScanId?: string
  diffMethod?: string
  status: 'pending' | 'running' | 'done' | 'error'
  entryCount: number
  fileCount: number
  directoryCount: number
  totalSizeBytes: number
  changedPathCount: number
  redundancyCount: number
  redundancyBytes: number
  errorMessage?: string
  startedAt?: string
  finishedAt?: string
  latencyMs: number
}

export interface MetadataWorkspaceOverview {
  workspace_type: 'user' | 'account' | 'public'
  workspace_name: string
  logical_path: string
  last_scan_id: string
  last_scan_status: 'pending' | 'running' | 'done' | 'error'
  last_scan_at?: string
  entry_count: number
  file_count: number
  directory_count: number
  redundancy_count: number
  redundancy_bytes: number
  top_directories: Array<{
    path: string
    name: string
    depth: number
    file_count: number
    directory_count: number
    total_size_bytes: number
    is_top_level: boolean
  }>
  largest_files: Array<{
    path: string
    name: string
    size_bytes: number
    modified_at?: string
  }>
}

export interface MetadataRedundancyHit {
  id: number
  workspaceType: 'user' | 'account' | 'public'
  workspaceName: string
  scanId: string
  targetType: 'file' | 'directory'
  targetPath: string
  publicPath: string
  matchKey: string
  evidence: string
  confidence: string
  verificationStatus: 'suspected' | 'verified'
  verificationMode?: string
  hashAlgorithm?: string
  targetHash?: string
  publicHash?: string
  estimatedBytes: number
  createdAt: string
  updatedAt: string
}

export interface MetadataCandidate {
  id: number
  workspaceType: 'user' | 'account' | 'public'
  workspaceName: string
  scanId: string
  candidateType: string
  targetPath: string
  publicPath: string
  evidence: string
  candidateScore: number
  status: 'suspected' | 'verified' | 'rejected'
  createdAt: string
  updatedAt: string
}

export interface MetadataCandidateFile {
  id: number
  workspaceType: 'user' | 'account' | 'public'
  workspaceName: string
  scanId: string
  candidatePath: string
  filePath: string
  fileName: string
  relativePath: string
  sizeBytes: number
  matchedPublicFile: string
  hashAlgorithm?: string
  fileHash?: string
  verificationStatus: 'suspected' | 'verified'
  createdAt: string
  updatedAt: string
}

export interface MetadataFolderCompareTiming {
  scan_ms: number
  pairing_ms: number
  header_ms: number
  sampled_hash_ms: number
  full_hash_ms: number
  total_ms: number
}

export interface MetadataFolderCompareFile {
  left_relative_path: string
  right_relative_path: string
  file_name: string
  size_bytes: number
  verification_mode: string
  same: boolean
  reason?: string
  header_matched?: boolean
  sampled_hash_match?: boolean
}

export interface MetadataFolderCompareResult {
  compare_type: 'model' | 'dataset' | 'auto'
  compare_mode: 'optimized' | 'full_hash'
  left_path: string
  right_path: string
  same: boolean
  left_key_file_count: number
  right_key_file_count: number
  exact_match_count: number
  fallback_match_count: number
  compared_file_count: number
  verified_file_count: number
  missing_left: string[]
  missing_right: string[]
  files: MetadataFolderCompareFile[]
  timing: MetadataFolderCompareTiming
}

export interface MetadataFolderCompareJob {
  job_id: string
  status: 'pending' | 'running' | 'done' | 'error'
  left_path: string
  right_path: string
  compare_type: 'model' | 'dataset' | 'auto'
  compare_mode: 'optimized' | 'full_hash'
  result?: MetadataFolderCompareResult
  error?: string
  started_at?: string
  finished_at?: string
}

export const apiAdminTriggerMetadataScan = (payload: {
  workspace_type: 'user' | 'account' | 'public'
  workspace_name?: string
}): Promise<IResponse<{ scan_id: string; workspace_type: string; workspace_name: string }>> =>
  apiPost<IResponse<{ scan_id: string; workspace_type: string; workspace_name: string }>>(
    'v1/admin/storage/index/scan',
    payload
  )

export const apiAdminGetMetadataScan = (scanId: string): Promise<IResponse<MetadataScanJob>> =>
  apiGet<IResponse<MetadataScanJob>>(`v1/admin/storage/index/scans/${scanId}`)

export const apiAdminGetMetadataWorkspaceOverview = (
  workspaceType: 'user' | 'account' | 'public',
  workspaceName: string
): Promise<IResponse<MetadataWorkspaceOverview>> =>
  apiGet<IResponse<MetadataWorkspaceOverview>>(
    `v1/admin/storage/index/workspaces/${workspaceType}/${workspaceName}/overview`
  )

export const apiAdminGetMetadataWorkspaceRedundancyHits = (
  workspaceType: 'user' | 'account' | 'public',
  workspaceName: string,
  page: number = 1,
  pageSize: number = 20
): Promise<
  IResponse<{
    items: MetadataRedundancyHit[]
    total: number
    page: number
    pageSize: number
    totalPages: number
  }>
> =>
  apiGet<
    IResponse<{
      items: MetadataRedundancyHit[]
      total: number
      page: number
      pageSize: number
      totalPages: number
    }>
  >(
    `v1/admin/storage/index/workspaces/${workspaceType}/${workspaceName}/redundancy-hits?page=${page}&pageSize=${pageSize}`
  )

export const apiAdminGetMetadataWorkspaceCandidates = (
  workspaceType: 'user' | 'account' | 'public',
  workspaceName: string,
  page: number = 1,
  pageSize: number = 20
): Promise<
  IResponse<{
    items: MetadataCandidate[]
    total: number
    page: number
    pageSize: number
    totalPages: number
  }>
> =>
  apiGet<
    IResponse<{
      items: MetadataCandidate[]
      total: number
      page: number
      pageSize: number
      totalPages: number
    }>
  >(
    `v1/admin/storage/index/workspaces/${workspaceType}/${workspaceName}/candidates?page=${page}&pageSize=${pageSize}`
  )

export const apiAdminGetMetadataWorkspaceCandidateFiles = (
  workspaceType: 'user' | 'account' | 'public',
  workspaceName: string,
  candidatePath: string,
  page: number = 1,
  pageSize: number = 200
): Promise<
  IResponse<{
    items: MetadataCandidateFile[]
    total: number
    page: number
    pageSize: number
    totalPages: number
  }>
> =>
  apiGet<
    IResponse<{
      items: MetadataCandidateFile[]
      total: number
      page: number
      pageSize: number
      totalPages: number
    }>
  >(
    `v1/admin/storage/index/workspaces/${workspaceType}/${workspaceName}/candidate-files?candidate_path=${encodeURIComponent(candidatePath)}&page=${page}&pageSize=${pageSize}`
  )

export const apiAdminCompareMetadataFolders = (payload: {
  left_path: string
  right_path: string
  compare_type?: 'auto' | 'model' | 'dataset'
  compare_mode?: 'optimized' | 'full_hash'
}): Promise<IResponse<{ job_id: string }>> =>
  apiPost<IResponse<{ job_id: string }>>('v1/admin/storage/index/compare-folders', payload, {
    timeout: 300000,
  })

export const apiAdminGetMetadataFolderCompareJob = (
  jobId: string
): Promise<IResponse<MetadataFolderCompareJob>> =>
  apiGet<IResponse<MetadataFolderCompareJob>>(`v1/admin/storage/index/compare-folders/${jobId}`, {
    timeout: 300000,
  })
