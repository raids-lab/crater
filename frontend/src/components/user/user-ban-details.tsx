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
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import { IUserBanRestrictions } from '@/services/api/user-ban'

import { cn } from '@/lib/utils'

const restrictionKeys: Array<keyof IUserBanRestrictions> = [
  'platformAccess',
  'jobSubmission',
  'imageBuild',
  'modelDownload',
  'datasetDownload',
]

interface UserBanRestrictionBadgesProps {
  restrictions: IUserBanRestrictions
  className?: string
}

export function UserBanRestrictionBadges({
  restrictions,
  className,
}: UserBanRestrictionBadgesProps) {
  const { t } = useTranslation()
  const activeRestrictions = restrictionKeys.filter((key) => restrictions[key])

  if (activeRestrictions.length === 0) {
    return <span className="text-muted-foreground text-sm">{t('userBan.details.none')}</span>
  }

  return (
    <div className={cn('flex flex-wrap gap-1.5', className)}>
      {activeRestrictions.map((key) => (
        <Badge key={key} variant="outline" className="font-normal">
          {t(`userBan.restrictions.${key}`)}
        </Badge>
      ))}
    </div>
  )
}

interface UserBanDetailsProps {
  permanentBanned?: boolean
  bannedTimestamp?: string
  banRestrictions: IUserBanRestrictions
  reason?: string
  operatorName?: string
  layout?: 'inline' | 'stacked'
  className?: string
}

export function UserBanDetails({
  permanentBanned = false,
  bannedTimestamp,
  banRestrictions,
  reason,
  operatorName,
  layout = 'inline',
  className,
}: UserBanDetailsProps) {
  const { t } = useTranslation()
  const expiration = permanentBanned
    ? t('userBan.status.permanent')
    : bannedTimestamp
      ? new Date(bannedTimestamp).toLocaleString()
      : t('userBan.details.unknownExpiration')

  if (layout === 'stacked') {
    return (
      <dl
        className={cn(
          'grid grid-cols-[max-content_minmax(0,1fr)] items-start gap-x-3 gap-y-2 text-sm',
          className
        )}
      >
        <dt className="text-muted-foreground text-xs leading-5">
          {t('userBan.details.expiration')}
        </dt>
        <dd className="leading-5 font-medium">{expiration}</dd>
        <dt className="text-muted-foreground text-xs leading-5">{t('userBan.details.reason')}</dt>
        <dd className="leading-5 break-words">{reason || t('userBan.details.noReason')}</dd>
        <dt className="text-muted-foreground text-xs leading-5">
          {t('userBan.details.restrictions')}
        </dt>
        <dd>
          <UserBanRestrictionBadges restrictions={banRestrictions} />
        </dd>
        {operatorName && (
          <>
            <dt className="text-muted-foreground text-xs leading-5">
              {t('userBan.details.operator')}
            </dt>
            <dd className="leading-5 break-words">{operatorName}</dd>
          </>
        )}
      </dl>
    )
  }

  return (
    <dl className={cn('flex flex-wrap items-center gap-x-5 gap-y-2 text-sm', className)}>
      <div className="flex min-w-0 items-baseline gap-1.5">
        <dt className="text-muted-foreground shrink-0 text-xs">
          {t('userBan.details.expiration')}
        </dt>
        <dd className="font-medium whitespace-nowrap">{expiration}</dd>
      </div>
      <div className="flex min-w-0 items-baseline gap-1.5">
        <dt className="text-muted-foreground shrink-0 text-xs">{t('userBan.details.reason')}</dt>
        <dd className="break-words">{reason || t('userBan.details.noReason')}</dd>
      </div>
      <div className="flex min-w-0 items-center gap-1.5">
        <dt className="text-muted-foreground shrink-0 text-xs">
          {t('userBan.details.restrictions')}
        </dt>
        <dd>
          <UserBanRestrictionBadges restrictions={banRestrictions} />
        </dd>
      </div>
      {operatorName && (
        <div className="flex min-w-0 items-baseline gap-1.5">
          <dt className="text-muted-foreground shrink-0 text-xs">
            {t('userBan.details.operator')}
          </dt>
          <dd className="break-all">{operatorName}</dd>
        </div>
      )}
    </dl>
  )
}
