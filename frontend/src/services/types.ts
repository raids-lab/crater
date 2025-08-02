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
import { ErrorCode } from './error_code'

// https://codevoweb.com/react-query-context-api-axios-interceptors-jwt-auth/

export interface IResponse<T> {
  data: T
  code: ErrorCode
  msg: string
}

export type IErrorResponse = IResponse<never>

/**
 * Refresh
 */
export interface IRefresh {
  refreshToken: string
}

export interface IRefreshResponse {
  accessToken: string
  refreshToken: string
}
