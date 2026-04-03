import { useQuery } from '@tanstack/react-query'
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
import { ArrowRight, CalendarIcon, EyeIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import LoadingCircleIcon from '@/components/icon/loading-circle-icon'
import PageTitle from '@/components/layout/page-title'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTablePagination } from '@/components/query-table/pagination'

import {
  IGetOperationLogsParams,
  IOperationLog,
  JsonObject,
  JsonValue,
  getOperationLogs,
} from '@/services/api/admin/operationLog'

import { betterResourceQuantity, convertKResourceToResource } from '@/utils/resource'

import { cn } from '@/lib/utils'

type OperationLogFilters = {
  operation_type: string
  operator: string
  target: string
}

type ResourceDiffRow = {
  resourceName: string
  oldValue: string
  newValue: string
}

type ResourceDiffCard = {
  containerName: string
  rows: ResourceDiffRow[]
}

type ContainerResourceMap = Record<string, Record<string, string>>
type OperationLogTimeRange = 'all' | '1d' | '3d' | '7d' | '15d' | '1m' | '3m'

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
  'DeleteJob',
  'SetExclusive',
  'CancelExclusive',
  'SetUnschedulable',
  'CancelUnschedulable',
  'DrainNode',
  'UpdateVPA',
] as const

const OPERATION_LOG_TIME_RANGE_OPTIONS: Array<{
  value: OperationLogTimeRange
  labelKey: string
  defaultValue: string
}> = [
  {
    value: '1d',
    labelKey: 'operationLog.timeRange.1d',
    defaultValue: '近1天日志',
  },
  {
    value: '3d',
    labelKey: 'operationLog.timeRange.3d',
    defaultValue: '近3天日志',
  },
  {
    value: '7d',
    labelKey: 'operationLog.timeRange.7d',
    defaultValue: '近7天日志',
  },
  {
    value: '15d',
    labelKey: 'operationLog.timeRange.15d',
    defaultValue: '近15天日志',
  },
  {
    value: '1m',
    labelKey: 'operationLog.timeRange.1m',
    defaultValue: '近1个月日志',
  },
  {
    value: '3m',
    labelKey: 'operationLog.timeRange.3m',
    defaultValue: '近3个月日志',
  },
  {
    value: 'all',
    labelKey: 'operationLog.timeRange.all',
    defaultValue: '全部日志',
  },
]

const OPERATION_TYPE_BADGE_STYLES: Record<string, string> = {
  DeleteJob: 'text-highlight-red bg-highlight-red/20',
  SetExclusive: 'text-highlight-yellow bg-highlight-yellow/20',
  CancelExclusive: 'text-highlight-yellow bg-highlight-yellow/20',
  SetUnschedulable: 'text-highlight-orange bg-highlight-orange/20',
  CancelUnschedulable: 'text-highlight-orange bg-highlight-orange/20',
  DrainNode: 'text-highlight-red bg-highlight-red/20',
  UpdateVPA: 'text-highlight-blue bg-highlight-blue/20',
}

const DEFAULT_OPERATION_TYPE_BADGE_STYLE = 'text-highlight-slate bg-highlight-slate/20'

const CPU_MILLI_CORES = 1000
const KIBI_BYTES = 1024
const MEBI_BYTES = 1024 * KIBI_BYTES
const GIBI_BYTES = 1024 * MEBI_BYTES
const TEBI_BYTES = 1024 * GIBI_BYTES

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === 'object' && value !== null && !Array.isArray(value)

const formatDecimal = (value: number): string => {
  if (Number.isInteger(value)) {
    return `${value}`
  }

  return value.toFixed(3).replace(/\.?0+$/, '')
}

const parseCpuToMillicores = (value: string): number | null => {
  if (/^\d+m$/.test(value)) {
    return Number.parseInt(value.slice(0, -1), 10)
  }

  if (/^\d+(\.\d+)?$/.test(value)) {
    return Math.round(Number.parseFloat(value) * CPU_MILLI_CORES)
  }

  return null
}

const parseMemoryToBytes = (value: string): number | null => {
  if (/^\d+$/.test(value)) {
    return Number.parseInt(value, 10)
  }

  const match = value.match(/^(\d+)(Ki|Mi|Gi|Ti)$/)
  if (!match) {
    return null
  }

  const amount = Number.parseInt(match[1], 10)
  const unit = match[2]

  switch (unit) {
    case 'Ki':
      return amount * KIBI_BYTES
    case 'Mi':
      return amount * MEBI_BYTES
    case 'Gi':
      return amount * GIBI_BYTES
    case 'Ti':
      return amount * TEBI_BYTES
    default:
      return null
  }
}

const formatMemoryBytes = (bytes: number): string => {
  if (bytes === 0) {
    return '0'
  }

  const units = [
    ['Ti', TEBI_BYTES],
    ['Gi', GIBI_BYTES],
    ['Mi', MEBI_BYTES],
    ['Ki', KIBI_BYTES],
  ] as const

  for (const [unit, size] of units) {
    if (bytes % size === 0) {
      return `${bytes / size}${unit}`
    }
  }

  return `${bytes}B`
}

const normalizeResourceQuantityForDisplay = (
  resourceName: string,
  value: string,
  options?: { forceMemoryGi?: boolean }
): string => {
  if (resourceName.endsWith('cpu')) {
    const millicores = parseCpuToMillicores(value)
    if (millicores === null) {
      return value
    }

    const normalizedMillicores =
      resourceName === 'cpu' && value.endsWith('m') ? Math.max(millicores - 1, 0) : millicores

    return `${formatDecimal(normalizedMillicores / CPU_MILLI_CORES)}C`
  }

  if (resourceName.endsWith('memory')) {
    const bytes = parseMemoryToBytes(value)
    if (bytes === null) {
      return value
    }

    const normalizedBytes = resourceName === 'memory' ? bytes + MEBI_BYTES : bytes

    if (options?.forceMemoryGi) {
      return `${formatDecimal(normalizedBytes / GIBI_BYTES)}Gi`
    }

    return formatMemoryBytes(normalizedBytes)
  }

  return value
}

const formatResourceValueForDisplay = (
  resourceName: string,
  value: JsonValue | undefined,
  naLabel: string,
  options?: { forceMemoryGi?: boolean }
): string => {
  if (value === undefined || value === null || value === '') {
    return naLabel
  }

  if (typeof value === 'string') {
    return normalizeResourceQuantityForDisplay(resourceName, value, options)
  }

  if (typeof value === 'number' || typeof value === 'boolean') {
    return `${value}`
  }

  return JSON.stringify(value)
}

const getResourceChangeCards = (
  changes: JsonObject,
  naLabel: string
): ResourceDiffCard[] | null => {
  let hasDiffStructure = false
  const cards: ResourceDiffCard[] = []

  for (const [containerName, rawResources] of Object.entries(changes)) {
    if (!isRecord(rawResources)) {
      continue
    }

    const rows: ResourceDiffRow[] = []

    for (const [resourceName, rawValues] of Object.entries(rawResources)) {
      if (!isRecord(rawValues)) {
        continue
      }

      const oldValue = formatResourceValueForDisplay(
        resourceName,
        rawValues.old as JsonValue,
        naLabel
      )
      const newValue = formatResourceValueForDisplay(
        resourceName,
        rawValues.new as JsonValue,
        naLabel
      )

      if (rawValues.old === undefined && rawValues.new === undefined) {
        continue
      }

      hasDiffStructure = true

      if (newValue === naLabel || oldValue === newValue) {
        continue
      }

      rows.push({
        resourceName,
        oldValue,
        newValue,
      })
    }

    if (rows.length > 0) {
      cards.push({
        containerName,
        rows,
      })
    }
  }

  return hasDiffStructure ? cards : null
}

const toContainerResourceMap = (value: JsonValue | undefined): ContainerResourceMap => {
  if (!isRecord(value)) {
    return {}
  }

  const result: ContainerResourceMap = {}

  for (const [containerName, rawResources] of Object.entries(value)) {
    if (!isRecord(rawResources)) {
      continue
    }

    const resources: Record<string, string> = {}

    for (const [resourceName, rawValue] of Object.entries(rawResources)) {
      if (typeof rawValue === 'string') {
        resources[resourceName] = rawValue
      }
    }

    if (Object.keys(resources).length > 0) {
      result[containerName] = resources
    }
  }

  return result
}

const formatVpaLimitValue = (
  resourceName: 'cpu' | 'memory',
  value: string,
  naLabel: string
): string => {
  const normalizedValue =
    resourceName === 'memory' && /^\d+$/.test(value)
      ? `${Math.round(Number.parseInt(value, 10) / GIBI_BYTES)}Gi`
      : value

  const converted = convertKResourceToResource(resourceName, normalizedValue)
  if (converted === undefined) {
    return naLabel
  }

  return betterResourceQuantity(resourceName, converted, true)
}

const getOperationLogStartTime = (timeRange: OperationLogTimeRange): string | undefined => {
  if (timeRange === 'all') {
    return undefined
  }

  const now = new Date()
  const start = new Date(now)

  switch (timeRange) {
    case '1d':
      start.setDate(start.getDate() - 1)
      break
    case '3d':
      start.setDate(start.getDate() - 3)
      break
    case '7d':
      start.setDate(start.getDate() - 7)
      break
    case '15d':
      start.setDate(start.getDate() - 15)
      break
    case '1m':
      start.setMonth(start.getMonth() - 1)
      break
    case '3m':
      start.setMonth(start.getMonth() - 3)
      break
    default:
      return undefined
  }

  return start.toISOString()
}

const OperationTypeBadge = ({ operationType, label }: { operationType: string; label: string }) => (
  <Badge
    variant="outline"
    className={cn(
      'border-none',
      OPERATION_TYPE_BADGE_STYLES[operationType] ?? DEFAULT_OPERATION_TYPE_BADGE_STYLE
    )}
  >
    {label}
  </Badge>
)

const DeleteJobDetailsView = ({ details }: { details?: JsonObject }) => {
  const { t } = useTranslation()

  if (!details || Object.keys(details).length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('operationLog.details.empty', { defaultValue: '暂无变更记录' })}
      </p>
    )
  }

  const detailItems = [
    {
      key: 'jobDisplayName',
      label: t('operationLog.details.deleteJob.jobDisplayName', { defaultValue: '作业名称' }),
    },
    {
      key: 'jobName',
      label: t('operationLog.details.deleteJob.jobName', { defaultValue: '作业 ID' }),
    },
    {
      key: 'owner',
      label: t('operationLog.details.deleteJob.owner', { defaultValue: '所属用户' }),
    },
    {
      key: 'account',
      label: t('operationLog.details.deleteJob.account', { defaultValue: '所属账户' }),
    },
    {
      key: 'jobType',
      label: t('operationLog.details.deleteJob.jobType', { defaultValue: '作业类型' }),
    },
    {
      key: 'previousStatus',
      label: t('operationLog.details.deleteJob.previousStatus', { defaultValue: '删除前状态' }),
    },
    {
      key: 'deletedRecord',
      label: t('operationLog.details.deleteJob.deletedRecord', { defaultValue: '删除数据库记录' }),
    },
    {
      key: 'deletedClusterJob',
      label: t('operationLog.details.deleteJob.deletedClusterJob', {
        defaultValue: '删除集群作业',
      }),
    },
  ]

  const rows = detailItems.reduce<Array<{ key: string; label: string; value: string }>>(
    (acc, item) => {
      const rawValue = details[item.key]

      if (rawValue === undefined || rawValue === null || rawValue === '') {
        return acc
      }

      const value =
        typeof rawValue === 'boolean'
          ? t(rawValue ? 'common.yes' : 'common.no', { defaultValue: rawValue ? '是' : '否' })
          : `${rawValue}`

      acc.push({
        key: item.key,
        label: item.label,
        value,
      })

      return acc
    },
    []
  )

  if (rows.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('operationLog.details.empty', { defaultValue: '暂无变更记录' })}
      </p>
    )
  }

  return (
    <Card className="bg-muted/30">
      <CardContent className="grid gap-3 py-4 sm:grid-cols-2">
        {rows.map(({ key, label, value }) => (
          <div key={key} className="flex flex-col gap-1 text-sm">
            <span className="text-muted-foreground font-medium">{label}</span>
            <span className="break-all">{value}</span>
          </div>
        ))}
      </CardContent>
    </Card>
  )
}

export const Route = createFileRoute('/admin/operation-logs/')({
  component: OperationLogsPage,
})

const ResourceChangeView = ({ changes }: { changes: JsonObject }) => {
  const { t } = useTranslation()

  if (Object.keys(changes).length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('operationLog.details.empty', { defaultValue: '暂无变更记录' })}
      </p>
    )
  }

  const naLabel = t('operationLog.details.na', { defaultValue: '无' })
  const cards = getResourceChangeCards(changes, naLabel)

  if (cards === null) {
    return (
      <div className="space-y-2">
        <p className="text-muted-foreground text-xs">
          {t('operationLog.details.rawJson', { defaultValue: '原始 JSON 详情' })}
        </p>
        <pre className="bg-muted/50 rounded-md p-4 text-xs break-all whitespace-pre-wrap">
          {JSON.stringify(changes, null, 2)}
        </pre>
      </div>
    )
  }

  if (cards.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('operationLog.details.empty', { defaultValue: '暂无变更记录' })}
      </p>
    )
  }

  const containerLabel = t('operationLog.details.containerLabel', { defaultValue: '容器' })
  const oldLabel = t('operationLog.details.oldValue', { defaultValue: '旧值' })
  const newLabel = t('operationLog.details.newValue', { defaultValue: '新值' })

  return (
    <div className="space-y-4">
      {cards.map(({ containerName, rows }) => (
        <Card key={containerName} className="bg-muted/30">
          <CardHeader className="py-3">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <span className="text-muted-foreground">{containerLabel}:</span>
              {containerName}
            </CardTitle>
          </CardHeader>
          <CardContent className="py-3">
            <div className="grid gap-2">
              {rows.map(({ resourceName, oldValue, newValue }) => (
                <div
                  key={resourceName}
                  className="flex items-center border-b pb-2 text-sm last:border-0 last:pb-0"
                >
                  <div className="text-muted-foreground w-28 font-medium capitalize">
                    {resourceName}
                  </div>
                  <div className="flex flex-1 items-center gap-2">
                    <code
                      className="rounded bg-red-100 px-1.5 py-0.5 text-xs text-red-600 dark:bg-red-900/30 dark:text-red-400"
                      title={oldLabel}
                    >
                      {oldValue}
                    </code>
                    <ArrowRight className="text-muted-foreground h-3 w-3" aria-hidden />
                    <code
                      className="rounded bg-green-100 px-1.5 py-0.5 text-xs text-green-600 dark:bg-green-900/30 dark:text-green-400"
                      title={newLabel}
                    >
                      {newValue}
                    </code>
                  </div>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

const VpaResourceDiff = ({ details }: { details?: JsonObject }) => {
  const { t } = useTranslation()
  const oldResources = toContainerResourceMap(details?.oldResources)
  const newResources = toContainerResourceMap(details?.newResources)

  const formatResourceLabel = (key: string) => {
    const labelMap: Record<string, string> = {
      cpu: t('operationLog.details.vpa.cpu', { defaultValue: 'CPU Limit' }),
      memory: t('operationLog.details.vpa.memory', { defaultValue: 'Memory Limit' }),
      'requests.cpu': t('operationLog.details.vpa.requestsCpu', { defaultValue: 'CPU Request' }),
      'requests.memory': t('operationLog.details.vpa.requestsMemory', {
        defaultValue: 'Memory Request',
      }),
    }

    if (labelMap[key]) {
      return labelMap[key]
    }

    return key
      .replace(
        'requests.',
        `${t('operationLog.details.vpa.requestsPrefix', { defaultValue: 'Request ' })}`
      )
      .toUpperCase()
  }

  const naLabel = t('operationLog.details.na', { defaultValue: '无' })
  const containerNames = Array.from(
    new Set([...Object.keys(oldResources), ...Object.keys(newResources)])
  )
  const resourceNamesToDisplay: Array<'cpu' | 'memory'> = ['cpu', 'memory']

  const cards = containerNames.reduce<ResourceDiffCard[]>((acc, containerName) => {
    const previousResources = oldResources[containerName] ?? {}
    const nextResources = newResources[containerName] ?? {}

    const rows = resourceNamesToDisplay.reduce<ResourceDiffRow[]>((resourceAcc, resourceName) => {
      const previousValue = previousResources[resourceName]
      const nextValue = nextResources[resourceName]

      if (!previousValue && !nextValue) {
        return resourceAcc
      }

      const oldValue =
        typeof previousValue === 'string'
          ? formatVpaLimitValue(resourceName, previousValue, naLabel)
          : naLabel
      const newValue =
        typeof nextValue === 'string'
          ? formatVpaLimitValue(resourceName, nextValue, naLabel)
          : naLabel

      if (newValue === naLabel || oldValue === newValue) {
        return resourceAcc
      }

      resourceAcc.push({
        resourceName,
        oldValue,
        newValue,
      })

      return resourceAcc
    }, [])

    if (rows.length > 0) {
      acc.push({
        containerName,
        rows,
      })
    }

    return acc
  }, [])

  if (cards.length === 0) {
    return (
      <p className="text-muted-foreground text-sm">
        {t('operationLog.details.vpa.noData', { defaultValue: '暂无 VPA 资源变更记录' })}
      </p>
    )
  }

  return (
    <div className="space-y-4">
      {cards.map(({ containerName, rows }) => (
        <Card key={containerName} className="bg-muted/30">
          <CardHeader className="py-3">
            <CardTitle className="flex items-center gap-2 text-sm font-medium">
              <span className="text-muted-foreground">
                {t('operationLog.details.containerLabel', { defaultValue: '容器' })}:
              </span>
              {containerName || t('operationLog.details.containerLabel', { defaultValue: '容器' })}
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            {rows.map(({ resourceName, oldValue, newValue }) => (
              <div
                key={`${containerName}-${resourceName}`}
                className="flex items-center justify-between border-b pb-2 text-sm last:border-b-0 last:pb-0"
              >
                <span className="text-muted-foreground w-36 font-medium capitalize">
                  {formatResourceLabel(resourceName)}
                </span>
                <div className="flex items-center gap-2 font-mono text-xs">
                  <span className="bg-destructive/10 text-destructive rounded px-2 py-1">
                    {oldValue}
                  </span>
                  <ArrowRight className="text-muted-foreground h-3.5 w-3.5" aria-hidden />
                  <span className="rounded bg-emerald-100 px-2 py-1 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300">
                    {newValue}
                  </span>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      ))}
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
  const [timeRange, setTimeRange] = useState<OperationLogTimeRange>('all')
  const [sorting, setSorting] = useState<SortingState>(() => [...DEFAULT_SORTING])
  const [selectedLog, setSelectedLog] = useState<IOperationLog | null>(null)

  const handleFilterChange = (next: Partial<OperationLogFilters>) => {
    setFilters((prev) => ({ ...prev, ...next }))
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
  }

  const operationTypeLabelMap = useMemo<Record<string, string>>(
    () => ({
      DeleteJob: t('operationLog.type.DeleteJob', { defaultValue: '删除作业' }),
      SetExclusive: t('operationLog.type.SetExclusive', { defaultValue: '设为独占' }),
      CancelExclusive: t('operationLog.type.CancelExclusive', { defaultValue: '取消独占' }),
      SetUnschedulable: t('operationLog.type.SetUnschedulable', {
        defaultValue: '标记不可调度',
      }),
      CancelUnschedulable: t('operationLog.type.CancelUnschedulable', {
        defaultValue: '取消不可调度',
      }),
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
          <span className="text-foreground font-mono text-xs">
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
        cell: ({ row }) => (
          <span className="text-sm font-medium">{row.original.operator || '-'}</span>
        ),
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
          const operationType = row.original.operation_type
          const label = operationTypeLabelMap[operationType] ?? operationType

          return <OperationTypeBadge operationType={operationType} label={label} />
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
            <Badge variant={status === 'Success' ? 'default' : 'destructive'}>{statusLabel}</Badge>
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
    const startTime = getOperationLogStartTime(timeRange)

    if (filters.operation_type && filters.operation_type !== 'all') {
      params.operation_type = filters.operation_type
    }
    if (filters.operator) {
      params.operator = filters.operator
    }
    if (filters.target) {
      params.target = filters.target
    }
    if (startTime) {
      params.start_time = startTime
      params.end_time = new Date().toISOString()
    }

    return params
  }, [filters, pagination, timeRange])

  const { data, isLoading, refetch, dataUpdatedAt } = useQuery({
    queryKey: ['operation-logs', queryParams],
    queryFn: () => getOperationLogs(queryParams),
  })

  const defaultData = useMemo<IOperationLog[]>(() => [], [])

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
    setTimeRange('all')
    setPagination((prev) => ({ ...prev, pageIndex: 0 }))
    setSorting([...DEFAULT_SORTING])
  }

  const lastUpdatedAt = useMemo(() => {
    if (!dataUpdatedAt) {
      return '--'
    }

    return new Date(dataUpdatedAt).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }, [dataUpdatedAt])

  const timeRangeLabelMap = useMemo(
    () =>
      Object.fromEntries(
        OPERATION_LOG_TIME_RANGE_OPTIONS.map((option) => [
          option.value,
          t(option.labelKey, { defaultValue: option.defaultValue }),
        ])
      ) as Record<OperationLogTimeRange, string>,
    [t]
  )

  return (
    <div className="flex flex-1 flex-col gap-4 p-4 sm:p-6 md:gap-6">
      <PageTitle
        title={t('operationLog.title', { defaultValue: '操作日志' })}
        description={t('operationLog.description', {
          defaultValue: '审计和追踪系统关键操作记录。',
        })}
      >
        <div className="flex items-center">
          <Select
            value={timeRange}
            onValueChange={(value) => {
              setTimeRange(value as OperationLogTimeRange)
              setPagination((prev) => ({ ...prev, pageIndex: 0 }))
            }}
          >
            <SelectTrigger className="bg-background h-9 w-[156px] pr-2 pl-3">
              <CalendarIcon />
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {OPERATION_LOG_TIME_RANGE_OPTIONS.map((option) => (
                <SelectItem key={option.value} value={option.value}>
                  {timeRangeLabelMap[option.value]}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>
      </PageTitle>

      <section className="bg-card/40 rounded-lg border p-4 shadow-xs">
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 lg:items-end">
          <div className="flex flex-col gap-1">
            <span className="text-muted-foreground text-xs font-medium">
              {t('operationLog.column.type', { defaultValue: '操作类型' })}
            </span>
            <Select
              value={filters.operation_type}
              onValueChange={(value) => handleFilterChange({ operation_type: value })}
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
            <span className="text-muted-foreground text-xs font-medium">
              {t('operationLog.column.operator', { defaultValue: '操作人' })}
            </span>
            <Input
              value={filters.operator}
              onChange={(event) => handleFilterChange({ operator: event.target.value })}
              placeholder={t('operationLog.filters.operatorPlaceholder', {
                defaultValue: '输入操作人姓名，例如：张三',
              })}
              className="h-9"
            />
          </div>

          <div className="flex flex-col gap-1">
            <span className="text-muted-foreground text-xs font-medium">
              {t('operationLog.column.target', { defaultValue: '操作对象' })}
            </span>
            <Input
              value={filters.target}
              onChange={(event) => handleFilterChange({ target: event.target.value })}
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

      <div className="bg-background overflow-hidden rounded-lg border shadow-xs">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead
                    key={header.id}
                    className="text-muted-foreground px-4 py-3 text-xs font-medium whitespace-nowrap"
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
            {table.getRowModel().rows.length > 0 ? (
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
                      <LoadingCircleIcon className="text-muted-foreground h-6 w-6 animate-spin" />
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
        <DataTablePagination
          table={table}
          updatedAt={lastUpdatedAt}
          refetch={() => void refetch()}
        />
      </div>

      <Dialog open={!!selectedLog} onOpenChange={(open) => !open && setSelectedLog(null)}>
        <DialogContent className="max-w-3xl">
          <DialogHeader>
            <DialogTitle>{t('operationLog.details', { defaultValue: '操作详情' })}</DialogTitle>
          </DialogHeader>
          <ScrollArea className="max-h-[80vh] w-full pr-4">
            <div className="space-y-6">
              <div className="grid gap-x-8 gap-y-4 text-sm md:grid-cols-2">
                <div className="flex flex-col gap-1">
                  <span className="text-muted-foreground font-medium">
                    {t('operationLog.column.operator', { defaultValue: '操作人' })}
                  </span>
                  <span>{selectedLog?.operator || '-'}</span>
                </div>

                <div className="flex flex-col gap-1">
                  <span className="text-muted-foreground font-medium">
                    {t('operationLog.column.createdAt', { defaultValue: '操作时间' })}
                  </span>
                  <span>
                    {selectedLog
                      ? format(new Date(selectedLog.created_at), 'yyyy-MM-dd HH:mm:ss')
                      : '-'}
                  </span>
                </div>

                <div className="flex flex-col gap-1">
                  <span className="text-muted-foreground font-medium">
                    {t('operationLog.column.type', { defaultValue: '操作类型' })}
                  </span>
                  <div className="flex">
                    <OperationTypeBadge
                      operationType={selectedLog?.operation_type ?? ''}
                      label={
                        selectedLog
                          ? (operationTypeLabelMap[selectedLog.operation_type] ??
                            selectedLog.operation_type)
                          : '-'
                      }
                    />
                  </div>
                </div>

                <div className="flex flex-col gap-1">
                  <span className="text-muted-foreground font-medium">
                    {t('operationLog.column.target', { defaultValue: '操作对象' })}
                  </span>
                  <span>{selectedLog?.target || '-'}</span>
                </div>

                <div className="flex flex-col gap-1">
                  <span className="text-muted-foreground font-medium">
                    {t('operationLog.column.status', { defaultValue: '状态' })}
                  </span>
                  <div className="flex">
                    <Badge variant={selectedLog?.status === 'Success' ? 'default' : 'destructive'}>
                      {selectedLog ? (statusLabels[selectedLog.status] ?? selectedLog.status) : ''}
                    </Badge>
                  </div>
                </div>
              </div>

              {selectedLog?.error_message && (
                <div className="bg-destructive/10 text-destructive rounded-md p-4 text-sm">
                  <p className="mb-1 font-semibold">
                    {t('operationLog.details.errorMessageTitle', { defaultValue: '错误信息' })}
                  </p>
                  <pre className="font-mono text-xs whitespace-pre-wrap">
                    {selectedLog.error_message}
                  </pre>
                </div>
              )}

              <div className="space-y-2">
                <p className="text-sm font-semibold">
                  {t('operationLog.details.sectionTitle', { defaultValue: '变更详情' })}
                </p>
                {selectedLog &&
                  (selectedLog.operation_type === 'UpdateVPA' ? (
                    <VpaResourceDiff details={selectedLog.details} />
                  ) : selectedLog.operation_type === 'DeleteJob' ? (
                    <DeleteJobDetailsView details={selectedLog.details} />
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
