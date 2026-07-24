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
import { useQuery } from '@tanstack/react-query'
import { CircleAlertIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'

import { UserBanDetails } from '@/components/user/user-ban-details'

import {
  USER_BAN_STATUS_REFETCH_INTERVAL,
  apiGetCurrentUserBanStatus,
} from '@/services/api/user-ban'

import { cn } from '@/lib/utils'

export function UserBanAlert({ className }: { className?: string }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['current-user', 'ban'],
    queryFn: () => apiGetCurrentUserBanStatus().then((res) => res.data),
    refetchInterval: USER_BAN_STATUS_REFETCH_INTERVAL,
  })

  const status = query.data
  if (!status?.banned) {
    return null
  }

  return (
    <Alert
      variant="destructive"
      className={cn('border-highlight-red/40 bg-highlight-red/5', className)}
    >
      <CircleAlertIcon />
      <AlertTitle>{t('userBan.banner.title')}</AlertTitle>
      <AlertDescription className="mt-1">
        <UserBanDetails
          permanentBanned={status.permanentBanned}
          bannedTimestamp={status.bannedTimestamp}
          banRestrictions={status.banRestrictions}
          reason={status.reason}
          className="w-full"
        />
      </AlertDescription>
    </Alert>
  )
}
