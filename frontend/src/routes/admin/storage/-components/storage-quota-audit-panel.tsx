import { useQuery } from '@tanstack/react-query'
import { History, RefreshCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { JsonObject, getOperationLogs } from '@/services/api/admin/operationLog'

function auditDetailNumber(details: JsonObject, key: string): number | null {
  const value = details[key]
  return typeof value === 'number' ? value : null
}

function auditDetailString(details: JsonObject, key: string): string {
  const value = details[key]
  return typeof value === 'string' ? value : '-'
}

function formatBytes(bytes: number | null, unlimited: string): string {
  if (bytes === null) return '-'
  if (bytes < 0) return unlimited
  if (bytes === 0) return '0 B'

  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']
  const exponent = Math.min(Math.floor(Math.log(bytes) / Math.log(1024)), units.length - 1)
  return `${(bytes / 1024 ** exponent).toFixed(1)} ${units[exponent]}`
}

export default function StorageQuotaAuditPanel() {
  const { t } = useTranslation()
  const [userFilter, setUserFilter] = useState('')

  const auditQuery = useQuery({
    queryKey: ['admin', 'storage-quota-audit', userFilter],
    queryFn: () =>
      getOperationLogs({
        page: 1,
        limit: 100,
        operation_type: 'SetStorageQuota',
        target: userFilter || undefined,
      }).then((response) => response.data.items),
    staleTime: 30 * 1000,
  })

  const records = auditQuery.data ?? []

  return (
    <Card className="mt-4 overflow-hidden rounded-md shadow-xs">
      <CardHeader className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div>
          <CardTitle className="flex items-center gap-2 text-base">
            <History className="h-5 w-5" />
            {t('storageQuotaAudit.title')}
          </CardTitle>
          <CardDescription>{t('storageQuotaAudit.description')}</CardDescription>
        </div>
        <div className="flex w-full gap-2 md:w-auto">
          <Input
            className="min-w-0 md:w-56"
            value={userFilter}
            onChange={(event) => setUserFilter(event.target.value)}
            placeholder={t('storageQuotaAudit.userFilter')}
          />
          <Button
            variant="outline"
            size="icon"
            title={t('storageQuotaAudit.refresh')}
            aria-label={t('storageQuotaAudit.refresh')}
            onClick={() => void auditQuery.refetch()}
            disabled={auditQuery.isFetching}
          >
            <RefreshCcw className={`h-4 w-4 ${auditQuery.isFetching ? 'animate-spin' : ''}`} />
          </Button>
        </div>
      </CardHeader>
      <CardContent className="p-0">
        <div className="overflow-x-auto border-t">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('storageQuotaAudit.createdAt')}</TableHead>
                <TableHead>{t('storageQuotaAudit.operator')}</TableHead>
                <TableHead>{t('storageQuotaAudit.user')}</TableHead>
                <TableHead>{t('storageQuotaAudit.oldQuota')}</TableHead>
                <TableHead>{t('storageQuotaAudit.newQuota')}</TableHead>
                <TableHead>{t('storageQuotaAudit.provider')}</TableHead>
                <TableHead>{t('storageQuotaAudit.status')}</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {records.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} className="text-muted-foreground h-24 text-center">
                    {auditQuery.isLoading
                      ? t('storageQuotaAudit.loading')
                      : t('storageQuotaAudit.empty')}
                  </TableCell>
                </TableRow>
              ) : (
                records.map((record) => (
                  <TableRow key={record.id}>
                    <TableCell className="text-xs whitespace-nowrap">
                      {new Date(record.created_at).toLocaleString()}
                    </TableCell>
                    <TableCell>{record.operator}</TableCell>
                    <TableCell>{record.target}</TableCell>
                    <TableCell>
                      {formatBytes(
                        auditDetailNumber(record.details, 'old_quota'),
                        t('storageManagement.unlimited')
                      )}
                    </TableCell>
                    <TableCell>
                      {formatBytes(
                        auditDetailNumber(record.details, 'new_quota'),
                        t('storageManagement.unlimited')
                      )}
                    </TableCell>
                    <TableCell>{auditDetailString(record.details, 'provider')}</TableCell>
                    <TableCell>
                      <Badge variant={record.status === 'Success' ? 'default' : 'destructive'}>
                        {record.status === 'Success'
                          ? t('storageQuotaAudit.success')
                          : t('storageQuotaAudit.failed')}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))
              )}
            </TableBody>
          </Table>
        </div>
      </CardContent>
    </Card>
  )
}
