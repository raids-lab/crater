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

import instance, { VERSION } from '@/services/axios'
import { IResponse } from '@/services/types'
import { Role } from '../auth'
import { ProjectStatus } from '../account'

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
  attributes: IUserAttributes
}

export const apiAdminUserList = () => instance.get<IResponse<IUser[]>>(`${VERSION}/admin/users`)

export const apiAdminUserDelete = (userName: string) =>
  instance.delete<IResponse<string>>(`${VERSION}/admin/users/${userName}`)

export const apiAdminUpdateUserAttributes = (username: string, data: IUserAttributes) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${username}/attributes`, data)

export const apiAdminUserUpdateRole = (userName: string, role: Role) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${userName}/role`, {
    role,
  })
