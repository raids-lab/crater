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
import { apiV1Get } from '@/services/client'

import { IResponse } from '../types'
import { ProjectStatus } from './account'
import { Role } from './auth'

// type UserDetailResp struct {
// 	ID        uint         `json:"id"`        // 用户ID
// 	Name      string       `json:"name"`      // 用户名称
// 	Nickname  string       `json:"nickname"`  // 用户昵称
// 	Role      model.Role   `json:"role"`      // 用户角色
// 	Status    model.Status `json:"status"`    // 用户状态
// 	CreatedAt time.Time    `json:"createdAt"` // 创建时间
// 	Teacher   *string      `json:"teacher"`   // 导师
// 	Group     *string      `json:"group"`     // 课题组
// 	Avatar    *string      `json:"avatar"`    // 头像
// }
export interface IUser {
  id: number // 用户ID
  name: string // 用户名称
  nickname: string // 用户昵称
  role: Role // 用户角色
  status: ProjectStatus // 用户状态
  createdAt: string // 创建时间
  teacher?: string // 导师
  group?: string // 课题组
  avatar?: string // 头像
}
export interface BaseUserInfo {
  name: string
  nickname: string
  space: string
}

export const apiGetUser = (userName: string) => apiV1Get<IResponse<IUser>>(`users/${userName}`)

export interface EmailVerifiedResponse {
  verified: boolean // 是否已验证
  lastEmailVerifiedAt?: string // 上次验证时间
}

export const apiUserEmailVerified = () =>
  apiV1Get<IResponse<EmailVerifiedResponse>>(`users/email/verified`)

export const apiGetBaseUserInfo = () => apiV1Get<IResponse<BaseUserInfo[]>>(`/admin/users/baseinfo`)
