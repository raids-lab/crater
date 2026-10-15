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

export const createLoginFormSchema = (t: (key: string) => string) =>
  z.object({
    username: z
      .string()
      .trim()
      .toLowerCase()
      .min(1, { message: t('auth.usernameRequired') })
      .max(20, { message: t('auth.usernameMaxLength') })
      .regex(/^[a-z][a-z0-9-]*[a-z0-9]$/, { message: t('auth.usernameInvalid') }),
    password: z
      .string()
      .min(1, { message: t('auth.passwordRequired') })
      .max(20, { message: t('auth.passwordMaxLength') }),
    acceptPrivacy: z.boolean().refine((value) => value === true, {
      message: t('auth.privacyRequired'),
    }),
  })
