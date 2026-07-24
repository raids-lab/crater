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
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { Skeleton } from '@/components/ui/skeleton'

import { PhaseBadge, PhaseBadgeData } from '@/components/badge/phase-badge'
import { UserBanDetails } from '@/components/user/user-ban-details'

import { apiAdminGetUserBanStatus, getUserBanOperatorDisplayName } from '@/services/api/admin/user'
import {
  IUserBanRestrictions,
  apiGetCurrentUserBanStatus,
  apiGetUserBanStatus,
} from '@/services/api/user-ban'
import { markApiErrorHandled } from '@/services/client'

const emptyRestrictions: IUserBanRestrictions = {
  platformAccess: false,
  jobSubmission: false,
  imageBuild: false,
  modelDownload: false,
  datasetDownload: false,
}

type UserBanPhase = 'normal' | 'banned' | 'permanent'

interface UserBanStatusBadgeProps {
  banned: boolean
  permanentBanned?: boolean
  bannedTimestamp?: string
  banRestrictions?: IUserBanRestrictions
  reason?: string
  adminUserName?: string
  visibleUserName?: string
  isCurrentUser?: boolean
}

export function UserBanStatusBadge({
  banned,
  permanentBanned = false,
  bannedTimestamp,
  banRestrictions = emptyRestrictions,
  reason,
  adminUserName,
  visibleUserName,
  isCurrentUser = false,
}: UserBanStatusBadgeProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const canFetchDetails = Boolean(adminUserName || visibleUserName || isCurrentUser)
  const adminQuery = useQuery({
    queryKey: ['admin', 'users', adminUserName, 'ban'],
    queryFn: () => apiAdminGetUserBanStatus(adminUserName ?? '').then((res) => res.data),
    enabled: Boolean(banned && open && adminUserName),
    staleTime: 30 * 1000,
  })
  const visibleUserQuery = useQuery({
    queryKey: ['users', visibleUserName, 'ban'],
    queryFn: async () => {
      try {
        return (await apiGetUserBanStatus(visibleUserName ?? '')).data
      } catch (error) {
        markApiErrorHandled(error)
        throw error
      }
    },
    enabled: Boolean(banned && open && !adminUserName && visibleUserName),
    staleTime: 30 * 1000,
  })
  const currentUserQuery = useQuery({
    queryKey: ['current-user', 'ban'],
    queryFn: () => apiGetCurrentUserBanStatus().then((res) => res.data),
    enabled: Boolean(banned && open && !adminUserName && !visibleUserName && isCurrentUser),
    staleTime: 30 * 1000,
  })
  const query = adminUserName ? adminQuery : visibleUserName ? visibleUserQuery : currentUserQuery

  const resolvedStatus = canFetchDetails ? query.data : undefined
  const resolvedPermanentBanned = resolvedStatus?.permanentBanned ?? permanentBanned
  const operatorRecord = adminUserName ? adminQuery.data?.records[0] : undefined
  const resolvedOperatorName = operatorRecord
    ? getUserBanOperatorDisplayName(operatorRecord)
    : undefined
  const phase: UserBanPhase = !banned ? 'normal' : resolvedPermanentBanned ? 'permanent' : 'banned'
  const getPhaseLabel = (value: UserBanPhase): PhaseBadgeData => {
    switch (value) {
      case 'normal':
        return {
          label: t('userBan.status.normal'),
          color: 'text-highlight-emerald bg-highlight-emerald/20',
          description: t('userBan.status.normalDescription'),
        }
      case 'permanent':
        return {
          label: t('userBan.status.permanent'),
          color: 'text-highlight-red bg-highlight-red/20',
          description: t('userBan.status.permanentDescription'),
        }
      default:
        return {
          label: t('userBan.status.banned'),
          color: 'text-highlight-red bg-highlight-red/20',
          description: t('userBan.status.bannedDescription'),
        }
    }
  }

  const canShowDetails =
    banned &&
    Boolean(adminUserName || visibleUserName || isCurrentUser || bannedTimestamp || reason)
  if (!canShowDetails) {
    return <PhaseBadge phase={phase} getPhaseLabel={getPhaseLabel} />
  }

  return (
    <HoverCard open={open} onOpenChange={setOpen} openDelay={100} closeDelay={100}>
      <HoverCardTrigger asChild>
        <span className="inline-flex cursor-help">
          <PhaseBadge phase={phase} getPhaseLabel={getPhaseLabel} disableDefaultTooltip />
        </span>
      </HoverCardTrigger>
      <HoverCardContent side="top" align="start" className="w-[min(20rem,calc(100vw-2rem))] p-3">
        {query.isLoading && canFetchDetails ? (
          <Skeleton className="h-16 w-full" />
        ) : (
          <UserBanDetails
            permanentBanned={resolvedPermanentBanned}
            bannedTimestamp={resolvedStatus?.bannedTimestamp ?? bannedTimestamp}
            banRestrictions={resolvedStatus?.banRestrictions ?? banRestrictions}
            reason={resolvedStatus?.reason ?? reason}
            operatorName={resolvedOperatorName}
            layout="stacked"
          />
        )}
      </HoverCardContent>
    </HoverCard>
  )
}
