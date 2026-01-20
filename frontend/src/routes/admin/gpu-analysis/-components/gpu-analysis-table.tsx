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
// i18n-processed-v1.1.0
// This is the implementation for the GPU Analysis Overview page.
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import {
  AlertTriangleIcon,
  BookTextIcon,
  CheckCircle2Icon,
  CircleIcon,
  CopyIcon,
  EllipsisVerticalIcon,
  MegaphoneIcon,
  PlayCircleIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XCircleIcon,
} from 'lucide-react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import JobTypeLabel from '@/components/badge/job-type-badge'
import NodeBadges from '@/components/badge/node-badges'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import { MarkdownRenderer } from '@/components/form/markdown-renderer'
import { JobNameCell } from '@/components/label/job-name-label'
import UserLabel from '@/components/label/user-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'
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
} from '@/components/ui-custom/alert-dialog'

import {
  IGpuAnalysis,
  ReviewStatus,
  apiAdminConfirmAndStopJob,
  apiAdminListGpuAnalyses,
  apiAdminTriggerAllJobsAnalysis,
  apiAdminUpdateGpuAnalysisReviewStatus,
} from '@/services/api/gpu-analysis'
import { IJobInfo, JobType } from '@/services/api/vcjob'

import { logger } from '@/utils/loglevel'
import { showErrorToast } from '@/utils/toast'

import { cn } from '@/lib/utils'

/**
 * A badge component to display the review status with appropriate colors and icons.
 * Colors updated to match AdminJobOverview palette (Emerald/Purple/Gray).
 */
const ReviewStatusBadge = ({ status }: { status: ReviewStatus }) => {
  const { t } = useTranslation()

  const statusConfig = {
    [ReviewStatus.Pending]: {
      label: t('gpuAnalysis.status.pending'),
      icon: <CircleIcon className="mr-1 size-3" />,
      className: 'text-highlight-gray bg-highlight-gray/20',
    },
    [ReviewStatus.Confirmed]: {
      label: t('gpuAnalysis.status.confirmed'),
      icon: <CheckCircle2Icon className="mr-1 size-3" />,
      className: 'text-highlight-purple bg-highlight-purple/20', // Changed from Sky to Purple to match "Unlock" style
    },
    [ReviewStatus.Ignored]: {
      label: t('gpuAnalysis.status.ignored'),
      icon: <XCircleIcon className="mr-1 size-3" />,
      className: 'text-highlight-emerald bg-highlight-emerald/20', // Changed from Green to Emerald
    },
  }[status]

  return (
    <Badge variant="default" className={cn('hover:opacity-90', statusConfig.className)}>
      {statusConfig.icon}
      {statusConfig.label}
    </Badge>
  )
}

/**
 * Replaces the two separate score columns with a single, intuitive risk level badge.
 * Colors updated to match AdminJobOverview palette (Red/Orange/Emerald).
 */
const RiskLevelBadge = ({
  phase1Score,
  phase2Score,
}: {
  phase1Score: number
  phase2Score: number
}) => {
  const { t } = useTranslation()

  const riskConfig = useMemo(() => {
    if (phase2Score >= 7) {
      return {
        label: t('gpuAnalysis.riskLevel.high'),
        className: 'text-highlight-red bg-highlight-red/20', // Kept Red/Destructive family
        icon: <AlertTriangleIcon className="mr-1.5 size-3.5" />,
      }
    }
    if (phase2Score > 3) {
      return {
        label: t('gpuAnalysis.riskLevel.medium'),
        className: 'text-highlight-orange bg-highlight-orange/20', // Changed from Amber to Orange
        icon: <AlertTriangleIcon className="mr-1.5 size-3.5" />,
      }
    }
    return {
      label: t('gpuAnalysis.riskLevel.low'),
      className: 'text-highlight-emerald bg-highlight-emerald/20', // Changed from Green to Emerald
      icon: <ShieldCheckIcon className="mr-1.5 size-3.5" />,
    }
  }, [phase2Score, t])

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger>
          <Badge
            variant="default"
            className={cn('flex items-center font-semibold hover:opacity-90', riskConfig.className)}
          >
            {riskConfig.icon}
            {riskConfig.label}
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p>
            {t('gpuAnalysis.headers.Phase1Score')}: {phase1Score}
          </p>
          <p>
            {t('gpuAnalysis.headers.Phase2Score')}: {phase2Score}
          </p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

/**
 * A hover card to display the full details of a GPU analysis record.
 */
const DetailsHoverCard = ({
  children,
  record,
}: {
  children: React.ReactNode
  record: IGpuAnalysis
}) => {
  const { t } = useTranslation()

  const historicalMetrics = useMemo(() => {
    try {
      return JSON.parse(record.HistoricalMetrics)
    } catch (error) {
      logger.error('Failed to parse HistoricalMetrics JSON:', error)
      return null
    }
  }, [record.HistoricalMetrics])

  const metricDisplayOrder = [
    { key: 'gpuUtilAvg', label: t('gpuAnalysis.metrics.gpuUtilAvg') },
    { key: 'gpuUtilStdDev', label: t('gpuAnalysis.metrics.gpuUtilStdDev') },
    { key: 'gpuMemUsedAvg', label: t('gpuAnalysis.metrics.gpuMemUsedAvg') },
    { key: 'gpuMemUsedStdDev', label: t('gpuAnalysis.metrics.gpuMemUsedStdDev') },
  ]

  return (
    <HoverCard openDelay={200} closeDelay={100}>
      <HoverCardTrigger asChild>{children}</HoverCardTrigger>
      <HoverCardContent className="w-[500px]" side="top" align="start">
        <div className="grid max-h-[55vh] w-full gap-4 overflow-x-hidden overflow-y-auto p-1">
          <div className="space-y-2">
            <h4 className="text-sm font-medium">{t('gpuAnalysis.detailsDialog.commandTitle')}</h4>
            <pre className="bg-muted text-muted-foreground rounded-md border p-3 font-mono text-xs break-all whitespace-pre-wrap">
              <code>{record.Command}</code>
            </pre>
          </div>
          {historicalMetrics && (
            <div className="space-y-2">
              <h4 className="text-sm font-medium">
                {t('gpuAnalysis.detailsDialog.historicalMetricsTitle')}
              </h4>
              <div className="bg-muted space-y-1.5 rounded-md border p-3 text-xs">
                {metricDisplayOrder.map(({ key, label }) => (
                  <div key={key} className="flex items-center justify-between">
                    <span className="text-muted-foreground">{label}</span>
                    <span className="font-mono font-semibold">
                      {historicalMetrics[key] ?? 'N/A'}
                    </span>
                  </div>
                ))}
              </div>
            </div>
          )}
          <div className="space-y-2">
            <h4 className="text-sm font-medium">{t('gpuAnalysis.detailsDialog.llmReasonTitle')}</h4>
            <div className="prose prose-sm dark:prose-invert bg-muted max-w-none rounded-md border p-3 text-xs leading-relaxed break-words [&_code]:break-all [&_pre]:break-all [&_pre]:whitespace-pre-wrap">
              <MarkdownRenderer>
                {record.Phase2LLMReason || record.Phase1LLMReason}
              </MarkdownRenderer>
            </div>
          </div>
        </div>
      </HoverCardContent>
    </HoverCard>
  )
}

// Main Component
const GpuAnalysisOverview = () => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const toolbarConfig: DataTableToolbarConfig = useMemo(
    () => ({
      globalSearch: {
        enabled: true,
        placeholder: t('gpuAnalysis.toolbar.searchPlaceholder'),
      },
      filterOptions: [
        {
          key: 'ReviewStatus',
          title: t('gpuAnalysis.toolbar.filters.status'),
          option: Object.values(ReviewStatus)
            .filter((value) => typeof value === 'number')
            .map((value) => ({
              label: t(`gpuAnalysis.status.${ReviewStatus[value as number].toLowerCase()}`),
              value: value as unknown as string, // 保持为数字，匹配原始数据类型
            })),
        },
        {
          key: 'Phase2Score',
          title: t('gpuAnalysis.headers.riskLevel'),
          // 这里的 value 必须能被 filterFn 识别
          option: [
            { label: t('gpuAnalysis.riskLevel.high'), value: 'high' },
            { label: t('gpuAnalysis.riskLevel.medium'), value: 'medium' },
            { label: t('gpuAnalysis.riskLevel.low'), value: 'low' },
          ],
        },
      ],
      getHeader: (key: string) => t(`gpuAnalysis.headers.${key}`),
    }),
    [t]
  )

  const analysisQuery = useQuery({
    queryKey: ['admin', 'gpu-analysis'],
    queryFn: apiAdminListGpuAnalyses,
    select: (res) => res.data,
  })

  const copyHoggingMessage = useCallback(
    async (message: string) => {
      try {
        await navigator.clipboard.writeText(message)
        toast.success(t('gpuAnalysis.toast.copySuccess'), {
          icon: <CopyIcon className="size-4" />,
        })
      } catch (err) {
        logger.error('Failed to copy text: ', err)
        toast.error(t('gpuAnalysis.toast.copyFailed'))
      }
    },
    [t]
  ) // 添加 t 作为依赖

  const { mutateAsync: confirmAndStopAsync } = useMutation({
    mutationFn: (id: number) => apiAdminConfirmAndStopJob(id),
    onError: (error) => {
      showErrorToast(error)
    },
  })

  const { mutate: updateReviewStatus } = useMutation({
    mutationFn: ({ id, status }: { id: number; status: ReviewStatus }) =>
      apiAdminUpdateGpuAnalysisReviewStatus(id, status),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['admin', 'gpu-analysis'] })
      toast.success(t('gpuAnalysis.toast.updateSuccess'))
    },
    onError: (error) => {
      showErrorToast(error)
    },
  })

  const { mutate: triggerAnalysis, isPending: isTriggering } = useMutation({
    mutationFn: apiAdminTriggerAllJobsAnalysis,
    onSuccess: (data) => {
      toast.success(t('gpuAnalysis.toast.triggerSuccessTitle'), {
        description: t('gpuAnalysis.toast.triggerSuccessDesc', {
          count: data.data.queuedJobs,
        }),
      })
    },
    onError: (error) => {
      showErrorToast(error)
    },
  })

  const columns = useMemo<ColumnDef<IGpuAnalysis>[]>(
    () => [
      {
        accessorKey: 'JobName',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.JobName')} />
        ),
        cell: ({ row }) => {
          const originalData = row.original
          const jobInfoForCell = {
            name: originalData.Name,
            jobName: originalData.JobName,
            status: originalData.status,
            locked:
              !!originalData.lockedTimestamp && new Date(originalData.lockedTimestamp) > new Date(),
            lockedTimestamp: originalData.lockedTimestamp,
            userInfo: {
              username: originalData.UserName,
              nickname: originalData.UserNickname,
            },
          }
          return <JobNameCell jobInfo={jobInfoForCell as unknown as IJobInfo} />
        },
      },
      {
        accessorKey: 'UserName',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.UserName')} />
        ),
        cell: ({ row }) => (
          <UserLabel
            info={{ username: row.original.UserName, nickname: row.original.UserNickname }}
          />
        ),
      },
      {
        accessorKey: 'JobType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.JobType')} />
        ),
        cell: ({ row }) => <JobTypeLabel jobType={row.original.JobType as JobType} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'Nodes',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.Nodes')} />
        ),
        cell: ({ row }) => {
          const nodes = Array.isArray(row.original.Nodes) ? row.original.Nodes : []
          return <NodeBadges nodes={nodes} />
        },
      },
      {
        accessorKey: 'Resources',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.Resources')} />
        ),
        cell: ({ row }) => {
          const resources = row.original.Resources
          return <ResourceBadges resources={resources} />
        },
      },
      {
        accessorKey: 'ReviewStatus',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.ReviewStatus')} />
        ),
        cell: ({ row }) => (
          <DetailsHoverCard record={row.original}>
            <div className="cursor-pointer">
              <ReviewStatusBadge status={row.original.ReviewStatus} />
            </div>
          </DetailsHoverCard>
        ),
        filterFn: (row, id, value: string[]) => {
          return value.includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'Phase2Score',
        accessorFn: (row) => {
          const score = row.Phase2Score
          if (score >= 7) return 'high'
          if (score > 3) return 'medium'
          return 'low'
        },
        header: ({ column }) => (
          <DataTableColumnHeader
            column={column}
            title={t('gpuAnalysis.headers.riskLevel')}
            className="justify-center"
          />
        ),
        cell: ({ row }) => (
          <div className="flex justify-center">
            <RiskLevelBadge
              phase1Score={row.original.Phase1Score}
              phase2Score={row.original.Phase2Score}
            />
          </div>
        ),
        // 添加以下 filterFn 逻辑
        filterFn: (row, id, value: string[]) => {
          return value.includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'CreatedAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('gpuAnalysis.headers.CreatedAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.getValue('CreatedAt')} />,
        sortingFn: 'datetime',
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const record = row.original
          const isActionable = record.ReviewStatus === ReviewStatus.Pending

          return (
            <div className="flex justify-end">
              <AlertDialog>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" className="h-8 w-8 p-0">
                      <span className="sr-only">{t('common.openMenu')}</span>
                      <EllipsisVerticalIcon className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end">
                    <DropdownMenuLabel className="text-muted-foreground text-xs">
                      {t('common.actions')}
                    </DropdownMenuLabel>
                    {isActionable && (
                      <>
                        {/* 1. 确认违规：移除菜单项颜色，仅图标加 text-highlight-orange */}
                        <DropdownMenuItem
                          onClick={() =>
                            updateReviewStatus(
                              {
                                id: record.ID,
                                status: ReviewStatus.Confirmed,
                              },
                              {
                                onSuccess: () => {
                                  const message = t('gpuAnalysis.copy.template', {
                                    jobList: `- ${record.JobName}`,
                                  })
                                  copyHoggingMessage(message)
                                },
                              }
                            )
                          }
                        >
                          <MegaphoneIcon className="text-highlight-orange mr-2 size-4" />
                          {t('gpuAnalysis.actions.confirm')}
                        </DropdownMenuItem>

                        {/* 2. 忽略：移除菜单项颜色，图标加 text-highlight-purple (参考第一段代码的 Lock 颜色) */}
                        <DropdownMenuItem
                          onClick={() =>
                            updateReviewStatus({
                              id: record.ID,
                              status: ReviewStatus.Ignored,
                            })
                          }
                        >
                          <XCircleIcon className="text-highlight-purple mr-2 size-4" />
                          {t('gpuAnalysis.actions.ignore')}
                        </DropdownMenuItem>

                        {/* 3. 违规停用：移除菜单项颜色，图标加 text-destructive (参考第一段代码的 Trash2 颜色) */}
                        <AlertDialogTrigger asChild>
                          <DropdownMenuItem>
                            <Trash2Icon className="text-destructive mr-2 size-4" />
                            {t('gpuAnalysis.actions.confirmAndStop')}
                          </DropdownMenuItem>
                        </AlertDialogTrigger>
                      </>
                    )}
                    {!isActionable && (
                      <DropdownMenuItem disabled>
                        <BookTextIcon className="mr-2 size-4" />
                        {t('gpuAnalysis.actions.noActionAvailable')}
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>

                {/* AlertDialogContent 部分保持不变 */}
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('gpuAnalysis.dialog.confirmAndStop.title') || 'Confirm and Stop Job'}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('gpuAnalysis.dialog.confirmAndStop.description', {
                        name: record.JobName,
                      }) ||
                        `Are you sure you want to stop job ${record.Name} and mark it as confirmed violation?`}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      onClick={async () => {
                        try {
                          await confirmAndStopAsync(record.ID)
                          await queryClient.invalidateQueries({
                            queryKey: ['admin', 'gpu-analysis'],
                          })
                          toast.success(t('gpuAnalysis.toast.confirmAndStopSuccess'))
                          const message = t('gpuAnalysis.copy.template', {
                            jobList: `- ${record.JobName} (Stopped)`,
                          })
                          copyHoggingMessage(message)
                        } catch {
                          // handled in mutation
                        }
                      }}
                    >
                      {t('gpuAnalysis.actions.confirmAndStop')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          )
        },
      },
    ],
    [t, updateReviewStatus, confirmAndStopAsync, queryClient, copyHoggingMessage]
  )

  return (
    <div className="space-y-4">
      <DataTable
        storageKey="admin_gpu_analysis"
        query={analysisQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        info={{
          title: t('gpuAnalysis.title'),
          description: t('gpuAnalysis.description'),
        }}
        multipleHandlers={[
          // 1. Bulk Stop
          {
            title: (rows) => t('gpuAnalysis.handlers.confirmAndStopTitle', { count: rows.length }),
            description: (rows) =>
              t('gpuAnalysis.handlers.confirmAndStopDescription', {
                jobs: rows.map((row) => row.original.Name).join(', '),
              }),
            icon: <Trash2Icon className="text-destructive size-5" />,
            isDanger: true,
            handleSubmit: (rows) => {
              const promises = rows.map((row) => confirmAndStopAsync(row.original.ID))
              Promise.all(promises)
                .then(() => {
                  queryClient.invalidateQueries({ queryKey: ['admin', 'gpu-analysis'] })
                  toast.success(
                    t('gpuAnalysis.handlers.confirmAndStopSuccess', { count: rows.length })
                  )

                  // Bulk Copy
                  const jobList = rows
                    .map((row) => `- ${row.original.JobName} (Stopped)`)
                    .join('\n')
                  const message = t('gpuAnalysis.copy.template', {
                    jobList: jobList,
                  })
                  copyHoggingMessage(message)
                })
                .catch((error) => {
                  showErrorToast(error)
                })
            },
          },
          // 2. Bulk Confirm - Changed color to highlight-orange to match "SquareIcon" style in File 1
          {
            title: (rows) => t('gpuAnalysis.handlers.confirmTitle', { count: rows.length }),
            description: (rows) =>
              t('gpuAnalysis.handlers.confirmDescription', {
                jobs: rows.map((row) => row.original.JobName).join(', '),
              }),
            icon: <MegaphoneIcon className="text-highlight-orange size-5" />,
            handleSubmit: (rows) => {
              const promises = rows.map((row) =>
                apiAdminUpdateGpuAnalysisReviewStatus(row.original.ID, ReviewStatus.Confirmed)
              )
              Promise.all(promises)
                .then(() => {
                  queryClient.invalidateQueries({ queryKey: ['admin', 'gpu-analysis'] })
                  toast.success(t('gpuAnalysis.handlers.confirmSuccess', { count: rows.length }))
                  // Bulk Copy
                  const jobList = rows.map((row) => `- ${row.original.JobName}`).join('\n')
                  const message = t('gpuAnalysis.copy.template', {
                    jobList: jobList,
                  })
                  copyHoggingMessage(message)
                })
                .catch((error) => {
                  showErrorToast(error)
                })
            },
          },
          // 3. Bulk Ignore
          {
            title: (rows) => t('gpuAnalysis.handlers.ignoreTitle', { count: rows.length }),
            description: (rows) =>
              t('gpuAnalysis.handlers.ignoreDescription', {
                jobs: rows.map((row) => row.original.JobName).join(', '),
              }),
            icon: <XCircleIcon className="text-muted-foreground size-5" />,
            handleSubmit: (rows) => {
              const promises = rows.map((row) =>
                apiAdminUpdateGpuAnalysisReviewStatus(row.original.ID, ReviewStatus.Ignored)
              )
              Promise.all(promises)
                .then(() => {
                  queryClient.invalidateQueries({ queryKey: ['admin', 'gpu-analysis'] })
                  toast.success(t('gpuAnalysis.handlers.ignoreSuccess', { count: rows.length }))
                })
                .catch((error) => {
                  showErrorToast(error)
                })
            },
          },
        ]}
      >
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            disabled={isTriggering}
            onClick={() => triggerAnalysis()}
          >
            <PlayCircleIcon
              className={cn('mr-2 size-4', {
                'animate-spin': isTriggering,
              })}
            />
            {t('gpuAnalysis.actions.triggerAnalysis')}
          </Button>

          <Button
            variant="outline"
            size="sm"
            disabled={analysisQuery.isFetching}
            onClick={() => analysisQuery.refetch()}
          >
            <RefreshCwIcon
              className={cn('mr-2 size-4', {
                'animate-spin': analysisQuery.isFetching,
              })}
            />
            {t('common.refresh')}
          </Button>
        </div>
      </DataTable>
    </div>
  )
}

export default GpuAnalysisOverview
