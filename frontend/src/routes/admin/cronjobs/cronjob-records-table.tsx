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
import { useQuery } from '@tanstack/react-query'
import { ColumnDef } from '@tanstack/react-table'
import { CalendarIcon, CopyIcon, Trash2Icon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { DateRange } from 'react-day-picker'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Card, CardContent } from '@/components/ui/card'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  CronJobRecord,
  apiAdminCronJobRecordDelete,
  apiAdminCronJobRecordList,
  apiAdminCronJobRecordTimeRange,
} from '@/services/api/vcjob'

import CronJobRecordStatus from './cronjob-record-status'

export default function CronJobRecordsTable() {
  const { t } = useTranslation()

  // 定义状态选项（使用 i18n）
  const cronJobRecordStatuses = useMemo(
    () => [
      {
        label: t('cronJob.record.table.success'),
        value: 'success',
      },
      {
        label: t('cronJob.record.table.failed'),
        value: 'failed',
      },
      {
        label: t('cronJob.record.table.unknown'),
        value: 'unknown',
      },
    ],
    [t]
  )
  const [dateRange, setDateRange] = useState<DateRange | undefined>()
  const [minDate, setMinDate] = useState<Date | undefined>()
  const [maxDate, setMaxDate] = useState<Date | undefined>()
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [daysToDelete, setDaysToDelete] = useState<number>(1)

  // 加载时间范围
  useEffect(() => {
    const loadTimeRange = async () => {
      try {
        const res = await apiAdminCronJobRecordTimeRange()
        if (res.code === 0 && res.data) {
          if (res.data.startTime) {
            setMinDate(new Date(res.data.startTime))
          }
          if (res.data.endTime) {
            setMaxDate(new Date(res.data.endTime))
          }
        }
      } catch (error) {
        toast.error(t('cronJob.record.table.loadTimeRangeError') + error)
      }
    }
    loadTimeRange()
  }, [t])

  // 使用 useQuery 获取记录
  const recordsQuery = useQuery({
    queryKey: ['admin', 'cronjob', 'records', dateRange],
    queryFn: async () => {
      const res = await apiAdminCronJobRecordList({
        startTime: dateRange?.from?.toISOString(),
        endTime: dateRange?.to?.toISOString(),
        pageNum: 1,
        pageSize: 10000, // 获取所有记录，客户端分页
      })
      if (res.code === 0 && res.data) {
        return res.data.records || []
      }
      throw new Error(res.msg || t('cronPolicy.recordsLoadError'))
    },
    refetchInterval: 30000, // 每30秒自动刷新
  })

  // 工具函数
  const formatDate = (dateStr: string) => {
    if (!dateStr) return '-'
    const date = new Date(dateStr)
    return date.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
  }

  const copyToClipboard = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(t('cronJob.record.table.copySuccess'))
    } catch (error) {
      toast.error(t('cronJob.record.table.copyError') + error)
    }
  }

  const formatJobData = (jobData: unknown): string => {
    return JSON.stringify(jobData, undefined, 2)
  }

  const truncateText = (text: string, maxLength: number = 200): string => {
    if (text.length <= maxLength) return text
    return text.substring(0, maxLength) + '...'
  }

  const isJobDataEmpty = (jobData?: { reminded?: string[]; deleted?: string[] }): boolean => {
    if (!jobData) return true
    const hasReminded = jobData.reminded && jobData.reminded.length > 0
    const hasDeleted = jobData.deleted && jobData.deleted.length > 0
    return !hasReminded && !hasDeleted
  }

  const handleDeleteRecords = async () => {
    try {
      const endTime = new Date()
      endTime.setUTCDate(endTime.getUTCDate() - daysToDelete)

      const res = await apiAdminCronJobRecordDelete({
        endTime: endTime.toISOString(),
      })

      if (res.code === 0 && res.data) {
        const deletedCount = parseInt(res.data.deleted, 10) || 0
        toast.success(t('cronJob.record.table.deleteSuccess', { count: deletedCount }))
        setDeleteDialogOpen(false)
        recordsQuery.refetch()
      } else {
        toast.error(t('cronJob.record.table.deleteError') + res.msg)
      }
    } catch (error) {
      toast.error(t('cronJob.record.table.deleteError') + error)
    }
  }

  // 定义表格列
  const columns = useMemo<ColumnDef<CronJobRecord>[]>(() => {
    const getHeader = (key: string): string => {
      switch (key) {
        case 'name':
          return t('cronJob.record.table.jobName')
        case 'executeTime':
          return t('cronJob.record.table.executeTime')
        case 'status':
          return t('cronJob.record.table.status')
        case 'message':
          return t('cronJob.record.table.message')
        case 'affected':
          return t('cronJob.record.table.affected')
        default:
          return key
      }
    }

    return [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
        cell: ({ row }) => <div className="font-mono text-sm">{row.getValue('name')}</div>,
      },
      {
        accessorKey: 'executeTime',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('executeTime')} />
        ),
        cell: ({ row }) => <div>{formatDate(row.getValue('executeTime'))}</div>,
        sortingFn: 'datetime',
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('status')} />
        ),
        cell: ({ row }) => <CronJobRecordStatus status={row.getValue('status')} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'message',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('message')} />
        ),
        cell: ({ row }) => {
          const message = row.getValue<string>('message')
          return (
            <div className="max-w-xs truncate" title={message}>
              {message || '-'}
            </div>
          )
        },
      },
      {
        id: 'affected',
        header: () => <div className="text-xs">{getHeader('affected')}</div>,
        cell: ({ row }) => {
          const record = row.original
          if (isJobDataEmpty(record.jobData)) {
            return <div>-</div>
          }
          return (
            <TooltipProvider>
              <div className="flex items-center gap-2">
                <Tooltip>
                  <TooltipTrigger asChild>
                    <div className="cursor-help text-sm">
                      {record.jobData!.reminded && record.jobData!.reminded.length > 0 && (
                        <div>
                          {t('cronJob.record.table.reminded')}: {record.jobData!.reminded.length}
                        </div>
                      )}
                      {record.jobData!.deleted && record.jobData!.deleted.length > 0 && (
                        <div>
                          {t('cronJob.record.table.deleted')}: {record.jobData!.deleted.length}
                        </div>
                      )}
                    </div>
                  </TooltipTrigger>
                  <TooltipContent side="left" className="max-h-96 max-w-md overflow-auto">
                    <pre className="font-mono text-xs whitespace-pre-wrap">
                      {truncateText(formatJobData(record.jobData))}
                    </pre>
                  </TooltipContent>
                </Tooltip>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-6 w-6"
                  onClick={() => copyToClipboard(formatJobData(record.jobData))}
                >
                  <CopyIcon className="h-3 w-3" />
                </Button>
              </div>
            </TooltipProvider>
          )
        },
      },
    ]
  }, [t])

  // 配置 Toolbar
  const toolbarConfig: DataTableToolbarConfig = {
    globalSearch: {
      enabled: true,
    },
    filterOptions: [
      {
        key: 'name',
        title: t('cronJob.record.table.jobName'),
      },
      {
        key: 'status',
        title: t('cronJob.record.table.statusFilter'),
        option: cronJobRecordStatuses,
      },
    ],
    getHeader: (key: string) => {
      switch (key) {
        case 'name':
          return t('cronJob.record.table.jobName')
        case 'executeTime':
          return t('cronJob.record.table.executeTime')
        case 'status':
          return t('cronJob.record.table.status')
        case 'message':
          return t('cronJob.record.table.message')
        case 'affected':
          return t('cronJob.record.table.affected')
        default:
          return key
      }
    },
  }

  return (
    <Card className="mt-1 p-0">
      <CardContent className="p-6">
        <DataTable
          storageKey="admin_cronjob_records"
          query={recordsQuery}
          columns={columns}
          toolbarConfig={toolbarConfig}
          withI18n={true}
          briefChildren={
            <div className="flex flex-wrap items-center justify-between gap-4">
              <div className="flex items-center gap-2">
                <span className="text-sm">{t('cronJob.record.table.dateRange')}:</span>
                <Popover>
                  <PopoverTrigger asChild>
                    <Button variant="outline" className="w-[280px] justify-start text-left">
                      <CalendarIcon className="mr-2 h-4 w-4" />
                      {dateRange?.from ? (
                        dateRange?.to ? (
                          <>
                            {dateRange.from.toLocaleDateString()} -{' '}
                            {dateRange.to.toLocaleDateString()}
                          </>
                        ) : (
                          dateRange.from.toLocaleDateString()
                        )
                      ) : (
                        <span>{t('cronJob.record.table.selectDateRange')}</span>
                      )}
                    </Button>
                  </PopoverTrigger>
                  <PopoverContent className="w-auto p-0" align="start">
                    <Calendar
                      mode="range"
                      selected={dateRange}
                      onSelect={setDateRange}
                      numberOfMonths={2}
                      fromDate={minDate}
                      toDate={maxDate}
                      disabled={(date) => {
                        if (minDate && date < minDate) return true
                        if (maxDate && date > maxDate) return true
                        return false
                      }}
                    />
                  </PopoverContent>
                </Popover>
                {(dateRange?.from || dateRange?.to) && (
                  <Button variant="ghost" size="sm" onClick={() => setDateRange(undefined)}>
                    {t('cronJob.record.table.clearDate')}
                  </Button>
                )}
              </div>

              <AlertDialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="destructive" size="sm">
                      <Trash2Icon className="mr-2 h-4 w-4" />
                      {t('cronJob.record.table.deleteRecords')}
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    <DropdownMenuItem
                      onClick={() => {
                        setDaysToDelete(1)
                        setDeleteDialogOpen(true)
                      }}
                    >
                      {t('cronJob.record.table.delete1DayBefore')}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={() => {
                        setDaysToDelete(7)
                        setDeleteDialogOpen(true)
                      }}
                    >
                      {t('cronJob.record.table.delete7DaysBefore')}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      onClick={() => {
                        setDaysToDelete(15)
                        setDeleteDialogOpen(true)
                      }}
                    >
                      {t('cronJob.record.table.delete15DaysBefore')}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>
                      {t('cronJob.record.table.deleteConfirmTitle')}
                    </AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('cronJob.record.table.deleteConfirmMessage', { days: daysToDelete })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('cronJob.record.table.cancel')}</AlertDialogCancel>
                    <AlertDialogAction onClick={handleDeleteRecords}>
                      {t('cronJob.record.table.confirmDelete')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            </div>
          }
        />
      </CardContent>
    </Card>
  )
}
