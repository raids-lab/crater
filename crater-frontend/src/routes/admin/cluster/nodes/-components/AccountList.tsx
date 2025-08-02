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
// i18n-processed-v1.1.0
import { useQuery } from '@tanstack/react-query'
import React from 'react'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { apiAdminAccountList } from '@/services/api/account'
import { IAccount } from '@/services/api/account'

interface AccountSelectProps {
  value: string
  onChange: (value: string) => void
}

const AccountSelect: React.FC<AccountSelectProps> = ({ value, onChange }) => {
  const { t } = useTranslation()
  const { data } = useQuery({
    queryKey: ['admin', 'accounts'],
    queryFn: apiAdminAccountList,
  })

  return (
    <div>
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger id="account-select">
          <SelectValue placeholder={t('accountSelect.placeholder')} />
        </SelectTrigger>
        <SelectContent>
          {data?.data
            .filter((account: IAccount) => account.name !== 'default')
            .map((account: IAccount) => (
              <SelectItem key={account.name} value={account.name.toString()}>
                {account.nickname}
              </SelectItem>
            ))}
        </SelectContent>
      </Select>
    </div>
  )
}

export default AccountSelect
