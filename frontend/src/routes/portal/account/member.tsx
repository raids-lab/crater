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
import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'
import { useAtomValue } from 'jotai'
import { useMemo } from 'react'

import { Skeleton } from '@/components/ui/skeleton'

import { AccountMemberTable } from '@/components/account/account-member-table'
import PageTitle from '@/components/layout/page-title'

import { apiAccountGetByName } from '@/services/api/account'
import { Role } from '@/services/api/auth'

import { atomUserContext } from '@/utils/store'

export const Route = createFileRoute('/portal/account/member')({
  component: RouteComponent,
  loader: () => {
    return {
      crumb: t('navigation.memberManagement'),
    }
  },
})

function RouteComponent() {
  const accountContext = useAtomValue(atomUserContext)
  const accountName = accountContext?.queue

  // Get account info by name to get account ID
  // Note: hooks must be called before conditional returns to ensure consistent hook call order on each render
  const {
    data: accountInfo,
    isLoading,
    error,
  } = useQuery({
    queryKey: ['account', accountName],
    queryFn: () => apiAccountGetByName(accountName!),
    select: (res) => res.data,
    enabled: !!accountName && accountName !== 'default',
    retry: false, // Disable auto-retry to show errors immediately
  })

  const accountId = useMemo(() => accountInfo?.id, [accountInfo])

  // Default account does not support member management
  if (accountName === 'default') {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="text-center">
          <h1 className="text-muted-foreground text-2xl font-bold">
            {t('navigation.memberManagement')}
          </h1>
          <p className="text-muted-foreground mt-2">
            {t('accountDetail.defaultAccount.notSupported')}
          </p>
        </div>
      </div>
    )
  }

  // Loading state
  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-10 w-64" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  // Error state or account does not exist
  if (error || !accountInfo || !accountId) {
    return (
      <div className="flex h-96 items-center justify-center">
        <div className="text-center">
          <h1 className="text-2xl font-bold text-red-500">{t('accountDetail.error.title')}</h1>
          <p className="text-muted-foreground mt-2">{t('accountDetail.error.description')}</p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-4">
      <PageTitle
        title={t('navigation.memberManagement')}
        description={t('accountDetail.tabs.users')}
      />
      <AccountMemberTable
        accountId={accountId}
        editable={accountContext?.roleQueue === Role.Admin}
        storageKey="portal_account_members"
      />
    </div>
  )
}
