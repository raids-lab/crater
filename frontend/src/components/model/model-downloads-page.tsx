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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { ArrowLeft, Copy, Pause, Play, RotateCw, Trash2 } from 'lucide-react'
import { useMemo } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'

import ModelDownloadPhaseBadge from '@/components/badge/model-download-phase-badge'
import DocsButton from '@/components/button/docs-button'
import { TimeDistance } from '@/components/custom/time-distance'
import ModelDownloadLabel from '@/components/label/model-download-label'
import SimpleTooltip from '@/components/label/simple-tooltip'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import {
  ModelDownload,
  apiDeleteModelDownload,
  apiListModelDownloads,
  apiPauseModelDownload,
  apiResumeModelDownload,
  apiRetryModelDownload,
} from '@/services/api/modeldownload'

import { logger } from '@/utils/loglevel'

const getHeader = (key: string): string => {
  const headers: Record<string, string> = {
    name: '名称',
    source: '来源',
    category: '类别',
    status: '状态',
    path: '保存路径',
    createdAt: '创建时间',
  }
  return headers[key] || key
}

const toolbarConfig = {
  filterInput: {
    key: 'name',
    placeholder: '搜索模型名称...',
  },
  filterOptions: [
    {
      key: 'status',
      title: '状态',
      option: [
        { label: '等待中', value: 'Pending' },
        { label: '下载中', value: 'Downloading' },
        { label: '已暂停', value: 'Paused' },
        { label: '已完成', value: 'Ready' },
        { label: '失败', value: 'Failed' },
      ],
    },
  ],
  getHeader,
}

interface ModelDownloadsPageProps {
  category?: 'model' | 'dataset'
}

export function ModelDownloadsPage({ category }: ModelDownloadsPageProps) {
  const queryClient = useQueryClient()

  const query = useQuery({
    queryKey: ['model-downloads', category],
    queryFn: async () => {
      try {
        const res = await apiListModelDownloads(category)
        return Array.isArray(res.data) ? res.data : []
      } catch (error) {
        logger.error('Failed to fetch model downloads:', error)
        return []
      }
    },
    refetchInterval: 5000,
  })

  const refetchDownloads = async () => {
    try {
      await queryClient.invalidateQueries({ queryKey: ['model-downloads'] })
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }

  const { mutate: pauseDownload } = useMutation({
    mutationFn: apiPauseModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      toast.success('已暂停下载')
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || '暂停失败')
    },
  })

  const { mutate: resumeDownload } = useMutation({
    mutationFn: apiResumeModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      toast.success('已恢复下载')
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || '恢复失败')
    },
  })

  const { mutate: retryDownload } = useMutation({
    mutationFn: apiRetryModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      toast.success('已重新下载')
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || '重试失败')
    },
  })

  const { mutate: deleteDownload } = useMutation({
    mutationFn: apiDeleteModelDownload,
    onSuccess: async () => {
      await refetchDownloads()
      // 同时刷新对应的列表(模型或数据集)
      if (category === 'dataset') {
        await queryClient.invalidateQueries({ queryKey: ['data', 'dataset'] })
      } else {
        await queryClient.invalidateQueries({ queryKey: ['data', 'model'] })
      }
      toast.success('删除成功')
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || '删除失败')
    },
  })

  const columns = useMemo<ColumnDef<ModelDownload>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('name')} />,
        cell: ({ row }) => (
          <Link
            to="/portal/data/models/downloads/$id"
            params={{ id: row.original.id.toString() }}
            className="hover:text-primary cursor-pointer font-medium transition-colors duration-200"
          >
            <ModelDownloadLabel
              name={row.original.name}
              source={row.original.source}
              revision={row.original.revision}
              path={row.original.path}
            />
          </Link>
        ),
      },
      {
        accessorKey: 'source',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('source')} />
        ),
        cell: ({ row }) => (
          <span className="text-sm">
            {row.original.source === 'modelscope' ? 'ModelScope' : 'HuggingFace'}
          </span>
        ),
      },
      {
        accessorKey: 'category',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('category')} />
        ),
        cell: ({ row }) => (
          <span className="text-sm">{row.original.category === 'model' ? '模型' : '数据集'}</span>
        ),
      },
      {
        accessorKey: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('status')} />
        ),
        cell: ({ row }) => <ModelDownloadPhaseBadge status={row.original.status} />,
        filterFn: (row, id, value) => {
          return (value as string[]).includes(row.getValue(id))
        },
      },
      {
        accessorKey: 'path',
        header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('path')} />,
        cell: ({ row }) => {
          const copyPath = () => {
            navigator.clipboard.writeText(row.original.path)
            toast.success('路径已复制到剪贴板')
          }

          return (
            <div className="flex items-center gap-2">
              <SimpleTooltip tooltip={row.original.path}>
                <div className="text-muted-foreground max-w-[300px] truncate text-sm">
                  {row.original.path}
                </div>
              </SimpleTooltip>
              <Button
                variant="ghost"
                size="icon"
                className="h-7 w-7"
                onClick={copyPath}
                title="复制路径"
              >
                <Copy className="h-3.5 w-3.5" />
              </Button>
            </div>
          )
        },
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={getHeader('createdAt')} />
        ),
        cell: ({ row }) => {
          return <TimeDistance date={row.getValue('createdAt')} />
        },
        sortingFn: 'datetime',
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const download = row.original
          return (
            <div className="flex flex-row space-x-1">
              {download.status === 'Downloading' && (
                <SimpleTooltip tooltip="暂停下载">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => pauseDownload(download.id)}
                  >
                    <Pause className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
              {download.status === 'Paused' && (
                <SimpleTooltip tooltip="恢复下载">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => resumeDownload(download.id)}
                  >
                    <Play className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
              {download.status === 'Failed' && (
                <SimpleTooltip tooltip="重新下载">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8"
                    onClick={() => retryDownload(download.id)}
                  >
                    <RotateCw className="h-4 w-4" />
                  </Button>
                </SimpleTooltip>
              )}
              <SimpleTooltip tooltip="删除">
                <Button
                  variant="ghost"
                  size="icon"
                  className="text-destructive hover:bg-destructive/10 h-8 w-8"
                  onClick={() => deleteDownload(download.id)}
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </SimpleTooltip>
            </div>
          )
        },
      },
    ],
    [pauseDownload, resumeDownload, retryDownload, deleteDownload]
  )

  const pageInfo =
    category === 'dataset'
      ? {
          title: '数据集下载管理',
          description: '查看和管理从 ModelScope 或 HuggingFace 下载的数据集',
        }
      : {
          title: '模型下载管理',
          description: '查看和管理从 ModelScope 或 HuggingFace 下载的模型',
        }

  return (
    <DataTable
      info={pageInfo}
      storageKey={category === 'dataset' ? 'dataset-downloads' : 'model-downloads'}
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
      multipleHandlers={[
        {
          title: (rows) => `删除 ${rows.length} 个下载任务`,
          description: (rows) => (
            <>{rows.map((row) => row.original.name).join(', ')} 将被删除，确认要继续吗？</>
          ),
          icon: <Trash2 className="text-destructive" />,
          handleSubmit: (rows) => {
            rows.forEach((row) => {
              deleteDownload(row.original.id)
            })
          },
          isDanger: true,
        },
      ]}
    >
      <div className="flex flex-row gap-3">
        <Link to={category === 'dataset' ? '/portal/data/datasets' : '/portal/data/models'}>
          <Button variant="outline" size="sm">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {category === 'dataset' ? '返回数据集列表' : '返回模型列表'}
          </Button>
        </Link>
        <DocsButton title="查看文档" url={category === 'dataset' ? 'file/dataset' : 'file/model'} />
      </div>
    </DataTable>
  )
}
