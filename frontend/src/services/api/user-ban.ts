/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { apiV1Get } from '@/services/client'
import { IResponse } from '@/services/types'

export type UserBanAction = 'ban' | 'extend' | 'unban'

export interface IUserBanRestrictions {
  platformAccess: boolean
  jobSubmission: boolean
  imageBuild: boolean
  modelDownload: boolean
  datasetDownload: boolean
}

export interface IUserBanSummary {
  banned: boolean
  permanentBanned: boolean
  bannedTimestamp?: string
  banRestrictions: IUserBanRestrictions
  reason: string
}

export interface IUserBanRecord {
  id: number
  createdAt: string
  action: UserBanAction
  permanentBanned: boolean
  bannedTimestamp?: string
  banRestrictions: IUserBanRestrictions
  reason: string
}

export interface IUserBanStatus extends IUserBanSummary {
  records: IUserBanRecord[]
}

export const USER_BAN_STATUS_REFETCH_INTERVAL = 60 * 1000

export const apiGetCurrentUserBanStatus = () => apiV1Get<IResponse<IUserBanSummary>>('users/ban')

export const apiGetUserBanStatus = (userName: string) =>
  apiV1Get<IResponse<IUserBanStatus>>(`users/${userName}/ban`)
