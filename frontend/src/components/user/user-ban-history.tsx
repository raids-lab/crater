import { useQuery } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'

import { UserBanStatusBadge } from '@/components/badge/user-ban-status-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import { UserBanDetails, UserBanRestrictionBadges } from '@/components/user/user-ban-details'

import {
  IUserBanRecord as IAdminUserBanRecord,
  apiAdminGetUserBanStatus,
  getUserBanOperatorDisplayName,
} from '@/services/api/admin/user'
import {
  IUserBanRecord,
  USER_BAN_STATUS_REFETCH_INTERVAL,
  apiGetUserBanStatus,
} from '@/services/api/user-ban'
import { markApiErrorHandled } from '@/services/client'

interface UserBanHistoryProps {
  username: string
  showOperator?: boolean
}

const isAdminUserBanRecord = (
  record: IUserBanRecord | IAdminUserBanRecord
): record is IAdminUserBanRecord => 'operatorName' in record && 'operatorNickname' in record

export function UserBanHistory({ username, showOperator = false }: UserBanHistoryProps) {
  const { t } = useTranslation()
  const adminQuery = useQuery({
    queryKey: ['admin', 'users', username, 'ban'],
    queryFn: () => apiAdminGetUserBanStatus(username).then((res) => res.data),
    enabled: Boolean(username && showOperator),
    refetchInterval: USER_BAN_STATUS_REFETCH_INTERVAL,
  })
  const visibleQuery = useQuery({
    queryKey: ['users', username, 'ban'],
    queryFn: async () => {
      try {
        return (await apiGetUserBanStatus(username)).data
      } catch (error) {
        markApiErrorHandled(error)
        throw error
      }
    },
    enabled: Boolean(username && !showOperator),
    refetchInterval: USER_BAN_STATUS_REFETCH_INTERVAL,
  })
  const query = showOperator ? adminQuery : visibleQuery

  if (query.isLoading) {
    return <Skeleton className="h-24 w-full" />
  }

  if (query.isError) {
    return (
      <div className="flex items-center justify-between gap-3 rounded-lg border px-4 py-3">
        <p className="text-destructive text-sm">{t('userBan.history.loadError')}</p>
        <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
          {t('common.refresh')}
        </Button>
      </div>
    )
  }

  const status = query.data
  const currentRecord = status?.records[0]
  const currentOperatorName =
    showOperator && currentRecord && isAdminUserBanRecord(currentRecord)
      ? getUserBanOperatorDisplayName(currentRecord)
      : undefined

  return (
    <div className="space-y-3">
      {/* 当前封禁状态卡：badge 独行，封禁详情换行展示避免挤成一行 */}
      <div className="rounded-lg border px-3 py-3">
        <div className="flex flex-wrap items-center gap-x-3 gap-y-1.5">
          <span className="text-sm font-medium">{t('userBan.history.currentStatus')}</span>
          <UserBanStatusBadge
            banned={status?.banned ?? false}
            permanentBanned={status?.permanentBanned}
            bannedTimestamp={status?.bannedTimestamp}
            banRestrictions={status?.banRestrictions}
            reason={status?.reason}
            adminUserName={showOperator ? username : undefined}
          />
        </div>
        {status?.banned && (
          <div className="mt-2.5 border-t pt-2.5">
            <UserBanDetails
              permanentBanned={status.permanentBanned}
              bannedTimestamp={status.bannedTimestamp}
              banRestrictions={status.banRestrictions}
              reason={status.reason}
              operatorName={currentOperatorName}
            />
          </div>
        )}
      </div>

      {/* 历史操作记录列表 */}
      <div className="divide-y overflow-hidden rounded-lg border">
        {(status?.records ?? []).map((record) => {
          const operatorName =
            showOperator && isAdminUserBanRecord(record)
              ? getUserBanOperatorDisplayName(record)
              : undefined
          const hasRestrictions = Object.values(record.banRestrictions).some(Boolean)
          const hasDetails = Boolean(
            record.reason || record.bannedTimestamp || hasRestrictions || operatorName
          )
          const isUnban = record.action === 'unban'

          return (
            <div
              key={record.id}
              className={`flex flex-wrap items-start gap-x-3 gap-y-2 px-3 py-3 transition-colors ${isUnban ? 'hover:bg-muted/25' : 'hover:bg-muted/40'}`}
            >
              <Badge
                variant={isUnban ? 'secondary' : 'destructive'}
                className="mt-0.5 whitespace-nowrap"
              >
                {t(`userBan.actions.${record.action}`)}
              </Badge>
              {hasDetails && (
                <div className="flex min-w-0 flex-1 basis-52 flex-col gap-1">
                  {/* 原因独占一行，超长时截断 */}
                  {record.reason && (
                    <span className="line-clamp-2 text-sm leading-snug font-medium break-words">
                      {record.reason}
                    </span>
                  )}
                  {/* 元信息（时间、限制标签）放在下方辅助行 */}
                  <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
                    {operatorName && (
                      <span className="text-muted-foreground text-xs">
                        {t('userBan.history.operator', { name: operatorName })}
                      </span>
                    )}
                    {record.bannedTimestamp && (
                      <span className="text-muted-foreground text-xs">
                        {t('userBan.history.resultUntil')}{' '}
                        {record.permanentBanned ? (
                          t('userBan.history.permanent')
                        ) : (
                          <TimeDistance date={record.bannedTimestamp} />
                        )}
                      </span>
                    )}
                    {hasRestrictions && (
                      <UserBanRestrictionBadges restrictions={record.banRestrictions} />
                    )}
                  </div>
                </div>
              )}
              <div className="text-muted-foreground ml-auto shrink-0 text-xs">
                <TimeDistance date={record.createdAt} />
              </div>
            </div>
          )
        })}
        {status?.records.length === 0 && (
          <div className="text-muted-foreground px-4 py-6 text-center text-sm">
            {t('userBan.history.empty')}
          </div>
        )}
      </div>
    </div>
  )
}
