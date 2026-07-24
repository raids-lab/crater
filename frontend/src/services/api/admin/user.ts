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
import { apiV1Delete, apiV1Get, apiV1Put } from '@/services/client'
import { IResponse } from '@/services/types'

import { ProjectStatus } from '../account'
import { Role } from '../auth'
import {
  IUserBanRestrictions,
  IUserBanRecord as IVisibleUserBanRecord,
  IUserBanStatus as IVisibleUserBanStatus,
} from '../user-ban'

export type { IUserBanRestrictions, IUserBanSummary, UserBanAction } from '../user-ban'

export interface IUserAttributes {
  id: number
  name: string
  nickname: string
  email?: string
  teacher?: string
  group?: string
  expiredAt?: string
  phone?: string
  avatar?: string
  uid?: string
  gid?: string
}

export interface IUser {
  id: number
  name: string
  role: Role
  status: ProjectStatus
  extraBalance?: number
  banned: boolean
  permanentBanned: boolean
  bannedTimestamp?: string
  banRestrictions: IUserBanRestrictions
  attributes: IUserAttributes
}

export interface IUserBanRecord extends IVisibleUserBanRecord {
  operatorId: number
  operatorName: string
  operatorNickname: string
}

export interface IUserBanStatus extends Omit<IVisibleUserBanStatus, 'records'> {
  records: IUserBanRecord[]
}

export const getUserBanOperatorDisplayName = (
  record: Pick<IUserBanRecord, 'operatorName' | 'operatorNickname'>
) => {
  const nickname = record.operatorNickname?.trim()
  return nickname && nickname !== record.operatorName
    ? `${nickname} (@${record.operatorName})`
    : record.operatorName
}

export interface IUpdateUserBanReq {
  banned: boolean
  isPermanent: boolean
  days: number
  hours: number
  minutes: number
  banRestrictions: IUserBanRestrictions
  reason: string
}

export const apiAdminUserList = () => apiV1Get<IResponse<IUser[]>>('admin/users')

export const apiAdminUserDelete = (userName: string) =>
  apiV1Delete<IResponse<string>>(`admin/users/${userName}`)

export const apiAdminUpdateUserAttributes = (username: string, data: IUserAttributes) =>
  apiV1Put<IResponse<string>>(`admin/users/${username}/attributes`, data)

export const apiAdminUserUpdateRole = (userName: string, role: Role) =>
  apiV1Put<IResponse<string>>(`admin/users/${userName}/role`, {
    role,
  })

export const apiAdminGetUserBanStatus = (userName: string) =>
  apiV1Get<IResponse<IUserBanStatus>>(`admin/users/${userName}/ban`)

export const apiAdminUpdateUserBanStatus = (userName: string, data: IUpdateUserBanReq) =>
  apiV1Put<IResponse<IUserBanStatus>>(`admin/users/${userName}/ban`, data)
