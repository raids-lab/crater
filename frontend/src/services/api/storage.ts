import { apiGet, apiPost, apiPut } from '@/services/client'
import { IResponse } from '@/services/types'

export interface UserSpace {
  user: string
  size: number
  quota: number
  unit: string
  formatted: string
  updated_at?: string | null
  quota_formatted: string
}

export interface PagedUserSpaces {
  items: UserSpace[]
  total: number
  page: number
  pageSize: number
  totalPages: number
}

export interface StorageCapabilities {
  backend: string
  configured: boolean
  quota_enabled: boolean
  pvc_name: string
  pvc_namespace?: string
  pv_name?: string
  csi_driver?: string
  quota_provider: 'auto' | 'storageServer' | 'toolbox' | 'disabled'
  storage_server_available: boolean
  toolbox_available: boolean
  usage_readable: boolean
  quota_readable: boolean
  quota_writable: boolean
  reasons?: string[]
}

export const apiAdminGetStorageCapabilities = (): Promise<IResponse<StorageCapabilities>> =>
  apiGet<IResponse<StorageCapabilities>>('v1/admin/storage/capabilities')

export const apiAdminGetUserSpaces = (
  page: number = 1,
  pageSize: number = 10
): Promise<IResponse<PagedUserSpaces>> =>
  apiGet<IResponse<PagedUserSpaces>>(
    `v1/admin/storage/user-spaces?page=${page}&pageSize=${pageSize}`
  )

export interface RefreshStorageUsageResponse {
  updated: number
  failed: number
  refreshed_at: string
}

export const apiAdminRefreshUserSpaceUsage = (): Promise<IResponse<RefreshStorageUsageResponse>> =>
  apiPost<IResponse<RefreshStorageUsageResponse>>(
    'v1/admin/storage/user-spaces/refresh',
    undefined,
    { timeout: false }
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
}

export const apiAdminSetUserSpaceQuota = (
  user: string,
  quota: number
): Promise<IResponse<SetQuotaResponse>> =>
  apiPut<IResponse<SetQuotaResponse>>(
    `v1/admin/storage/user-spaces/${encodeURIComponent(user)}/quota`,
    { quota }
  )
