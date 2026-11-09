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
import { z } from 'zod'

export const loginFormSchema = z.object({
  username: z
    .string()
    .trim()
    .toLowerCase()
    .min(1, { message: '用户名不能为空' })
    .max(20, { message: '用户名最多 20 个字符' })
    .regex(/^[a-z][a-z0-9-]*[a-z0-9]$/, {
      message: '用户名须以小写字母开头，以小写字母或数字结尾，只能包含小写字母、数字和中划线',
    }),
  password: z
    .string()
    .min(1, { message: '密码不能为空' })
    .max(20, { message: '密码最多 20 个字符' }),
  acceptPrivacy: z.boolean().refine((value) => value === true, {
    message: '请先阅读并同意隐私政策方可登录',
  }),
})
