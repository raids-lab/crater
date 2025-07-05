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

import { IAccount } from '@/services/api/account'
import { convertQuotaToForm, quotaSchema } from '@/utils/quota'
import { z } from 'zod'

export const formSchema = z.object({
  id: z.number().optional(),
  name: z
    .string()
    .min(1, {
      message: '账户名称不能为空',
    })
    .max(16, {
      message: '账户名称最多16个字符',
    }),
  resources: quotaSchema,
  expiredAt: z.date().optional(),
  admins: z.array(z.string()).optional(),
})

export type AccountFormSchema = z.infer<typeof formSchema>

export const accountToForm = (resourceTypes: string[], account: IAccount): AccountFormSchema => {
  const formData: AccountFormSchema = {
    name: account.name,
    expiredAt: account.expiredAt ? new Date(account.expiredAt) : undefined,
    resources: convertQuotaToForm(account.quota, resourceTypes),
    admins: [],
  }

  return formData
}
