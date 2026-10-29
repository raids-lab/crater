/**
 * Copyright 2025 RAIDS Lab
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
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ColumnDef, flexRender, getCoreRowModel, useReactTable } from '@tanstack/react-table'
import {
  AlertCircleIcon,
  ArrowLeft,
  CheckCircle2Icon,
  Copy,
  DownloadIcon,
  LayersIcon,
  Pause,
  PauseCircleIcon,
  Play,
  RotateCw,
  SearchIcon,
} from 'lucide-react'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
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

import ModelDownloadPhaseBadge from '@/components/badge/model-download-phase-badge'
import DocsButton from '@/components/button/docs-button'
import { TimeDistance } from '@/components/custom/time-distance'
import SimpleTooltip from '@/components/label/simple-tooltip'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import ModelDownloadProgress from '@/components/model/model-download-progress'
import ModelDownloadTokenDialog from '@/components/model/model-download-token-dialog'
import { DataTablePagination } from '@/components/query-table/pagination'
import { buildRemoteQueryKey, getLastPageIndex } from '@/components/query-table/remote-state'

import {
  ModelDownload,
  ModelDownloadStatus,
  apiGetModelDownloadSummary,
  apiListModelDownloadsPage,
  apiPauseModelDownload,
  apiResumeModelDownload,
  apiRetryModelDownload,
} from '@/services/api/modeldownload'
import type { IPage } from '@/services/types'

import useRemoteTableState from '@/hooks/use-remote-table-state'

import { logger } from '@/utils/loglevel'
import { showErrorToast } from '@/utils/toast'

import { cn } from '@/lib/utils'

// 有进行中的任务时快轮询,否则慢轮询
const ACTIVE_REFETCH_MS = 5000
const IDLE_REFETCH_MS = 30000

const sourceLabelMap: Record<string, string> = {
  modelscope: 'ModelScope',
  huggingface: 'HuggingFace',
}

function downloadRelationKey(download: ModelDownload) {
  if (download.relation === 'creator') return 'modelDownload.relation.creator'
  if (download.relation === 'submitted') return 'modelDownload.relation.submitted'
  if (download.status === 'Ready') return 'modelDownload.relation.publicReady'
  if (['Pending', 'Downloading', 'Paused'].includes(download.status)) {
    return 'modelDownload.relation.publicJoinable'
  }
  return null
}

export function ModelDownloadsPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const tableState = useRemoteTableState('model_downloads', {
    sorting: [{ id: 'updatedAt', desc: true }],
  })
  const [tokenTarget, setTokenTarget] = useState<{
    action: 'resume' | 'retry'
    download: ModelDownload
  } | null>(null)

  const getSingleFilter = useCallback(
    (id: string) => {
      const value = tableState.columnFilters.find((filter) => filter.id === id)?.value
      return Array.isArray(value) ? String(value[0] ?? 'all') : value ? String(value) : 'all'
    },
    [tableState.columnFilters]
  )
  const statusFilter = getSingleFilter('status') as ModelDownloadStatus | 'all'
  const categoryFilter = getSingleFilter('category') as 'all' | 'model' | 'dataset'
  const setSingleFilter = useCallback(
    (id: string, value: string) => {
      tableState.setColumnFilters((current) => {
        const remaining = current.filter((filter) => filter.id !== id)
        return value === 'all' ? remaining : [...remaining, { id, value: [value] }]
      })
    },
    [tableState]
  )

  const query = useQuery<IPage<ModelDownload>, Error>({
    queryKey: buildRemoteQueryKey('model-downloads', tableState.params),
    queryFn: ({ signal }) =>
      apiListModelDownloadsPage(tableState.params, signal).then((response) => response.data),
    placeholderData: keepPreviousData,
    refetchInterval: (q) => {
      const active =
        q.state.data?.items.some(
          (download) => download.status === 'Pending' || download.status === 'Downloading'
        ) ?? false
      return active ? ACTIVE_REFETCH_MS : IDLE_REFETCH_MS
    },
  })
  const summaryQuery = useQuery({
    queryKey: ['model-downloads-summary', categoryFilter],
    queryFn: ({ signal }) =>
      apiGetModelDownloadSummary(
        categoryFilter === 'all' ? undefined : categoryFilter,
        signal
      ).then((response) => response.data),
    refetchInterval: (summary) => {
      const counts = summary.state.data
      return (counts?.Pending ?? 0) + (counts?.Downloading ?? 0) > 0
        ? ACTIVE_REFETCH_MS
        : IDLE_REFETCH_MS
    },
  })

  const listData = query.data
  const summary = summaryQuery.data
  const total = listData?.total ?? 0
  const { pageIndex, pageSize } = tableState.pagination
  const setTablePagination = tableState.setPagination

  useEffect(() => {
    if (!query.data || query.isError || query.isFetching || query.isPlaceholderData) return
    const lastPageIndex = getLastPageIndex(query.data.total, pageSize)
    if (pageIndex > lastPageIndex) {
      setTablePagination((current) => ({ ...current, pageIndex: lastPageIndex }))
    }
  }, [
    pageIndex,
    pageSize,
    query.data,
    query.isError,
    query.isFetching,
    query.isPlaceholderData,
    setTablePagination,
  ])

  const refetchDownloads = async () => {
    try {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['remote-list', 'model-downloads'] }),
        queryClient.invalidateQueries({ queryKey: ['model-downloads-summary'] }),
      ])
    } catch (error) {
      logger.error('failed to refresh model download queries', error)
    }
  }

  const { mutate: pauseDownload, isPending: isPausing } = useMutation({
    mutationFn: apiPauseModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      toast.success(t('modelDownload.action.pauseSuccess'))
    },
    onError: showErrorToast,
  })

  const { mutate: resumeDownload, isPending: isResuming } = useMutation({
    mutationFn: apiResumeModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      setTokenTarget(null)
      toast.success(t('modelDownload.action.resumeSuccess'))
    },
    onError: showErrorToast,
  })

  const { mutate: retryDownload, isPending: isRetrying } = useMutation({
    mutationFn: apiRetryModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      setTokenTarget(null)
      toast.success(t('modelDownload.action.retrySuccess'))
    },
    onError: showErrorToast,
  })

  const columns = useMemo<ColumnDef<ModelDownload>[]>(
    () => [
      {
        accessorKey: 'name',
        header: t('modelDownload.list.name'),
        cell: ({ row }) => {
          const d = row.original
          const relationKey = downloadRelationKey(d)
          const unavailableRequesterCount = Math.max(0, d.requesterCount - d.requesters.length)
          return (
            <div className="flex max-w-[300px] flex-col gap-1">
              <Link
                to={
                  d.category === 'dataset'
                    ? '/portal/data/datasets/downloads/$id'
                    : '/portal/data/models/downloads/$id'
                }
                params={{ id: d.id.toString() }}
                className="hover:text-primary truncate text-sm font-medium transition-colors duration-200"
              >
                {d.name}
              </Link>
              <span className="text-muted-foreground truncate text-xs">
                {sourceLabelMap[d.source] ?? d.source}
                {d.revision ? ` · ${d.revision}` : ''}
              </span>
              {(relationKey || d.requesterCount > 0) && (
                <div className="flex items-center gap-1.5">
                  {relationKey && (
                    <Badge
                      variant={d.relation === 'none' ? 'outline' : 'secondary'}
                      className="h-5 px-1.5 text-[10px] font-normal"
                    >
                      {t(relationKey)}
                    </Badge>
                  )}
                  {d.requesterCount > 0 && (
                    <SimpleTooltip
                      tooltip={
                        <div className="max-h-48 max-w-72 space-y-1 overflow-y-auto py-1">
                          <p className="font-medium">
                            {t('modelDownload.relation.requesterListTitle')}
                          </p>
                          {d.requesters.map((requester) => (
                            <p key={requester.username}>
                              {requester.nickname && requester.nickname !== requester.username
                                ? `${requester.nickname} (@${requester.username})`
                                : `@${requester.username}`}
                            </p>
                          ))}
                          {unavailableRequesterCount > 0 && (
                            <p className="text-muted-foreground">
                              {t('modelDownload.relation.unavailableRequesterCount', {
                                count: unavailableRequesterCount,
                              })}
                            </p>
                          )}
                        </div>
                      }
                    >
                      <span className="text-muted-foreground cursor-help text-[10px] underline decoration-dotted underline-offset-2">
                        {t('modelDownload.relation.requesterCount', {
                          count: d.requesterCount,
                        })}
                      </span>
                    </SimpleTooltip>
                  )}
                </div>
              )}
            </div>
          )
        },
      },
      {
        accessorKey: 'category',
        header: t('modelDownload.list.category'),
        cell: ({ row }) => (
          <Badge variant="outline">{t(`modelDownload.category.${row.original.category}`)}</Badge>
        ),
      },
      {
        id: 'progress',
        header: t('modelDownload.list.progress'),
        cell: ({ row }) => (
          <ModelDownloadProgress download={row.original} className="max-w-[220px]" />
        ),
      },
      {
        accessorKey: 'status',
        header: t('modelDownload.list.status'),
        cell: ({ row }) => {
          const d = row.original
          return (
            <div className="flex flex-col items-start gap-1">
              <ModelDownloadPhaseBadge status={d.status} />
              {d.status === 'Failed' && d.message && (
                <SimpleTooltip tooltip={d.message}>
                  <span className="text-destructive/80 line-clamp-1 max-w-[220px] cursor-help text-xs">
                    {d.message}
                  </span>
                </SimpleTooltip>
              )}
            </div>
          )
        },
      },
      {
        accessorKey: 'userInfo',
        header: t('modelDownload.list.creator'),
        cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
      },
      {
        accessorKey: 'path',
        header: t('modelDownload.list.path'),
        cell: ({ row }) => {
          const copyPath = () => {
            navigator.clipboard.writeText(row.original.path)
            toast.success(t('modelDownload.pathCopied'))
          }

          return (
            <div className="flex items-center gap-1">
              <SimpleTooltip tooltip={row.original.path}>
                <div className="text-muted-foreground max-w-[200px] truncate font-mono text-xs">
                  {row.original.path}
                </div>
              </SimpleTooltip>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7 shrink-0"
                onClick={copyPath}
                title={t('modelDownload.copyPath')}
              >
                <Copy className="h-3.5 w-3.5" />
              </Button>
            </div>
          )
        },
      },
      {
        accessorKey: 'updatedAt',
        header: t('modelDownload.list.updatedAt'),
        cell: ({ row }) => <TimeDistance date={row.original.updatedAt} />,
      },
      {
        id: 'actions',
        header: '',
        cell: ({ row }) => {
          const download = row.original
          if (!download.canManage) {
            return (
              <SimpleTooltip tooltip={t('modelDownload.action.onlyManagers')}>
                <span className="text-muted-foreground/50 text-xs">--</span>
              </SimpleTooltip>
            )
          }
          return (
            <div className="flex flex-row space-x-1">
              {download.status === 'Downloading' && (
                <SimpleTooltip tooltip={t('modelDownload.action.pause')}>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    disabled={isPausing}
                    onClick={() => pauseDownload(download.id)}
                  >
                    <Pause className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
              {download.status === 'Paused' && (
                <SimpleTooltip tooltip={t('modelDownload.action.resume.confirm')}>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    disabled={isResuming}
                    onClick={() => setTokenTarget({ action: 'resume', download })}
                  >
                    <Play className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
              {download.status === 'Failed' && (
                <SimpleTooltip tooltip={t('modelDownload.action.retry.confirm')}>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    disabled={isRetrying}
                    onClick={() => setTokenTarget({ action: 'retry', download })}
                  >
                    <RotateCw className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
            </div>
          )
        },
      },
    ],
    [isPausing, isResuming, isRetrying, pauseDownload, t]
  )

  const defaultData = useMemo<ModelDownload[]>(() => [], [])

  const table = useReactTable({
    data: listData?.items ?? defaultData,
    columns,
    pageCount: total > 0 ? Math.ceil(total / tableState.pagination.pageSize) : 0,
    state: { pagination: tableState.pagination },
    onPaginationChange: tableState.setPagination,
    getCoreRowModel: getCoreRowModel(),
    manualPagination: true,
  })

  const handleStatusChange = (value: string) => {
    setSingleFilter('status', value)
  }

  const summaryTotal = summary
    ? Object.values(summary).reduce((acc, count) => acc + (count ?? 0), 0)
    : 0

  const statCards = [
    {
      key: 'all' as const,
      label: t('modelDownload.stats.all'),
      value: summaryTotal,
      icon: LayersIcon,
      iconClass: 'text-primary',
      hint: undefined as string | undefined,
    },
    {
      key: 'Downloading' as const,
      label: t('modelDownload.stats.downloading'),
      value: summary?.Downloading ?? 0,
      icon: DownloadIcon,
      iconClass: 'text-highlight-sky',
      hint:
        (summary?.Pending ?? 0) > 0
          ? t('modelDownload.stats.pendingHint', { count: summary?.Pending })
          : undefined,
    },
    {
      key: 'Ready' as const,
      label: t('modelDownload.stats.completed'),
      value: summary?.Ready ?? 0,
      icon: CheckCircle2Icon,
      iconClass: 'text-highlight-emerald',
      hint: undefined,
    },
    {
      key: 'Paused' as const,
      label: t('modelDownload.stats.paused'),
      value: summary?.Paused ?? 0,
      icon: PauseCircleIcon,
      iconClass: 'text-amber-600',
      hint: undefined,
    },
    {
      key: 'Failed' as const,
      label: t('modelDownload.stats.failed'),
      value: summary?.Failed ?? 0,
      icon: AlertCircleIcon,
      iconClass: 'text-highlight-red',
      hint: undefined,
    },
  ]

  const lastUpdatedAt = useMemo(() => {
    if (!query.dataUpdatedAt) {
      return '--'
    }
    return new Date(query.dataUpdatedAt).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }, [query.dataUpdatedAt])

  return (
    <div className="space-y-4">
      <PageTitle
        title={t('modelDownload.list.title')}
        description={t('modelDownload.list.description')}
      >
        <div className="flex flex-row gap-3">
          <Link to="/portal/data/models" search={{ organization: undefined }}>
            <Button variant="outline" size="sm">
              <ArrowLeft className="mr-2 h-4 w-4" />
              {t('modelDownload.list.back')}
            </Button>
          </Link>
          <DocsButton title={t('modelDownload.list.docs')} url="file/model" />
        </div>
      </PageTitle>

      {/* 状态统计卡片,点击可按状态筛选 */}
      <div className="grid grid-cols-2 gap-3 md:grid-cols-5">
        {statCards.map((card) => {
          const selected = statusFilter === card.key
          return (
            <Card
              key={card.key}
              role="button"
              tabIndex={0}
              onClick={() => handleStatusChange(selected ? 'all' : card.key)}
              className={cn(
                'cursor-pointer py-0 transition-all hover:shadow-md',
                selected && 'ring-primary/60 ring-2'
              )}
            >
              <CardContent className="flex items-center justify-between p-4">
                <div className="min-w-0">
                  <p className="text-muted-foreground text-xs">{card.label}</p>
                  <p className="text-2xl font-semibold tabular-nums">{card.value}</p>
                  {card.hint && (
                    <p className="text-muted-foreground truncate text-xs">{card.hint}</p>
                  )}
                </div>
                <card.icon className={cn('size-5 shrink-0', card.iconClass)} />
              </CardContent>
            </Card>
          )
        })}
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative">
          <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
          <Input
            placeholder={t('modelDownload.list.searchPlaceholder')}
            className="h-9 w-full pl-8 sm:w-[250px]"
            value={tableState.search}
            onChange={(e) => tableState.setSearch(e.target.value)}
          />
        </div>
        <Select
          value={categoryFilter}
          onValueChange={(value) => {
            setSingleFilter('category', value)
          }}
        >
          <SelectTrigger className="h-9 w-full sm:w-[140px]">
            <SelectValue placeholder={t('modelDownload.list.category')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('modelDownload.category.all')}</SelectItem>
            <SelectItem value="model">{t('modelDownload.category.model')}</SelectItem>
            <SelectItem value="dataset">{t('modelDownload.category.dataset')}</SelectItem>
          </SelectContent>
        </Select>
        <Select value={statusFilter} onValueChange={handleStatusChange}>
          <SelectTrigger className="h-9 w-full sm:w-[140px]">
            <SelectValue placeholder={t('modelDownload.list.status')} />
          </SelectTrigger>
          <SelectContent>
            <SelectItem value="all">{t('modelDownload.status.all')}</SelectItem>
            <SelectItem value="Pending">{t('modelDownload.status.pending')}</SelectItem>
            <SelectItem value="Downloading">{t('modelDownload.status.downloading')}</SelectItem>
            <SelectItem value="Paused">{t('modelDownload.status.paused')}</SelectItem>
            <SelectItem value="Ready">{t('modelDownload.status.completed')}</SelectItem>
            <SelectItem value="Failed">{t('modelDownload.status.failed')}</SelectItem>
          </SelectContent>
        </Select>
      </div>

      {/* 数据表 */}
      <div className="rounded-md border">
        <Table>
          <TableHeader>
            {table.getHeaderGroups().map((headerGroup) => (
              <TableRow key={headerGroup.id}>
                {headerGroup.headers.map((header) => (
                  <TableHead key={header.id}>
                    {header.isPlaceholder
                      ? null
                      : flexRender(header.column.columnDef.header, header.getContext())}
                  </TableHead>
                ))}
              </TableRow>
            ))}
          </TableHeader>
          <TableBody>
            {query.isError ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  <span className="text-destructive text-sm">
                    {t('modelDownload.list.loadError')}
                  </span>
                </TableCell>
              </TableRow>
            ) : query.isLoading ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  <span className="text-muted-foreground text-sm">{t('common.loading')}</span>
                </TableCell>
              </TableRow>
            ) : table.getRowModel().rows.length === 0 ? (
              <TableRow>
                <TableCell colSpan={columns.length} className="h-24 text-center">
                  <span className="text-muted-foreground text-sm">
                    {t('modelDownload.list.empty')}
                  </span>
                </TableCell>
              </TableRow>
            ) : (
              table.getRowModel().rows.map((row) => (
                <TableRow key={row.id}>
                  {row.getVisibleCells().map((cell) => (
                    <TableCell key={cell.id}>
                      {flexRender(cell.column.columnDef.cell, cell.getContext())}
                    </TableCell>
                  ))}
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </div>

      <DataTablePagination
        table={table}
        updatedAt={lastUpdatedAt}
        refetch={() => void query.refetch()}
        pageSizeOptions={[10, 20, 50, 100]}
        totalItems={total}
      />
      {tokenTarget && (
        <ModelDownloadTokenDialog
          action={tokenTarget.action}
          downloadName={tokenTarget.download.name}
          initialRevision={tokenTarget.download.revision}
          source={tokenTarget.download.source}
          isPending={isResuming || isRetrying}
          open
          onOpenChange={(open) => !open && setTokenTarget(null)}
          onSubmit={(token, revision) => {
            const request = { id: tokenTarget.download.id, token, revision }
            if (tokenTarget.action === 'resume') {
              resumeDownload(request)
            } else {
              retryDownload(request)
            }
          }}
        />
      )}
    </div>
  )
}
