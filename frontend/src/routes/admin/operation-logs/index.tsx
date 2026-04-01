import { useMutation, useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import {
  ColumnDef,
  PaginationState,
  SortingState,
  flexRender,
  getCoreRowModel,
  getSortedRowModel,
  useReactTable,
} from '@tanstack/react-table'
import { format } from 'date-fns'
import { ArrowRight, EyeIcon } from 'lucide-react'
import { toast } from 'sonner'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { DataTablePagination } from '@/components/query-table/pagination'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import PageTitle from '@/components/layout/page-title'

import {
  IGetOperationLogsParams,
  IOperationLog,
  clearOperationLogs,
  getOperationLogs,
} from '@/services/api/admin/operationLog'
import { showErrorToast } from '@/utils/toast'

type OperationLogFilters = {
  operation_type: string
  operator: string
  target: string
}

const INITIAL_FILTERS: OperationLogFilters = {
  operation_type: 'all',
  operator: '',
  target: '',
}

const DEFAULT_SORTING: SortingState = [
  {
    id: 'created_at',
    desc: true,
  },
]

const OPERATION_TYPE_KEYS = [
  'SetExclusive',
  'CancelExclusive',
  'SetUnschedulable',
  'CancelUnschedulable',
  'DrainNode',
  'UpdateVPA',
] as const

export const Route = createFileRoute('/admin/operation-logs/')({
  component: OperationLogsPage,
})

// VPA Diff Visualization Component
const ResourceChangeView = ({ changes }: { changes: Record<string, any> }) => {
  const { t } = useTranslation()

  if (!changes || Object.keys(changes).length === 0) {
    return (
      <p className="text-sm text-muted-foreground">
        {t('operationLog.details.empty', { defaultValue: '暂无变更记录' })}
      </p>
    )
  }

  const isVpaDiff = Object.values(changes).some(
    (val: any) => val && typeof val === 'object' && ('cpu' in val || 'memory' in val)
  )

  if (!isVpaDiff) {
    return (
      <div className="space-y-2">
        <p className="text-xs text-muted-foreground">
          {t('operationLog.details.rawJson', { defaultValue: '原始 JSON 详情' })}
        </p>
        <pre className="text-xs break-all whitespace-pre-wrap bg-muted/50 p-4 rounded-md">
          {JSON.stringify(changes, null, 2)}
        </pre>
      </div>
    )
  }

  const containerLabel = t('operationLog.details.containerLabel', { defaultValue: '容器' })
  const oldLabel = t('operationLog.details.oldValue', { defaultValue: '旧值' })
  const newLabel = t('operationLog.details.newValue', { defaultValue: '新值' })
  const naLabel = t('operationLog.details.na', { defaultValue: '无' })

  return (
    <div className="space-y-4">
      {Object.entries(changes).map(([containerName, resources]: [string, any]) => (
        <Card key={containerName} className="bg-muted/30">
          <CardHeader className="py-3">
            <CardTitle className="text-sm font-medium flex items-center gap-2">
              <span className="text-muted-foreground">{containerLabel}:</span> {containerName}
            </CardTitle>
          </CardHeader>
          <CardContent className="py-3">
            <div className="grid gap-2">
              {Object.entries(resources as Record<string, any>).map(([resourceName, values]) => {
                const oldValue = values?.old
                const newValue = values?.new
                if (!oldValue && !newValue) return null

                return (
                  <div key={resourceName} className="flex items-center text-sm border-b last:border-0 pb-2 last:pb-0">
                    <div className="w-28 font-medium text-muted-foreground capitalize">{resourceName}</div>
                    <div className="flex-1 flex items-center gap-2">
                      <code
                        className="bg-red-100 dark:bg-red-900/30 text-red-600 dark:text-red-400 px-1.5 py-0.5 rounded text-xs"
                        title={oldLabel}
                      >
                        {oldValue ?? naLabel}
                      </code>
                      <ArrowRight className="h-3 w-3 text-muted-foreground" aria-hidden />
                      <code
                        className="bg-green-100 dark:bg-green-900/30 text-green-600 dark:text-green-400 px-1.5 py-0.5 rounded text-xs"
                        title={newLabel}
                      >
                        {newValue ?? naLabel}
                      </code>
                    </div>
                  </div>
                )
              })}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

const VpaResourceDiff = ({ details }: { details?: Record<string, any> }) => {
  const { t } = useTranslation()
  const oldResources = (details?.oldResources ?? {}) as Record<string, Record<string, string>>
  const newResources = (details?.newResources ?? {}) as Record<string, Record<string, string>>

  const formatResourceLabel = (key: string) => {
    const labelMap: Record<string, string> = {
      cpu: t('operationLog.details.vpa.cpu', { defaultValue: 'CPU Limit' }),
      memory: t('operationLog.details.vpa.memory', { defaultValue: 'Memory Limit' }),
      'requests.cpu': t('operationLog.details.vpa.requestsCpu', { defaultValue: 'CPU Request' }),
      'requests.memory': t('operationLog.details.vpa.requestsMemory', { defaultValue: 'Memory Request' }),
    }
    if (labelMap[key]) return labelMap[key]
    return key.replace('requests.', `${t('operationLog.details.vpa.requestsPrefix', { defaultValue: 'Request ' })}`).toUpperCase()
  }

  const containers = Object.entries(oldResources)
  if (!containers.length) {
    return (
      <p className="text-sm text-muted-foreground">
        {t('operationLog.details.vpa.noData', { defaultValue: '暂无 VPA 资源变更记录' })}
      </p>
    )
  }

  return (
    <div className="space-y-4">
      {containers.map(([containerName, resources]) => {
        const nextResources = newResources?.[containerName] ?? {}
        const resourceKeys = Array.from(
          new Set([
            ...Object.keys(resources ?? {}),
            ...Object.keys(nextResources ?? {}),
          ])
        )

        if (!resourceKeys.length) {
          return null
        }

        return (
          <Card key={containerName} className="bg-muted/30">
            <CardHeader className="py-3">
              <CardTitle className="text-sm font-medium flex items-center gap-2">
                <span className="text-muted-foreground">
                  {t('operationLog.details.containerLabel', { defaultValue: '容器' })}:
                </span>
                {containerName || t('operationLog.details.containerLabel', { defaultValue: '容器' })}
              </CardTitle>
            </CardHeader>
            <CardContent className="space-y-2">
              {resourceKeys.map((key) => {
                const oldValue = resources?.[key] ?? t('operationLog.details.na', { defaultValue: '无' })
                const newValue = nextResources?.[key] ?? t('operationLog.details.na', { defaultValue: '无' })
                return (
                  <div
                    key={`${containerName}-${key}`}
                    className="flex items-center justify-between border-b pb-2 text-sm last:border-b-0 last:pb-0"
                  >
                    <span className="text-muted-foreground font-medium w-36 capitalize">
                      {formatResourceLabel(key)}
                    </span>
                    <div className="flex items-center gap-2 font-mono text-xs">
                      <span className="rounded bg-destructive/10 px-2 py-1 text-destructive">
                        {oldValue}
                      </span>
                      <ArrowRight className="h-3.5 w-3.5 text-muted-foreground" aria-hidden />
                      <span className="rounded bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                        {newValue}
                      </span>
                    </div>
                  </div>
                )
              })}
            </CardContent>
          </Card>
        )
      })}
    </div>
  )
}

function OperationLogsPage() {
  const { t } = useTranslation()
  const [pagination, setPagination] = useState<PaginationState>({
    pageIndex: 0,
    pageSize: 10,
  })

  const [filters, setFilters] = useState<OperationLogFilters>(INITIAL_FILTERS)
  const [sorting, setSorting] = useState<SortingState>(() => [...DEFAULT_SORTING])

  const handleFilterChange = (next: Partial<OperationLogFilters>) => {
    setFilters((prev) => ({ ...prev, ...next }))
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  // Dialog state for details
  const [selectedLog, setSelectedLog] = useState<IOperationLog | null>(null)

  const operationTypeLabelMap = useMemo<Record<string, string>>(
    () => ({
      SetExclusive: t('operationLog.type.SetExclusive', { defaultValue: '设为独占' }),
      CancelExclusive: t('operationLog.type.CancelExclusive', { defaultValue: '取消独占' }),
      SetUnschedulable: t('operationLog.type.SetUnschedulable', { defaultValue: '标记不可调度' }),
      CancelUnschedulable: t('operationLog.type.CancelUnschedulable', { defaultValue: '取消不可调度' }),
      DrainNode: t('operationLog.type.DrainNode', { defaultValue: '驱逐节点' }),
      UpdateVPA: t('operationLog.type.UpdateVPA', { defaultValue: '更新 VPA' }),
    }),
    [t]
  )

  const statusLabels = useMemo<Record<string, string>>(
    () => ({
      Success: t('operationLog.status.success', { defaultValue: '成功' }),
      Failed: t('operationLog.status.failed', { defaultValue: '失败' }),
    }),
    [t]
  )

  const operationTypeOptions = useMemo(
    () => [
      { value: 'all', label: t('operationLog.filters.allTypes', { defaultValue: '全部类型' }) },
      ...OPERATION_TYPE_KEYS.map((key) => ({
        value: key,
        label: operationTypeLabelMap[key] ?? key,
      })),
    ],
    [operationTypeLabelMap, t]
  )

  const columns = useMemo<ColumnDef<IOperationLog>[]>(
    () => [
      {
        accessorKey: 'created_at',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('operationLog.column.createdAt', { defaultValue: '操作时间' })}
          />
        ),
        cell: ({ row }) => (
          <span className="font-mono text-xs text-foreground">
            {format(new Date(row.original.created_at), 'yyyy-MM-dd HH:mm:ss')}
          </span>
        ),
      },
      {
        accessorKey: 'operator',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('operationLog.column.operator', { defaultValue: '操作人' })}
          />
        ),
        cell: ({ row }) => <span className="text-sm font-medium">{row.original.operator || '-'}</span>,
      },
      {
        accessorKey: 'operation_type',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('operationLog.column.type', { defaultValue: '操作类型' })}
          />
        ),
        cell: ({ row }) => {
          const typeLabel = operationTypeLabelMap[row.original.operation_type] ?? row.original.operation_type
          return <Badge variant="outline">{typeLabel}</Badge>
        },
      },
      {
        accessorKey: 'target',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('operationLog.column.target', { defaultValue: '操作对象' })}
          />
        ),
        cell: ({ row }) => row.original.target || '-',
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('operationLog.column.status', { defaultValue: '状态' })}
          />
        ),
        cell: ({ row }) => {
          const status = row.original.status
          const statusLabel = statusLabels[status] ?? status
          return (
            <Badge variant={status === 'Success' ? 'default' : 'destructive'}>
              {statusLabel}
            </Badge>
          )
        },
      },
      {
        id: 'actions',
        header: t('common.actions', { defaultValue: '操作' }),
        enableSorting: false,
        enableHiding: false,
        cell: ({ row }) => (
          <Button
            variant="ghost"
            size="icon"
            aria-label={t('operationLog.details', { defaultValue: '操作详情' })}
            onClick={() => setSelectedLog(row.original)}
          >
            <EyeIcon className="h-4 w-4" />
          </Button>
        ),
      },
    ],
    [operationTypeLabelMap, statusLabels, t]
  )

  const queryParams: IGetOperationLogsParams = useMemo(() => {
    const params: IGetOperationLogsParams = {
      page: pagination.pageIndex + 1,
      limit: pagination.pageSize,
    }
    if (filters.operation_type && filters.operation_type !== 'all') params.operation_type = filters.operation_type
    if (filters.operator) params.operator = filters.operator
    if (filters.target) params.target = filters.target
    return params
  }, [pagination, filters])

  const { data, isLoading, refetch, dataUpdatedAt } = useQuery({
    queryKey: ['operation-logs', queryParams],
    queryFn: () => getOperationLogs(queryParams),
  })

  const { mutate: handleClearLogs, isPending: isClearing } = useMutation({
    mutationFn: clearOperationLogs,
    onSuccess: () => {
      toast.success(
        t('operationLog.clear.success', {
          defaultValue: 'All operation logs have been cleared',
        })
      )
      void refetch()
    },
    onError: (error) => {
      showErrorToast(error)
    },
  })

  // Memoize empty array to prevent re-renders
  const defaultData = useMemo(() => [], [])

  const table = useReactTable({
    data: data?.data?.items ?? defaultData,
    columns,
    pageCount: data?.data?.total ? Math.ceil(data.data.total / pagination.pageSize) : -1,
    state: {
      pagination,
      sorting,
    },
    onPaginationChange: setPagination,
    onSortingChange: setSorting,
    getCoreRowModel: getCoreRowModel(),
    getSortedRowModel: getSortedRowModel(),
    manualPagination: true,
  })

  const resetFilters = () => {
    setFilters(INITIAL_FILTERS)
    setPagination((p) => ({ ...p, pageIndex: 0 }))
    setSorting([...DEFAULT_SORTING])
  }

  const lastUpdatedAt = useMemo(() => {
    if (!dataUpdatedAt) return '--'
    return new Date(dataUpdatedAt).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }, [dataUpdatedAt])

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 sm:p-6 md:gap-6">
      <PageTitle
        title={t('operationLog.title', { defaultValue: '操作日志' })}
        description={t('operationLog.description', {
          defaultValue: '审计和追踪系统关键操作记录。',
        })}
      >
        <AlertDialog>
          <AlertDialogTrigger asChild>
            <Button variant="destructive" size="sm" disabled={isClearing}>
              {isClearing
                ? t('operationLog.clear.loading', { defaultValue: '清空中...' })
                : t('operationLog.clear.action', { defaultValue: '清空全部日志' })}
            </Button>
          </AlertDialogTrigger>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t('operationLog.clear.confirmTitle', { defaultValue: '确认清空所有日志？' })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t('operationLog.clear.confirmDescription', {
                  defaultValue: '该操作会删除数据库中的全部操作日志，仅在测试场景下使用且不可恢复。',
                })}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel disabled={isClearing}>
                {t('common.cancel', { defaultValue: '取消' })}
              </AlertDialogCancel>
              <AlertDialogAction
                disabled={isClearing}
                className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                onClick={() => handleClearLogs()}
              >
                {isClearing
                  ? t('operationLog.clear.loading', { defaultValue: '清空中...' })
                  : t('operationLog.clear.confirmAction', { defaultValue: '立即清空' })}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      </PageTitle>

      <section className="rounded-lg border bg-card/40 p-4 shadow-xs">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 lg:items-end">
          <div className="flex flex-col gap-1">
            <span className="text-xs font-medium text-muted-foreground">
              {t('operationLog.column.type', { defaultValue: '操作类型' })}
            </span>
            <Select
              value={filters.operation_type}
              onValueChange={(val) => handleFilterChange({ operation_type: val })}
            >
              <SelectTrigger className="h-9 w-full">
                <SelectValue
                  placeholder={t('operationLog.column.type', { defaultValue: '操作类型' })}
                />
              </SelectTrigger>
              <SelectContent>
                {operationTypeOptions.map((option) => (
                  <SelectItem key={option.value} value={option.value}>
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="flex flex-col gap-1">
            <span className="text-xs font-medium text-muted-foreground">
              {t('operationLog.column.operator', { defaultValue: '操作人' })}
            </span>
            <Input
              value={filters.operator}
              onChange={(e) => handleFilterChange({ operator: e.target.value })}
              placeholder={t('operationLog.filters.operatorPlaceholder', {
                defaultValue: '输入操作人姓名，例如：张三',
              })}
              className="h-9"
            />
          </div>
          <div className="flex flex-col gap-1">
            <span className="text-xs font-medium text-muted-foreground">
              {t('operationLog.column.target', { defaultValue: '操作对象' })}
            </span>
            <Input
              value={filters.target}
              onChange={(e) => handleFilterChange({ target: e.target.value })}
              placeholder={t('operationLog.filters.targetPlaceholder', {
                defaultValue: '输入操作对象，例如：节点或资源',
              })}
              className="h-9"
            />
          </div>
          <div className="flex items-end">
            <Button variant="outline" className="h-9 w-full lg:w-auto" onClick={resetFilters}>
              {t('common.reset', { defaultValue: '重置' })}
            </Button>
          </div>
        </div>
      </section>

      <div className="overflow-hidden rounded-lg border bg-background shadow-xs">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="whitespace-nowrap px-4 py-3 text-xs font-medium text-muted-foreground"
                  >
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {table.getRowModel().rows?.length ? (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id} className="px-4 py-3 text-sm">
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            ) : (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center text-sm">
                  {isLoading ? (
                    <div className="flex items-center justify-center">
                      <LoadingCircleIcon className="h-6 w-6 animate-spin text-muted-foreground" />
                    </div>
                  ) : (
                    t('common.noData', { defaultValue: '暂无数据' })
                  )}
                </TableCell>
              </TableRow>
            )}
          </TableBody>
        </Table>
      </div>
      <div className="pt-2">
        <DataTablePagination table={table} updatedAt={lastUpdatedAt} refetch={() => void refetch()} />
      </div>

      <Dialog open={!!selectedLog} onOpenChange={(open) => !open && setSelectedLog(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>
              {t('operationLog.details', { defaultValue: '操作详情' })}
            </DialogTitle>
          </DialogHeader>
          <ScrollArea className="max-h-[80vh] w-full pr-4">
            <div className="space-y-6">
              <div className="grid gap-y-4 gap-x-8 text-sm md:grid-cols-2">
                <div className="flex flex-col gap-1">
                  <span className="font-medium text-muted-foreground">{t('operationLog.column.operator', { defaultValue: '操作人' })}</span>
                  <span>{selectedLog?.operator}</span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="font-medium text-muted-foreground">{t('operationLog.column.createdAt', { defaultValue: '操作时间' })}</span>{' '}
                  <span>
                    {selectedLog &&
                      format(new Date(selectedLog.created_at), 'yyyy-MM-dd HH:mm:ss')}
                  </span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="font-medium text-muted-foreground">{t('operationLog.column.type', { defaultValue: '操作类型' })}</span>
                  <span>{selectedLog ? operationTypeLabelMap[selectedLog.operation_type] ?? selectedLog.operation_type : '-'}</span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="font-medium text-muted-foreground">{t('operationLog.column.target', { defaultValue: '操作对象' })}</span>
                  <span>{selectedLog?.target}</span>
                </div>
                <div className="flex flex-col gap-1">
                  <span className="font-medium text-muted-foreground">{t('operationLog.column.status', { defaultValue: '状态' })}</span>
                  <div className="flex">
                    <Badge variant={selectedLog?.status === 'Success' ? 'default' : 'destructive'}>
                        {selectedLog ? statusLabels[selectedLog.status] ?? selectedLog.status : ''}
                    </Badge>
                  </div>
                </div>
              </div>
              
              {selectedLog?.error_message && (
                  <div className="rounded-md bg-destructive/10 p-4 text-destructive text-sm">
                      <p className="font-semibold mb-1">
                        {t('operationLog.details.errorMessageTitle', { defaultValue: '错误信息' })}
                      </p>
                      <pre className="whitespace-pre-wrap font-mono text-xs">{selectedLog.error_message}</pre>
                  </div>
              )}

              <div className="space-y-2">
                <p className="font-semibold text-sm">
                  {t('operationLog.details.sectionTitle', { defaultValue: '变更详情' })}
                </p>
                {selectedLog &&
                  (selectedLog.operation_type === 'UpdateVPA' ? (
                    <VpaResourceDiff details={selectedLog.details} />
                  ) : (
                    <ResourceChangeView changes={selectedLog.details} />
                  ))}
              </div>
            </div>
          </ScrollArea>
        </DialogContent>
      </Dialog>
    </div>
  )
}
