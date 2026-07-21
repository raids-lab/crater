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
  CalendarIcon,
  ClockIcon,
  Copy,
  DatabaseIcon,
  ExternalLinkIcon,
  FileTextIcon,
  FolderIcon,
  PackageIcon,
  Pause,
  Play,
  RotateCw,
  Trash2,
  UserIcon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'

import ModelDownloadPhaseBadge from '@/components/badge/model-download-phase-badge'
import { TimeDistance } from '@/components/custom/time-distance'
import UserLabel from '@/components/label/user-label'
import DetailPage from '@/components/layout/detail-page'
import { DetailPageCoreProps } from '@/components/layout/detail-page'
import PageTitle from '@/components/layout/page-title'
import ModelDownloadProgress from '@/components/model/model-download-progress'
import ModelDownloadTokenDialog from '@/components/model/model-download-token-dialog'
import RepositorySourceMark from '@/components/model/repository-source-mark'
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
  apiGetModelDownloadLogs,
  apiPauseModelDownload,
  apiResumeModelDownload,
  apiRetryModelDownload,
} from '@/services/api/modeldownload'

import useFixedLayout from '@/hooks/use-fixed-layout'

import { logger } from '@/utils/loglevel'

export function ModelDownloadDetail({ ...props }: DetailPageCoreProps) {
  useFixedLayout()
  const { t } = useTranslation()
  const { id } = useParams({ strict: false })
  const queryClient = useQueryClient()
  const navigate = useNavigate()
  const [tokenAction, setTokenAction] = useState<'resume' | 'retry' | null>(null)

  const { data: download, isLoading } = useQuery({
    queryKey: ['model-downloads', id],
    queryFn: async () => {
      const res = await apiGetModelDownload(Number(id))
      return res.data
    },
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status === 'Downloading' || status === 'Pending' ? 3000 : false
    },
  })

  const refetchDownload = async () => {
    try {
      await queryClient.invalidateQueries({ queryKey: ['model-downloads', id] })
      await queryClient.invalidateQueries({ queryKey: ['model-downloads'] })
    } catch (error) {
      logger.error('failed to refresh model download queries', error)
    }
  }

  const pauseMutation = useMutation({
    mutationFn: apiPauseModelDownload,
    onSuccess: async () => {
      await refetchDownload()
      toast.success(t('modelDownload.action.pauseSuccess'))
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || t('modelDownload.action.pauseFailed'))
    },
  })

  const resumeMutation = useMutation({
    mutationFn: apiResumeModelDownload,
    onSuccess: async () => {
      await refetchDownload()
      setTokenAction(null)
      toast.success(t('modelDownload.action.resumeSuccess'))
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || t('modelDownload.action.resumeFailed'))
    },
  })

  const retryMutation = useMutation({
    mutationFn: apiRetryModelDownload,
    onSuccess: async () => {
      await refetchDownload()
      setTokenAction(null)
      toast.success(t('modelDownload.action.retrySuccess'))
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || t('modelDownload.action.retryFailed'))
    },
  })

  const deleteMutation = useMutation({
    mutationFn: apiDeleteModelDownload,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['modelDownloads'] })
      toast.success(t('modelDownload.action.deleteSuccess'))
      const returnPath =
        download?.category === 'dataset'
          ? '/portal/data/datasets/downloads'
          : '/portal/data/models/downloads'
      navigate({ to: returnPath })
    },
    onError: (error: unknown) => {
      const err = error as { response?: { data?: { msg?: string } } }
      toast.error(err?.response?.data?.msg || t('modelDownload.action.deleteFailed'))
    },
  })

  const copyPath = () => {
    if (download?.path) {
      navigator.clipboard.writeText(download.path)
      toast.success(t('modelDownload.pathCopied'))
    }
  }

  if (isLoading || !download) {
    return <></>
  }

  return (
    <>
      <DetailPage
        {...props}
        header={
          <PageTitle
            title={download.name}
            description={t('modelDownload.detail.description', { id: download.id })}
            tipComponent={
              <div className="flex items-center gap-4">
                <RepositorySourceMark source={download.source} category={download.category} />
                <div className="min-w-0 flex-1">
                  <a
                    href={download.sourceUrl}
                    target="_blank"
                    rel="noreferrer"
                    className="text-muted-foreground hover:text-primary inline-flex items-center gap-1 text-sm transition-colors"
                  >
                    {download.source === 'modelscope' ? 'ModelScope' : 'HuggingFace'}
                    <ExternalLinkIcon className="size-3.5" />
                  </a>
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
                {t('modelDownload.detail.back')}
              </Button>
              {download.canManage && download.status === 'Downloading' && (
                <Button
                  variant="secondary"
                  disabled={pauseMutation.isPending}
                  onClick={() => pauseMutation.mutate(download.id)}
                >
                  <Pause className="size-4" />
                  {t('modelDownload.action.pause')}
                </Button>
              )}
              {download.canManage && download.status === 'Paused' && (
                <Button
                  variant="secondary"
                  disabled={resumeMutation.isPending}
                  onClick={() => setTokenAction('resume')}
                >
                  <Play className="size-4" />
                  {t('modelDownload.action.resume.confirm')}
                </Button>
              )}
              {download.canManage && download.status === 'Failed' && (
                <Button
                  variant="secondary"
                  disabled={retryMutation.isPending}
                  onClick={() => setTokenAction('retry')}
                >
                  <RotateCw className="size-4" />
                  {t('modelDownload.action.retry.confirm')}
                </Button>
              )}
              {download.canManage && (
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button
                      variant="destructive"
                      disabled={deleteMutation.isPending}
                      title={t('modelDownload.action.deleteTitle')}
                      className="cursor-pointer"
                    >
                      <Trash2 className="size-4" />
                      {t('modelDownload.action.delete')}
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>{t('modelDownload.action.deleteTitle')}</AlertDialogTitle>
                      <AlertDialogDescription>
                        {t('modelDownload.action.deleteDescription', { name: download.name })}
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        disabled={deleteMutation.isPending}
                        onClick={() => deleteMutation.mutate(download.id)}
                      >
                        {deleteMutation.isPending
                          ? t('modelDownload.action.processing')
                          : t('modelDownload.action.delete')}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              )}
            </div>
          </PageTitle>
        }
        info={[
          {
            title: t('modelDownload.detail.status'),
            icon: ActivityIcon,
            value: <ModelDownloadPhaseBadge status={download.status} />,
          },
          {
            title: t('modelDownload.detail.creator'),
            icon: UserIcon,
            value: <UserLabel info={download.userInfo} />,
          },
          {
            title: t('modelDownload.detail.createdAt'),
            icon: CalendarIcon,
            value: <TimeDistance date={download.createdAt} />,
          },
          {
            title: t('modelDownload.detail.localUpdatedAt'),
            icon: ClockIcon,
            value: <TimeDistance date={download.updatedAt} />,
          },
          ...(download.sourceUpdatedAt
            ? [
                {
                  title: t('modelDownload.detail.sourceUpdatedAt'),
                  icon: ClockIcon,
                  value: <TimeDistance date={download.sourceUpdatedAt} />,
                },
              ]
            : []),
          {
            title: t('modelDownload.detail.storageSpace'),
            icon: DatabaseIcon,
            value: t('modelDownload.detail.publicSpace'),
          },
          {
            title: t('modelDownload.detail.revision'),
            icon: PackageIcon,
            value: download.revision || 'main',
          },
        ]}
        tabs={[
          {
            key: 'info',
            icon: FileTextIcon,
            label: t('modelDownload.detail.basicInfo'),
            children: (
              <div className="space-y-1 md:space-y-2 lg:space-y-3">
                {(download.status === 'Downloading' ||
                  download.status === 'Paused' ||
                  download.status === 'Ready') && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="flex items-center text-xl">
                        <ActivityIcon className="mr-2 h-5 w-5 text-blue-500" />
                        {t('modelDownload.detail.progress')}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ModelDownloadProgress download={download} className="max-w-md" />
                    </CardContent>
                  </Card>
                )}

                <Card>
                  <CardHeader>
                    <CardTitle className="flex items-center text-xl">
                      <FolderIcon className="mr-2 h-5 w-5 text-blue-500" />
                      {download.category === 'dataset'
                        ? t('modelDownload.detail.datasetLocation')
                        : t('modelDownload.detail.modelLocation')}
                    </CardTitle>
                    <CardDescription>
                      {download.status === 'Ready'
                        ? t('modelDownload.detail.pathReady')
                        : t('modelDownload.detail.pathPending')}
                    </CardDescription>
                  </CardHeader>
                  <CardContent>
                    <div className="flex items-center gap-2">
                      <code className="bg-muted flex-1 overflow-hidden rounded px-3 py-2 font-mono text-sm text-ellipsis">
                        {download.path}
                      </code>
                      <Button
                        variant="outline"
                        size="icon"
                        onClick={copyPath}
                        title={t('modelDownload.copyPath')}
                      >
                        <Copy className="h-4 w-4" />
                      </Button>
                    </div>
                  </CardContent>
                </Card>

                {download.status === 'Failed' && download.message && (
                  <Card>
                    <CardHeader>
                      <CardTitle className="text-destructive flex items-center text-xl">
                        <ActivityIcon className="mr-2 h-5 w-5" />
                        {t('modelDownload.detail.error')}
                      </CardTitle>
                    </CardHeader>
                    <CardContent>
                      <ScrollArea className="max-h-[200px]">
                        <pre className="text-destructive rounded bg-red-50 p-4 text-sm break-words whitespace-pre-wrap dark:bg-red-950/20">
                          {download.message}
                        </pre>
                      </ScrollArea>
                    </CardContent>
                  </Card>
                )}
              </div>
            ),
            scrollable: true,
          },
          ...(download.canViewLogs
            ? [
                {
                  key: 'logs',
                  icon: FolderIcon,
                  label: t('modelDownload.detail.logs'),
                  children: <ModelDownloadStoredLogs id={download.id} status={download.status} />,
                },
              ]
            : []),
        ]}
      />
      <ModelDownloadTokenDialog
        action={tokenAction ?? 'resume'}
        downloadName={download.name}
        isPending={resumeMutation.isPending || retryMutation.isPending}
        open={tokenAction !== null}
        onOpenChange={(open) => !open && setTokenAction(null)}
        onSubmit={(token) => {
          if (tokenAction === 'resume') {
            resumeMutation.mutate({ id: download.id, token })
          } else if (tokenAction === 'retry') {
            retryMutation.mutate({ id: download.id, token })
          }
        }}
      />
    </>
  )
}

function ModelDownloadStoredLogs({ id, status }: { id: number; status: string }) {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['model-downloads', id, 'logs'],
    queryFn: () => apiGetModelDownloadLogs(id),
    refetchInterval: status === 'Downloading' || status === 'Pending' ? 5000 : false,
  })

  return (
    <Card className="dark:bg-muted/30 bg-sidebar h-[calc(100vh_-_300px)] overflow-hidden rounded-md p-1 dark:border">
      <ScrollArea className="h-full">
        <pre className="px-3 py-3 text-sm break-words whitespace-pre-wrap dark:text-blue-300">
          {isLoading
            ? t('modelDownload.detail.logsLoading')
            : data?.data || t('modelDownload.detail.noLogs')}
        </pre>
      </ScrollArea>
    </Card>
  )
}
