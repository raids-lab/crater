// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormControl, FormField, FormItem, FormLabel } from '@/components/ui/form'
import { Switch } from '@/components/ui/switch'

import { apiGetUserJobPermissions } from '@/services/api/context'

import { atomUserInfo } from '@/utils/store'

export function JobSysAdminFormField<T extends FieldValues>({
  form,
  name,
}: {
  form: UseFormReturn<T>
  name: FieldPath<T>
}) {
  const { t } = useTranslation()
  const user = useAtomValue(atomUserInfo)
  const { data } = useQuery({
    queryKey: ['context', 'job-permissions', user?.name],
    queryFn: apiGetUserJobPermissions,
    enabled: Boolean(user?.name),
  })

  if (data?.data.containerSysAdminEnabled !== true) return null

  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem className="flex flex-row items-center justify-between space-y-0 space-x-0">
          <FormLabel className="font-normal">{t('jobs.new.enableSysAdmin')}</FormLabel>
          <FormControl>
            <Switch checked={field.value === true} onCheckedChange={field.onChange} />
          </FormControl>
        </FormItem>
      )}
    />
  )
}
