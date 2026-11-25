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

import { usernameSchema } from '@/services/auth-username'

export const loginFormSchema = z.object({
  username: z.string().trim().toLowerCase().pipe(usernameSchema),
  // LDAP password policies are managed by the directory, not local signup rules.
  password: z.string().min(1, { message: '密码不能为空' }),
  acceptPrivacy: z.boolean().refine((value) => value === true, {
    message: '请先阅读并同意隐私政策方可登录',
  }),
})
