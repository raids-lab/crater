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
import { z } from 'zod'

import { IQuota } from '@/services/api/account'

import { convertKResourceToResource, convertResourceToKResource } from './resource'

type TranslationFn = (key: string) => string

const createNonNegativeNumberSchema = (t: TranslationFn) =>
  z
    .number()
    .finite({
      message: t('accountForm.validation.quotaFinite'),
    })
    .min(0, {
      message: t('accountForm.validation.quotaMin'),
    })
    .nullish()

export const createQuotaSchema = (t: TranslationFn) => {
  const nonNegativeNumberSchema = createNonNegativeNumberSchema(t)

  return z.array(
    z.object({
      name: z.string().min(1, {
        message: t('accountForm.validation.resourceNameRequired'),
      }),
      guaranteed: nonNegativeNumberSchema,
      deserved: nonNegativeNumberSchema,
      capability: nonNegativeNumberSchema,
      queueLimit: nonNegativeNumberSchema,
    })
  )
}

export type QuotaSchema = z.infer<ReturnType<typeof createQuotaSchema>>

export const convertQuotaToForm = (
  quota: IQuota,
  resourceTypes?: string[],
  queueQuota?: Record<string, string>
): QuotaSchema => {
  return (
    resourceTypes?.map((name) => ({
      name,
      guaranteed: convertKResourceToResource(name, quota.guaranteed?.[name]),
      deserved: convertKResourceToResource(name, quota.deserved?.[name]),
      capability: convertKResourceToResource(name, quota.capability?.[name]),
      queueLimit: convertKResourceToResource(name, queueQuota?.[name]),
    })) ?? []
  )
}

export const convertFormToQuota = (form: QuotaSchema): IQuota => {
  const quota: IQuota = {
    guaranteed: {},
    deserved: {},
    capability: {},
  }

  form.forEach((resource) => {
    if (
      resource.guaranteed !== undefined &&
      resource.guaranteed !== null &&
      !isNaN(resource.guaranteed)
    ) {
      quota.guaranteed![resource.name] = convertResourceToKResource(
        resource.name,
        resource.guaranteed
      )
    }
    if (
      resource.deserved !== undefined &&
      resource.deserved !== null &&
      !isNaN(resource.deserved)
    ) {
      quota.deserved![resource.name] = convertResourceToKResource(resource.name, resource.deserved)
    }
    if (
      resource.capability !== undefined &&
      resource.capability !== null &&
      !isNaN(resource.capability)
    ) {
      quota.capability![resource.name] = convertResourceToKResource(
        resource.name,
        resource.capability
      )
    }
  })

  return quota
}

export const convertFormToQueueQuota = (form: QuotaSchema): Record<string, string> => {
  const quota: Record<string, string> = {}

  form.forEach((resource) => {
    if (
      resource.queueLimit !== undefined &&
      resource.queueLimit !== null &&
      !isNaN(resource.queueLimit)
    ) {
      quota[resource.name] = convertResourceToKResource(resource.name, resource.queueLimit)
    }
  })

  return quota
}
