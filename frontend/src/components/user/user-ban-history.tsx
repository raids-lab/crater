import { useQuery } from '@tanstack/react-query'
import { ShieldBanIcon, ShieldCheckIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'

import { TimeDistance } from '@/components/custom/time-distance'

import { apiAdminGetUserBanStatus } from '@/services/api/admin/user'

export function UserBanHistory({ username }: { username: string }) {
  const { t } = useTranslation()
  const query = useQuery({
    queryKey: ['admin', 'users', username, 'ban'],
    queryFn: () => apiAdminGetUserBanStatus(username).then((res) => res.data),
    enabled: Boolean(username),
  })

  if (query.isLoading) {
    return <Skeleton className="h-40 w-full" />
  }

  if (query.isError) {
    return (
      <Card>
        <CardContent className="flex flex-col items-center gap-3 p-6 text-center">
          <p className="text-destructive text-sm">{t('userBan.history.loadError')}</p>
          <Button variant="outline" size="sm" onClick={() => void query.refetch()}>
            {t('common.refresh')}
          </Button>
        </CardContent>
      </Card>
    )
  }

  const status = query.data
  return (
    <div className="space-y-4">
      <Card>
        <CardHeader className="flex flex-row items-start justify-between gap-4 space-y-0">
          <div className="min-w-0 space-y-1">
            <CardTitle className="text-base sm:text-lg">
              {t('userBan.history.currentStatus')}
            </CardTitle>
            <CardDescription>{t('userBan.history.currentStatusDesc')}</CardDescription>
          </div>
          {status?.banned ? (
            <Badge variant="destructive" className="shrink-0 whitespace-nowrap">
              <ShieldBanIcon className="mr-1 size-3" />
              {t('userBan.status.banned')}
            </Badge>
          ) : (
            <Badge variant="secondary" className="shrink-0 whitespace-nowrap">
              <ShieldCheckIcon className="mr-1 size-3" />
              {t('userBan.status.normal')}
            </Badge>
          )}
        </CardHeader>
        {status?.bannedAt && (
          <CardContent className="text-muted-foreground text-sm">
            {t('userBan.history.bannedAt')} <TimeDistance date={status.bannedAt} />
          </CardContent>
        )}
      </Card>

      <div className="space-y-3">
        {(status?.records ?? []).map((record) => (
          <Card key={record.id}>
            <CardContent className="grid gap-3 p-4 sm:grid-cols-[auto_1fr_auto] sm:items-start">
              <Badge
                variant={record.action === 'ban' ? 'destructive' : 'secondary'}
                className="w-fit whitespace-nowrap"
              >
                {t(record.action === 'ban' ? 'userBan.actions.ban' : 'userBan.actions.unban')}
              </Badge>
              <div className="min-w-0 space-y-1">
                <p className="text-sm break-words">{record.reason}</p>
                <p className="text-muted-foreground text-xs">
                  {t('userBan.history.operator', { name: record.operatorName })}
                </p>
              </div>
              <div className="text-muted-foreground text-xs sm:text-right">
                <TimeDistance date={record.createdAt} />
              </div>
            </CardContent>
          </Card>
        ))}
        {status?.records.length === 0 && (
          <Card>
            <CardContent className="text-muted-foreground p-6 text-center text-sm">
              {t('userBan.history.empty')}
            </CardContent>
          </Card>
        )}
      </div>
    </div>
  )
}
