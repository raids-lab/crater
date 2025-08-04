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

import { IResponse } from '../types'
import { IUserAttributes } from './admin/user'
import { Role } from './auth'
import { QuotaResp } from './context'

export enum ProjectStatus {
  Pending = 1,
  Active,
  Inactive,
}

export interface ProjectBasic {
  id: number
  name: string
  role: Role
  accessmode: Access
}

export interface ICreateOrUpdateAccount {
  name: string
  quota: IQuota
  expiredAt: Date | undefined
  withoutVolcano: boolean
  admins?: number[]
}

export interface ICreateProjectResponse {
  id: number
}

export interface IQuota {
  guaranteed?: Record<string, string>
  deserved?: Record<string, string>
  capability?: Record<string, string>
}

export interface IAccount {
  id: number
  name: string
  nickname: string
  space: string
  quota: IQuota
  expiredAt?: string
}

export const apiAdminAccountList = () => apiV1Get<IResponse<IAccount[]>>(`/admin/accounts`)

export const apiAccountCreate = (account: ICreateOrUpdateAccount) =>
  apiV1Post<IResponse<ICreateProjectResponse>>('admin/accounts', account)

export const apiAccountUpdate = (id: number, account: ICreateOrUpdateAccount) =>
  apiV1Put<IResponse<ICreateProjectResponse>>(`admin/accounts/${id}`, account)

export interface IDeleteProjectResp {
  name: string
}
export const apiProjectDelete = (id: number) =>
  apiV1Delete<IResponse<IDeleteProjectResp>>(`admin/accounts/${id}`)

export interface IUserInAccountCreate {
  id: number
  name: string
  role: string
  accessmode: string
}

export type IUserInAccount = IUserInAccountCreate & {
  userInfo: IUserAttributes
  quota: IQuota
}

export enum Access {
  NA = 1,
  RO,
  RW,
  AO,
}

export const apiAddUser = async (pid: number, user: IUserInAccountCreate) =>
  apiV1Post<IResponse<IUserInAccount>>('admin/accounts/add/' + pid + '/' + user.id, {
    role: user.role,
    accessmode: user.accessmode,
  })

export const apiUpdateUser = async (pid: number, user: IUserInAccountCreate) =>
  apiV1Post<IResponse<IUserInAccount>>('admin/accounts/update/' + pid + '/' + user.id, {
    role: user.role,
    accessmode: user.accessmode,
  })

export const apiRemoveUser = async (pid: number, user: IUserInAccountCreate) =>
  apiV1Delete<IResponse<IUserInAccount>>('admin/accounts/' + pid + '/' + user.id)

export const apiUserInProjectList = (pid: number) =>
  apiV1Get<IResponse<IUserInAccount[]>>('/admin/accounts/userIn/' + pid)

export const apiUserOutOfProjectList = (pid: number) =>
  apiV1Get<IResponse<IUserInAccount[]>>('/admin/accounts/userOutOf/' + pid)

export const apiAccountQuotaGet = (pid: number) => {
  return apiV1Get<IResponse<QuotaResp>>(`admin/accounts/${pid}/quota`)
}

export const apiAccountGet = (pid: number) => {
  return apiV1Get<IResponse<IAccount>>(`admin/accounts/${pid}`)
}
export const apiAccountGetByName = (name: string) => {
  return apiV1Get<IResponse<IAccount>>(`accounts/${name}`)
}
