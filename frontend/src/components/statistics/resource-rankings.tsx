import { keepPreviousData, useQuery } from '@tanstack/react-query'
import { ColumnDef, PaginationState, getCoreRowModel, useReactTable } from '@tanstack/react-table'
import { InfoIcon } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import SimpleTooltip from '@/components/label/simple-tooltip'
import UserLabel from '@/components/label/user-label'
import { DataTableShell } from '@/components/query-table/shell'
import { type DataTableToolbarConfig } from '@/components/query-table/toolbar'
import {
  allGPUResource,
  statisticsResourceLabel,
  statisticsResourceUnit,
  utilizationTone,
} from '@/components/statistics/statistics-display'

import {
  IStatisticsRankingReq,
  IStatisticsRankingRow,
  IStatisticsReq,
  IStatisticsResp,
  apiAdminGetStatisticsRankings,
  apiGetStatisticsRankings,
} from '@/services/api/statistics'

import useIsAdmin from '@/hooks/use-admin'

import { cn } from '@/lib/utils'

interface Props {
  request: IStatisticsReq
  data?: IStatisticsResp
  enabled: boolean
}

// 后端 pageSize 上限为 100
const pageSizeOptions = [10, 20, 50, 100]

// ResourceRankings 以服务端分页表格展示资源用量和利用率排名。
export function ResourceRankings({ request, data, enabled }: Props) {
  const { t } = useTranslation()
  const isAdmin = useIsAdmin()
  const [dimension, setDimension] = useState<'user' | 'account'>('user')
  const [metric, setMetric] = useState<IStatisticsRankingReq['metric']>('usage')
  const [order, setOrder] = useState<'asc' | 'desc'>('desc')
  const [resource, setResource] = useState('cpu')
  const [pagination, setPagination] = useState<PaginationState>({ pageIndex: 0, pageSize: 20 })
  const params: IStatisticsRankingReq = {
    ...request,
    dimension,
    metric,
    order,
    page: pagination.pageIndex + 1,
    pageSize: pagination.pageSize,
    ...(resource === allGPUResource ? { resourceType: 'gpu' } : { resourceName: resource }),
  }
  const query = useQuery({
    queryKey: ['statistics-rankings', isAdmin, params],
    queryFn: () =>
      isAdmin ? apiAdminGetStatisticsRankings(params) : apiGetStatisticsRankings(params),
    select: (res) => res.data,
    enabled,
    placeholderData: keepPreviousData,
    refetchInterval: 60000,
  })
  const total = query.data?.total ?? 0
  const resourceIDs = new Set([
    'cpu',
    'memory',
    ...Object.keys(data?.totalUsage ?? {}),
    allGPUResource,
  ])
  const resources: [string, string][] = Array.from(resourceIDs, (id) => [
    id,
    statisticsResourceLabel(t, id, data?.totalUsage[id]?.label),
  ])
  const unit = statisticsResourceUnit(t, resource, data?.totalUsage[resource]?.type)

  const columns = useMemo<ColumnDef<IStatisticsRankingRow>[]>(() => {
    const number = (v: number) => v.toLocaleString(undefined, { maximumFractionDigits: 2 })
    const percent = (v: number | null) =>
      v === null ? (
        <span className="text-muted-foreground">{t('statistics.notEvaluated')}</span>
      ) : (
        `${number(v * 100)}%`
      )
    return [
      {
        id: 'rank',
        header: t('statistics.rank'),
        enableHiding: false,
        cell: ({ row }) => (
          // 前三名轻量高亮
          <span
            className={cn(
              'inline-flex size-6 items-center justify-center rounded-full text-xs tabular-nums',
              row.original.rank <= 3
                ? 'bg-primary/10 text-primary font-semibold'
                : 'text-muted-foreground'
            )}
          >
            {row.original.rank}
          </span>
        ),
      },
      {
        id: 'name',
        header: t(dimension === 'user' ? 'statistics.users' : 'statistics.accounts'),
        enableHiding: false,
        cell: ({ row: { original } }) =>
          dimension === 'user' && original.name ? (
            <UserLabel info={{ username: original.name, nickname: original.nickname }} />
          ) : (
            <div className="flex min-w-0 flex-col">
              <span className="truncate text-sm">
                {original.nickname || original.name || `#${original.id}`}
              </span>
              {original.nickname && original.name && original.nickname !== original.name && (
                <span className="text-muted-foreground truncate font-mono text-xs">
                  {original.name}
                </span>
              )}
            </div>
          ),
      },
      {
        accessorKey: 'usage',
        header: `${t('statistics.allocated')} (${unit})`,
        cell: ({ row }) => <span className="tabular-nums">{number(row.original.usage)}</span>,
      },
      {
        accessorKey: 'effectiveUsage',
        header: t('statistics.effective'),
        cell: ({ row }) =>
          row.original.profiledUsage > 0 ? (
            <span className="tabular-nums">{number(row.original.effectiveUsage)}</span>
          ) : (
            <span className="text-muted-foreground">{t('statistics.notEvaluated')}</span>
          ),
      },
      {
        accessorKey: 'utilization',
        header: t('statistics.utilization'),
        cell: ({ row }) => (
          <span
            className={cn(
              'tabular-nums',
              row.original.utilization !== null && utilizationTone(row.original.utilization).text
            )}
          >
            {percent(row.original.utilization)}
          </span>
        ),
      },
      {
        accessorKey: 'coverage',
        header: t('statistics.coverage'),
        cell: ({ row }) => <span className="tabular-nums">{percent(row.original.coverage)}</span>,
      },
    ]
  }, [t, dimension, unit])

  const table = useReactTable({
    data: query.data?.rows ?? [],
    columns,
    getRowId: (row) => String(row.id),
    pageCount: Math.ceil(total / pagination.pageSize),
    state: { pagination },
    manualPagination: true,
    onPaginationChange: setPagination,
    getCoreRowModel: getCoreRowModel(),
  })

  const selector = (
    label: string,
    value: string,
    options: [string, string][],
    change: (value: string) => void
  ) => (
    <Select
      value={value}
      onValueChange={(v) => {
        change(v)
        setPagination((p) => ({ ...p, pageIndex: 0 }))
      }}
    >
      <SelectTrigger aria-label={label} className="bg-background h-9 w-full min-w-0 sm:w-[150px]">
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        {options.map(([key, text]) => (
          <SelectItem key={key} value={key}>
            {text}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
  const toolbarConfig: DataTableToolbarConfig = {
    filterOptions: [],
    getHeader: (key) => {
      const header = columns.find(
        (column) => column.id === key || ('accessorKey' in column && column.accessorKey === key)
      )?.header
      return typeof header === 'string' ? header : key
    },
    globalSearch: { enabled: false },
  }

  return (
    <DataTableShell
      table={table}
      toolbarConfig={toolbarConfig}
      withI18n
      className="min-w-0"
      isLoading={query.isLoading || !enabled}
      isError={query.isError}
      errorMessage={query.isError ? t('statistics.loadError') : undefined}
      refetch={() => void query.refetch()}
      dataUpdatedAt={query.dataUpdatedAt}
      totalItems={total}
      pageSizeOptions={pageSizeOptions}
    >
      {request.scope === 'cluster' &&
        selector(
          t('statistics.dimension'),
          dimension,
          [
            ['user', t('statistics.users')],
            ['account', t('statistics.accounts')],
          ],
          (v) => setDimension(v as typeof dimension)
        )}
      {selector(t('statistics.resource'), resource, resources, setResource)}
      {selector(
        t('statistics.metric'),
        metric,
        [
          ['usage', t('statistics.allocated')],
          ['effectiveUsage', t('statistics.effective')],
          ['utilization', t('statistics.utilization')],
        ],
        (v) => setMetric(v as typeof metric)
      )}
      {selector(
        t('statistics.order'),
        order,
        [
          ['desc', t('statistics.desc')],
          ['asc', t('statistics.asc')],
        ],
        (v) => setOrder(v as typeof order)
      )}
      <SimpleTooltip
        ariaLabel={t('statistics.rankingNote')}
        tooltip={<p className="max-w-xs">{t('statistics.rankingNote')}</p>}
      >
        <InfoIcon className="text-muted-foreground size-4" />
      </SimpleTooltip>
    </DataTableShell>
  )
}
