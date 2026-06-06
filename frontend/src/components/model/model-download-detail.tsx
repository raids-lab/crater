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
import { useNavigate, useParams } from '@tanstack/react-router'
import {
  ActivityIcon,
  ArrowLeft,
  BotIcon,
  CalendarIcon,
  ClockIcon,
  Copy,
  DatabaseIcon,
  FileTextIcon,
  FolderIcon,
  PackageIcon,
  Pause,
  Play,
  RotateCw,
  Trash2,
} from 'lucide-react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'

import ModelDownloadPhaseBadge from '@/components/badge/model-download-phase-badge'
import DetailPageLog from '@/components/codeblock/detail-page-log'
import { TimeDistance } from '@/components/custom/time-distance'
import DetailPage from '@/components/layout/detail-page'
import { DetailPageCoreProps } from '@/components/layout/detail-page'
import PageTitle from '@/components/layout/page-title'
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
  apiDeleteModelDownload,
  apiGetModelDownload,
  apiPauseModelDownload,
  apiResumeModelDownload,
  apiRetryModelDownload,
} from '@/services/api/modeldownload'

import useFixedLayout from '@/hooks/use-fixed-layout'

import { logger } from '@/utils/loglevel'

export function ModelDownloadDetail({ ...props }: DetailPageCoreProps) {
  useFixedLayout()
  const { id } = useParams({ strict: false })
  const queryClient = useQueryClient()
  const navigate = useNavigate()

  const { data: download, isLoading } = useQuery({
    queryKey: ['model-downloads', id],
    queryFn: async () => {
      const res = await apiGetModelDownload(Number(id))
      return res.data
    },
    refetchInterval: (query) => {
      const status = query.state.data?.status
      // 如果是下载中，每 3 秒刷新一次
      return status === 'Downloading' ? 3000 : false
    },
  })

  const refetchDownload = async () => {
    try {
      await queryClient.invalidateQueries({ queryKey: ['model-downloads', id] })
      await queryClient.invalidateQueries({ queryKey: ['model-downloads'] })
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }

  const { mutate: pauseDownload } = useMutation({
    mutationFn: apiPauseModelDownload,
    onSuccess: async () => {
      await refetchDownload()
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
      await refetchDownload()
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
      await refetchDownload()
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
      await queryClient.invalidateQueries({ queryKey: ['modelDownloads'] })
      toast.success('删除成功')
      // 根据 category 返回对应页面
      const returnPath =
        download?.category === 'dataset'
          ? '/portal/data/datasets/downloads'
          : '/portal/data/models/downloads'
      navigate({ to: returnPath })
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || '删除失败')
    },
  })

  const copyPath = () => {
    if (download?.path) {
      navigator.clipboard.writeText(download.path)
      toast.success('路径已复制到剪贴板')
    }
  }

  if (isLoading || !download) {
    return <></>
  }

  return (
    <DetailPage
      {...props}
      header={
        <PageTitle
          title={download.name}
          description={`模型下载 #${download.id}`}
          tipComponent={
            <div className="flex items-center gap-4">
              <div className="bg-primary/10 rounded-xl p-3">
                <BotIcon className="text-primary size-8" />
              </div>
              <div className="min-w-0 flex-1">
                <span className="text-muted-foreground text-sm">
                  {download.source === 'modelscope' ? 'ModelScope' : 'HuggingFace'}
                </span>
              </div>
            </div>
          }
        >
          <div className="flex w-full flex-wrap gap-2 sm:w-auto sm:flex-nowrap sm:gap-3">
            <Button
              variant="outline"
              onClick={() =>
                navigate({
                  to:
                    download.category === 'dataset'
                      ? '/portal/data/datasets/downloads'
                      : '/portal/data/models/downloads',
                })
              }
            >
              <ArrowLeft className="size-4" />
              返回列表
            </Button>
            {download.status === 'Downloading' && (
              <Button variant="secondary" onClick={() => pauseDownload(download.id)}>
                <Pause className="size-4" />
                暂停
              </Button>
            )}
            {download.status === 'Paused' && (
              <Button variant="secondary" onClick={() => resumeDownload(download.id)}>
                <Play className="size-4" />
                恢复
              </Button>
            )}
            {download.status === 'Failed' && (
              <Button variant="secondary" onClick={() => retryDownload(download.id)}>
                <RotateCw className="size-4" />
                重试
              </Button>
            )}
            <AlertDialog>
              <AlertDialogTrigger asChild>
                <Button variant="destructive" title="删除下载任务" className="cursor-pointer">
                  <Trash2 className="size-4" />
                  删除
                </Button>
              </AlertDialogTrigger>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除下载任务</AlertDialogTitle>
                  <AlertDialogDescription>
                    下载任务 {download.name} 将被删除，但已下载的文件会保留。
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      deleteDownload(download.id)
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        </PageTitle>
      }
      info={[
        {
          title: '状态',
          icon: ActivityIcon,
          value: <ModelDownloadPhaseBadge status={download.status} />,
        },
        {
          title: '创建于',
          icon: CalendarIcon,
          value: <TimeDistance date={download.createdAt} />,
        },
        {
          title: '更新于',
          icon: ClockIcon,
          value: <TimeDistance date={download.updatedAt} />,
        },
        {
          title: '存储空间',
          icon: DatabaseIcon,
          value: '公共空间',
        },
        {
          title: '版本',
          icon: PackageIcon,
          value: download.revision || 'main',
        },
      ]}
      tabs={[
        {
          key: 'info',
          icon: FileTextIcon,
          label: '基本信息',
          children: (
            <div className="space-y-6 py-4">
              {/* 保存路径 */}
              <div className="space-y-3">
                <h3 className="text-lg font-semibold">模型位置</h3>
                <div className="space-y-2">
                  <div className="flex items-center gap-2">
                    <code className="bg-muted flex-1 overflow-hidden rounded px-3 py-2 font-mono text-sm text-ellipsis">
                      {download.path.replace(/^public\//, '公共空间/')}
                    </code>
                    <Button variant="outline" size="icon" onClick={copyPath} title="复制路径">
                      <Copy className="h-4 w-4" />
                    </Button>
                  </div>
                  <div className="bg-muted/50 space-y-1 rounded-lg border p-3 text-xs">
                    {download.status === 'Ready' ? (
                      <>
                        <p className="font-medium">✅ 模型下载完成</p>
                        <p className="text-muted-foreground">
                          文件已保存到共享存储 (PVC: crater-storage)
                        </p>
                        <p className="text-muted-foreground">
                          可在创建作业时通过以下路径挂载：
                          <code className="bg-background ml-1 rounded px-1.5 py-0.5">
                            {download.path}
                          </code>
                        </p>
                      </>
                    ) : (
                      <>
                        <p className="font-medium">📦 下载目标</p>
                        <p className="text-muted-foreground">
                          文件将保存到共享存储 (PVC: crater-storage)
                        </p>
                        <p className="text-muted-foreground">
                          目标路径：
                          <code className="bg-background ml-1 rounded px-1.5 py-0.5">
                            {download.path}
                          </code>
                        </p>
                      </>
                    )}
                  </div>
                </div>
              </div>

              {/* 错误信息 */}
              {download.status === 'Failed' && download.message && (
                <div className="space-y-3">
                  <h3 className="text-destructive text-lg font-semibold">错误信息</h3>
                  <ScrollArea className="h-[200px]">
                    <pre className="text-destructive rounded bg-red-50 p-4 text-sm dark:bg-red-950/20">
                      {download.message}
                    </pre>
                  </ScrollArea>
                </div>
              )}
            </div>
          ),
          scrollable: true,
        },
        {
          key: 'logs',
          icon: FolderIcon,
          label: '下载日志',
          children: (
            <DetailPageLog
              namespacedName={{
                namespace: 'crater-workspace',
                name: download?.jobName || '',
              }}
            />
          ),
        },
      ]}
    />
  )
}
