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
import { useNavigate } from '@tanstack/react-router'
import {
  ActivityIcon,
  Clock,
  CodeIcon,
  CopyIcon,
  Hash,
  Link2,
  LogsIcon,
  SquareIcon,
  Trash2Icon,
} from 'lucide-react'
import React from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

import ImagePhaseBadge from '@/components/badge/image-phase-badge'
import { CodeContent } from '@/components/codeblock/config-dialog'
import DetailPageLog from '@/components/codeblock/detail-page-log'
import { TimeDistance } from '@/components/custom/time-distance'
import DetailPage, { DetailPageCoreProps } from '@/components/layout/detail-page'
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

import { KanikoInfoResponse, apiUserRemoveKanikoList } from '@/services/api/imagepack'
import { queryBuildDetail } from '@/services/query/image'

import useIsAdmin from '@/hooks/use-admin'

import { shortenImageName } from '@/utils/formatter'
import { logger } from '@/utils/loglevel'

type KanikoCard = React.ComponentProps<typeof Card> & {
  kanikoInfo?: KanikoInfoResponse
  name?: string
}

function RegistryInfo({ kanikoInfo, name: imageName, ...props }: DetailPageCoreProps & KanikoCard) {
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const isAdmin = useIsAdmin()

  // 统一的移除函数(删除和取消都使用这个)
  const { mutate: removeKaniko, isPending } = useMutation({
    mutationFn: (params: { id: number; isCancel?: boolean; name?: string }) =>
      apiUserRemoveKanikoList([params.id]),
    onSuccess: async (_, variables) => {
      // 只在取消操作时等待1.5秒让后端有足够时间更新数据
      if (variables.isCancel) {
        await new Promise((resolve) => setTimeout(resolve, 2000))
      }

      if (variables.isCancel) {
        // 取消操作:刷新当前页面数据
        await queryClient.invalidateQueries({ queryKey: ['imagepack', 'get'] })
        toast.success('已取消构建任务')
      } else {
        // 删除操作:先移除查询缓存,再导航到列表页
        // 移除当前镜像的查询缓存,避免路由卸载时重新获取已删除的数据
        if (variables.name) {
          queryClient.removeQueries({ queryKey: ['imagepack', 'get', variables.name] })
        }
        const targetPath = isAdmin ? '/admin/env/registry' : '/portal/env/registry'
        await navigate({ to: targetPath })
        toast.success('镜像已删除')
      }
    },
    onError: (err, variables) => {
      if (variables.isCancel) {
        logger.error('取消镜像构建失败', err)
        toast.error('取消失败,请稍后重试')
      } else {
        logger.error('删除镜像失败', err)
        toast.error('删除失败,请稍后重试')
      }
    },
  })

  if (!kanikoInfo) {
    return null
  }

  // 判断是否应该显示删除按钮（Finished、Failed、Canceled状态）
  const shouldShowDelete =
    kanikoInfo.status === 'Finished' ||
    kanikoInfo.status === 'Failed' ||
    kanikoInfo.status === 'Canceled'

  // 判断是否应该显示取消按钮（Pending、Running状态）
  const shouldShowCancel = kanikoInfo.status === 'Pending' || kanikoInfo.status === 'Running'

  return (
    <DetailPage
      {...props}
      header={
        <PageTitle
          title={kanikoInfo.description}
          // description={shortestImageName(kanikoInfo.imageLink)}
        >
          <div className="flex flex-row gap-3">
            {/* 复制链接按钮 - 仅在镜像构建完成时显示 */}
            {kanikoInfo.status === 'Finished' && (
              <Button
                variant="outline"
                title="复制镜像链接"
                onClick={() => {
                  const text = kanikoInfo.imageLink ?? ''
                  if (!text) {
                    toast.error('没有可复制的链接')
                    return
                  }
                  navigator.clipboard
                    .writeText(text)
                    .then(() => toast.success('镜像链接已复制'))
                    .catch((err) => {
                      logger.error('复制镜像链接失败', err)
                      toast.error('复制失败，请手动复制')
                    })
                }}
              >
                <CopyIcon className="size-4" />
                镜像链接
              </Button>
            )}

            {shouldShowCancel && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button
                      title="取消构建"
                      className="bg-highlight-orange hover:bg-highlight-orange/90 cursor-pointer"
                      disabled={isPending}
                    >
                      <SquareIcon className="size-4" />
                      取消构建
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>取消构建任务</AlertDialogTitle>
                    <AlertDialogDescription>
                      镜像 {shortenImageName(kanikoInfo.imageLink)} 的构建任务将被取消。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel disabled={isPending}>返回</AlertDialogCancel>
                    <AlertDialogAction
                      className="bg-amber-600 hover:bg-amber-700"
                      disabled={isPending}
                      onClick={() => {
                        removeKaniko({ id: kanikoInfo.ID, isCancel: true, name: imageName })
                      }}
                    >
                      {isPending ? '取消中...' : '确认取消'}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
            )}
            {shouldShowDelete && (
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <div>
                    <Button variant="destructive" title="删除镜像" disabled={isPending}>
                      <Trash2Icon className="size-4" />
                      删除镜像
                    </Button>
                  </div>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>删除镜像构建任务</AlertDialogTitle>
                    <AlertDialogDescription>
                      镜像 {shortenImageName(kanikoInfo.imageLink)} 将被删除，请确认镜像已不再需要。
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel disabled={isPending}>取消</AlertDialogCancel>
                    <AlertDialogAction
                      variant="destructive"
                      disabled={isPending}
                      onClick={() => {
                        removeKaniko({ id: kanikoInfo.ID, isCancel: false, name: imageName })
                      }}
                    >
                      {isPending ? '删除中...' : '删除'}
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
          icon: Hash,
          title: 'ID',
          value: kanikoInfo.ID,
        },
        {
          icon: Link2,
          title: 'URL',
          value: <span className="font-mono">{shortenImageName(kanikoInfo?.imageLink)}</span>,
          className: 'col-span-2',
        },
        {
          icon: Clock,
          title: '创建于',
          value: <TimeDistance date={kanikoInfo.createdAt} />,
        },
        {
          icon: ActivityIcon,
          title: '状态',
          value: <ImagePhaseBadge status={kanikoInfo.status} />,
        },
      ]}
      tabs={[
        {
          key: 'info',
          icon: LogsIcon,
          label: '构建日志',
          children: (
            <DetailPageLog
              namespacedName={{
                namespace: kanikoInfo?.podNameSpace ?? '',
                name: kanikoInfo?.podName ?? '',
              }}
            />
          ),
          hidden: kanikoInfo?.podName === '',
        },
        {
          key: 'dockerfile',
          icon: CodeIcon,
          label:
            kanikoInfo.buildSource === 'Dockerfile' || kanikoInfo.buildSource === 'PipApt'
              ? 'Dockerfile'
              : 'Envd 配置',
          children: (
            <CodeContent
              data={kanikoInfo.dockerfile}
              language={
                kanikoInfo.buildSource === 'Dockerfile' || kanikoInfo.buildSource === 'PipApt'
                  ? 'dockerfile'
                  : 'python'
              }
            />
          ),
          hidden: kanikoInfo.buildSource === 'Snapshot',
        },
      ]}
    />
  )
}

const RegistryDetail = ({ name, ...props }: DetailPageCoreProps & { name: string }) => {
  const { data: kanikoInfo } = useQuery(queryBuildDetail(name))

  return <RegistryInfo kanikoInfo={kanikoInfo} name={name} {...props} />
}

export default RegistryDetail
