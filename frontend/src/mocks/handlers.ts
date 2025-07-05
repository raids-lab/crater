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

// 1. Import the "HttpResponse" class from the library.
import { AccessMode, IAuthResponse, ILogin, Role } from '@/services/api/auth'
import { IResponse } from '@/services/types'
import { logger } from '@/utils/loglevel'
import { http, HttpResponse } from 'msw'

const baseURL = import.meta.env.VITE_API_BASE_URL

export const handlers = [
  http.post<never, ILogin, IResponse<IAuthResponse>>(baseURL + `login`, async ({ request }) => {
    const { username, password } = await request.json()
    logger.info(`login with username: ${username}, password: ${password}`)
    const mockUser = {
      username: 'ganhao',
      password: '',
      accessToken:
        'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1aSI6NiwicWkiOjIzLCJ1biI6ImdhbmhhbyIsInFuIjoicS0yMyIsInJxIjozLCJycCI6MywiYW0iOjMsInBhIjoxLCJleHAiOjE3MTg4ODg3MDl9.-v3oQ_X9L1cwW96eRkk-SRPUGcJsRgHApM011BRTlWs',
      refreshToken: '',
      role: Role.Admin,
    }
    // 成功登录，返回accessToken、refreshToken和role
    return HttpResponse.json(
      {
        data: {
          accessToken: mockUser.accessToken,
          refreshToken: mockUser.refreshToken,
          context: {
            queue: '',
            roleQueue: Role.Guest,
            rolePlatform: Role.Guest,
            accessQueue: AccessMode.NotAllowed,
            accessPublic: AccessMode.NotAllowed,
            space: '',
          },
          user: {
            id: 0,
            name: '',
            nickname: '',
          },
        },
        code: 0,
        msg: '',
      },
      { status: 200 }
    )
  }),
]
