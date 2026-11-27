/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import {
  type RemoteTableParams,
  buildRemoteSearchParams,
} from '@/components/query-table/remote-state'

import { fetchAllPages } from '@/services/api/pagination'
import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'
import { IResponse } from '@/services/types'
import type { IPage } from '@/services/types'

import { IUserInfo, type JobLockDuration } from './vcjob'

export interface ApprovedJobLock {
  isPermanent: boolean
  durationMinutes?: number
  lockedUntil: string
}

export interface Content {
  approvalorderTypeID: number
  approvalorderReason: string
  approvalorderExtensionHours: number
  approvedLock?: ApprovedJobLock
}

export interface ApprovalOrder {
  id: number
  name: string
  type: string
  status: 'Pending' | 'Approved' | 'Rejected' | 'Canceled'
  content: Content
  reviewNotes: string
  creatorID: number
  creator: IUserInfo
  reviewer: IUserInfo
  createdAt: string
}

export interface ApprovalOrderSummary {
  totalPending: number
  pendingJobDelay: number
  pendingDataset: number
  updatedAt: string
}
export interface ApprovalOrderReq {
  name: string
  type: string
  status: string
  approvalorderTypeID: number
  approvalorderReason: string
  approvalorderExtensionHours: number
  reviewerID?: number
  reviewNotes?: string
}

export interface ApprovalOrderReviewReq {
  status: 'Approved' | 'Rejected' | 'Canceled'
  reviewNotes?: string
}

export const listApprovalOrders = () => {
  return fetchAllPages((page) =>
    apiGetApprovalOrderPage({ page, page_size: 200, sort: '-createdAt', filters: {} }, true)
  )
}
export const listMyApprovalOrder = () => {
  return fetchAllPages((page) =>
    apiGetApprovalOrderPage({ page, page_size: 200, sort: '-createdAt', filters: {} })
  )
}
export const apiGetApprovalOrderPage = (
  params: RemoteTableParams,
  admin = false,
  signal?: AbortSignal
) => {
  return apiV1Get<IResponse<IPage<ApprovalOrder>>>(
    admin ? 'admin/approvalorder' : 'approvalorder',
    {
      searchParams: buildRemoteSearchParams(params),
      signal,
    }
  )
}
export const apiGetApprovalOrderSummary = (signal?: AbortSignal) => {
  return apiV1Get<IResponse<ApprovalOrderSummary>>('admin/approvalorder/summary', { signal })
}
export const deleteApprovalOrder = (id: number) => {
  return apiV1Delete<IResponse<ApprovalOrder>>(`approvalorder/${id}`)
}
export const createApprovalOrder = (data: ApprovalOrderReq) => {
  return apiV1Post<IResponse<ApprovalOrder>>('approvalorder', data)
}
export const updateApprovalOrder = (id: number, data: ApprovalOrderReq) => {
  return apiV1Put<IResponse<ApprovalOrder>>(`approvalorder/${id}`, data)
}
export const reviewApprovalOrder = (id: number, data: ApprovalOrderReviewReq) => {
  return apiV1Put<IResponse<string>>(`admin/approvalorder/${id}/review`, data)
}
export const adminApproveJobOrder = (id: number, lock: JobLockDuration) =>
  apiV1Put<IResponse<{ message: string; lockMessage: string; approvedLock: ApprovedJobLock }>>(
    `admin/approvalorder/${id}/approve-and-lock`,
    { lock }
  )
export const listJobApprovalOrders = (jobName: string) => {
  return apiV1Get<IResponse<ApprovalOrder[]>>(`approvalorder/name/${encodeURIComponent(jobName)}`, {
    searchParams: { type: 'job' },
  })
}
export const getApprovalOrder = (id: number) => {
  return apiV1Get<IResponse<ApprovalOrder>>(`approvalorder/${id}`)
}
export const adminGetApprovalOrder = (id: number) => {
  return apiV1Get<IResponse<ApprovalOrder>>(`admin/approvalorder/${id}`)
}
