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
import { apiV1Delete, apiV1Get, apiV1Post, apiV1Put } from '@/services/client'
import { IResponse } from '@/services/types'

import { IUserInfo } from './vcjob'

export interface Content {
  approvalorderTypeID: number
  approvalorderReason: string
  approvalorderExtensionHours: number
}

export interface ApprovalOrder {
  id: number
  name: string
  type: string
  status: 'Pending' | 'Approved' | 'Rejected' | 'Canceled'
  content: Content
  reviewNotes: string
  creator: IUserInfo
  reviewer: IUserInfo
  createdAt: string
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
export const listApprovalOrders = () => {
  return apiV1Get<IResponse<ApprovalOrder[]>>('admin/approvalorder')
}
export const listMyApprovalOrder = () => {
  return apiV1Get<IResponse<ApprovalOrder[]>>('approvalorder')
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
export const listApprovalOrdersbyName = (name: string) => {
  return apiV1Get<IResponse<ApprovalOrder[]>>(`approvalorder/name/${name}`)
}
export const getApprovalOrder = (id: number) => {
  return apiV1Get<IResponse<ApprovalOrder>>(`approvalorder/${id}`)
}
export const adminGetApprovalOrder = (id: number) => {
  return apiV1Get<IResponse<ApprovalOrder>>(`admin/approvalorder/${id}`)
}
export const checkPendingApprovalOrder = () => {
  return apiV1Put<IResponse<string>>('admin/approvalorder/check')
}
