import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { AlertTriangle, RefreshCcw } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { RemoteDataTable } from '@/components/query-table/remote'

import {
  UserSpace,
  apiAdminGetStorageCapabilities,
  apiAdminGetUserSpaces,
  apiAdminRefreshUserSpaceUsage,
  apiAdminSetUserSpaceQuota,
} from '@/services/api/storage'

import useRemoteTableState from '@/hooks/use-remote-table-state'

import StorageQuotaAuditPanel from './-components/storage-quota-audit-panel'

export const Route = createFileRoute('/admin/storage/')({
  component: StorageManagementPage,
})

const QUOTA_UNIT_BYTES = {
  B: 1,
  KB: 1024,
  MB: 1024 ** 2,
  GB: 1024 ** 3,
  TB: 1024 ** 4,
} as const

function quotaValueForBytes(bytes: number, unitBytes: number): number {
  const rawValue = bytes / unitBytes
  for (let precision = 0; precision <= 12; precision += 1) {
    const candidate = Number(rawValue.toFixed(precision))
    if (Math.round(candidate * unitBytes) === bytes) return candidate
  }
  return rawValue
}

function normalizeQuotaDisplay(value: number, unit: string): { value: number; unit: string } {
  if (!Number.isFinite(value) || value < 0 || unit === 'unlimited') {
    return { value: Math.max(0, value || 0), unit }
  }

  const unitBytes = QUOTA_UNIT_BYTES[unit as keyof typeof QUOTA_UNIT_BYTES]
  if (!unitBytes) return { value, unit }

  const bytes = Math.round(value * unitBytes)
  const orderedUnits: Array<keyof typeof QUOTA_UNIT_BYTES> = ['TB', 'GB', 'MB', 'KB', 'B']

  const exactUnit = orderedUnits.find((candidate) => {
    const candidateBytes = QUOTA_UNIT_BYTES[candidate]
    return bytes >= candidateBytes && bytes % candidateBytes === 0
  })
  if (exactUnit) {
    return {
      value: bytes / QUOTA_UNIT_BYTES[exactUnit],
      unit: exactUnit,
    }
  }

  const fallbackUnit = orderedUnits.find((candidate) => bytes >= QUOTA_UNIT_BYTES[candidate]) ?? 'B'
  return {
    value: quotaValueForBytes(bytes, QUOTA_UNIT_BYTES[fallbackUnit]),
    unit: fallbackUnit,
  }
}

export default function StorageManagementPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const tableState = useRemoteTableState('admin-storage', { pageSize: 20 })

  // Quota confirmation dialog state.
  const [isQuotaDialogOpen, setIsQuotaDialogOpen] = useState(false)
  const [selectedUser, setSelectedUser] = useState<UserSpace | null>(null)
  const [quotaValue, setQuotaValue] = useState<number>(0)
  const [quotaUnit, setQuotaUnit] = useState<string>('GB')

  const storageCapabilitiesQuery = useQuery({
    queryKey: ['admin', 'storage', 'capabilities'],
    queryFn: () => apiAdminGetStorageCapabilities().then((res) => res.data),
    staleTime: 60 * 1000,
  })
  const storageCapabilities = storageCapabilitiesQuery.data
  const usageAvailable =
    !!storageCapabilities?.quota_enabled && !!storageCapabilities?.usage_readable
  const quotaReadable = usageAvailable && !!storageCapabilities?.quota_readable
  const quotaWritable = quotaReadable && !!storageCapabilities?.quota_writable

  const userSpacesQuery = useQuery({
    queryKey: [
      'admin',
      'user-spaces',
      tableState.pagination.pageIndex,
      tableState.pagination.pageSize,
    ],
    queryFn: async () => {
      const response = await apiAdminGetUserSpaces(
        tableState.pagination.pageIndex + 1,
        tableState.pagination.pageSize
      )
      return {
        items: response.data.items,
        total: response.data.total,
        page: response.data.page,
        page_size: response.data.pageSize,
      }
    },
    enabled: usageAvailable,
    staleTime: 5 * 60 * 1000,
    placeholderData: keepPreviousData,
  })

  // Apply a quota only after explicit confirmation in the dialog.
  const setQuotaMutation = useMutation({
    mutationFn: ({ user, quota }: { user: string; quota: number }) =>
      apiAdminSetUserSpaceQuota(user, quota),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      queryClient.invalidateQueries({ queryKey: ['admin', 'storage-quota-audit'] })
      toast.success(t('storageManagement.setQuotaSuccess'))
      setIsQuotaDialogOpen(false)
    },
  })

  const refreshUsageMutation = useMutation({
    mutationFn: apiAdminRefreshUserSpaceUsage,
    onSuccess: (response) => {
      queryClient.invalidateQueries({ queryKey: ['admin', 'user-spaces'] })
      toast.success(
        t('storageManagement.refreshSuccess', {
          updated: response.data.updated,
          failed: response.data.failed,
        })
      )
    },
  })

  const convertToBytes = (value: number, unit: string): number => {
    switch (unit) {
      case 'B':
        return Math.round(value)
      case 'KB':
        return Math.round(value * 1024)
      case 'MB':
        return Math.round(value * 1024 * 1024)
      case 'GB':
        return Math.round(value * 1024 * 1024 * 1024)
      case 'TB':
        return Math.round(value * 1024 * 1024 * 1024 * 1024)
      default:
        return Math.round(value)
    }
  }

  const alignQuotaInput = () => {
    if (quotaUnit === 'unlimited') return
    const normalized = normalizeQuotaDisplay(quotaValue, quotaUnit)
    setQuotaValue(normalized.value)
    setQuotaUnit(normalized.unit)
  }

  const handleSetQuota = () => {
    if (!selectedUser) return
    alignQuotaInput()
    const normalized = normalizeQuotaDisplay(quotaValue, quotaUnit)
    const quotaInBytes =
      normalized.unit === 'unlimited' ? -1 : convertToBytes(normalized.value, normalized.unit)
    if (quotaInBytes !== -1 && (!Number.isSafeInteger(quotaInBytes) || quotaInBytes <= 0)) {
      toast.error(t('storageManagement.invalidQuota'))
      return
    }
    setQuotaMutation.mutate({ user: selectedUser.user, quota: quotaInBytes })
  }

  const openSetQuotaDialog = (user: UserSpace) => {
    setSelectedUser(user)
    if (user.quota === -1) {
      setQuotaValue(0)
      setQuotaUnit('unlimited')
    } else {
      const normalized = normalizeQuotaDisplay(user.quota, 'B')
      setQuotaValue(normalized.value)
      setQuotaUnit(normalized.unit)
    }
    setIsQuotaDialogOpen(true)
  }

  const usageColumns: ColumnDef<UserSpace>[] = [
    {
      accessorKey: 'user',
      enableSorting: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('storageManagement.user')} />
      ),
    },
    {
      accessorKey: 'size',
      enableSorting: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('storageManagement.usedSpace')} />
      ),
      cell: ({ row }) =>
        row.original.size < 0 ? t('storageManagement.usagePending') : row.original.formatted,
    },
    {
      accessorKey: 'updated_at',
      enableSorting: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('storageManagement.refreshedAt')} />
      ),
      cell: ({ row }) =>
        row.original.updated_at
          ? new Date(row.original.updated_at).toLocaleString()
          : t('storageManagement.neverRefreshed'),
    },
  ]

  const quotaColumns: ColumnDef<UserSpace>[] = [
    {
      accessorKey: 'quota',
      enableSorting: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('storageManagement.quota')} />
      ),
      cell: ({ row }) => {
        const { quota, quota_formatted } = row.original
        return quota === -1 ? t('storageManagement.unlimited') : quota_formatted
      },
    },
    {
      accessorKey: 'usage_ratio',
      enableSorting: false,
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={t('storageManagement.usageRatio')} />
      ),
      cell: ({ row }) => {
        const { size, quota } = row.original
        if (size < 0 || quota <= 0) return '-'
        const ratio = (size / quota) * 100
        const color =
          ratio >= 90
            ? 'text-red-500 font-semibold'
            : ratio >= 70
              ? 'text-yellow-500'
              : 'text-green-600'
        return <span className={color}>{ratio.toFixed(1)}%</span>
      },
    },
  ]
  const quotaActionColumns: ColumnDef<UserSpace>[] = [
    {
      accessorKey: 'actions',
      enableSorting: false,
      header: t('storageManagement.actions'),
      cell: ({ row }) => {
        const user = row.original
        return (
          <div className="flex gap-2">
            <Button variant="ghost" size="sm" onClick={() => openSetQuotaDialog(user)}>
              {t('storageManagement.setQuota')}
            </Button>
          </div>
        )
      },
    },
  ]
  const columns = [
    ...usageColumns,
    ...(quotaReadable ? quotaColumns : []),
    ...(quotaWritable ? quotaActionColumns : []),
  ]
  const proposedQuota = quotaUnit === 'unlimited' ? -1 : convertToBytes(quotaValue, quotaUnit)
  const quotaBelowUsage =
    !!selectedUser &&
    selectedUser.size >= 0 &&
    proposedQuota > 0 &&
    proposedQuota <= selectedUser.size

  return (
    <>
      {usageAvailable ? (
        <RemoteDataTable
          info={{
            title: t('navigation.storageManagement'),
            description: quotaWritable
              ? t('storageManagement.description')
              : t('storageManagement.readOnlyDescription'),
          }}
          query={userSpacesQuery}
          state={tableState}
          columns={columns}
          getRowId={(row) => row.user}
        >
          <Button
            variant="outline"
            onClick={() => refreshUsageMutation.mutate()}
            disabled={refreshUsageMutation.isPending}
          >
            <RefreshCcw
              className={`mr-2 h-4 w-4 ${refreshUsageMutation.isPending ? 'animate-spin' : ''}`}
            />
            {refreshUsageMutation.isPending
              ? t('storageManagement.refreshingUsage')
              : t('storageManagement.refreshUsage')}
          </Button>
        </RemoteDataTable>
      ) : (
        <Card>
          <CardHeader>
            <CardTitle>{t('navigation.storageManagement')}</CardTitle>
            <CardDescription>{t('storageManagement.unavailable')}</CardDescription>
          </CardHeader>
          <CardContent className="text-muted-foreground text-sm">
            {storageCapabilitiesQuery.isLoading
              ? t('storageManagement.detecting')
              : storageCapabilities?.reasons?.join('; ') || t('storageManagement.requirements')}
          </CardContent>
        </Card>
      )}
      {quotaWritable && <StorageQuotaAuditPanel />}

      {quotaWritable && (
        <Dialog open={isQuotaDialogOpen} onOpenChange={setIsQuotaDialogOpen}>
          <DialogContent className="sm:max-w-md">
            <DialogHeader>
              <DialogTitle>
                {t('storageManagement.setQuotaFor', { user: selectedUser?.user })}
              </DialogTitle>
            </DialogHeader>
            <div className="grid gap-4 py-4">
              <div className="grid gap-2 rounded-md border p-3 text-sm">
                <div>
                  <span className="text-muted-foreground">
                    {t('storageManagement.currentQuota')}:
                  </span>{' '}
                  {selectedUser?.quota === -1
                    ? t('storageManagement.unlimited')
                    : selectedUser?.quota_formatted}
                </div>
                <div className="text-muted-foreground">
                  {t('storageManagement.currentUsage', {
                    usage:
                      selectedUser && selectedUser.size >= 0
                        ? selectedUser.formatted
                        : t('storageManagement.usagePending'),
                  })}
                </div>
              </div>
              <div className="grid grid-cols-4 items-center gap-4">
                <Label htmlFor="quota" className="text-right">
                  {t('storageManagement.newQuota')}
                </Label>
                <Input
                  id="quota"
                  type="number"
                  min={0}
                  step="any"
                  value={quotaValue}
                  onChange={(e) => setQuotaValue(Number(e.target.value))}
                  onBlur={alignQuotaInput}
                  disabled={quotaUnit === 'unlimited'}
                />
                <Select value={quotaUnit} onValueChange={setQuotaUnit}>
                  <SelectTrigger>
                    <SelectValue placeholder={t('storageManagement.selectUnit')} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="unlimited">{t('storageManagement.unlimited')}</SelectItem>
                    <SelectItem value="TB">TB</SelectItem>
                    <SelectItem value="GB">GB</SelectItem>
                    <SelectItem value="MB">MB</SelectItem>
                    <SelectItem value="KB">KB</SelectItem>
                    <SelectItem value="B">B</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              {quotaBelowUsage && (
                <div className="flex gap-2 rounded-md border border-amber-500/50 bg-amber-500/10 p-3 text-sm text-amber-700 dark:text-amber-300">
                  <AlertTriangle className="mt-0.5 size-4 shrink-0" />
                  {t('storageManagement.quotaBelowUsageWarning')}
                </div>
              )}
            </div>
            <DialogFooter>
              <Button
                type="button"
                onClick={handleSetQuota}
                disabled={setQuotaMutation.isPending || !selectedUser}
              >
                {setQuotaMutation.isPending
                  ? t('storageManagement.settingQuota')
                  : t('storageManagement.setQuota')}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      )}
    </>
  )
}
