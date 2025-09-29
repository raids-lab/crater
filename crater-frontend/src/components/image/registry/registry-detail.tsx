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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useRouter } from '@tanstack/react-router'
import { ActivityIcon, Clock, CodeIcon, Hash, Link2, LogsIcon, Trash2Icon } from 'lucide-react'
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

import { KanikoInfoResponse, apiUserDeleteKaniko } from '@/services/api/imagepack'
import { queryBuildDetail } from '@/services/query/image'

import { shortenImageName } from '@/utils/formatter'

type KanikoCard = React.ComponentProps<typeof Card> & {
  kanikoInfo?: KanikoInfoResponse
}

function RegistryInfo({ kanikoInfo, ...props }: DetailPageCoreProps & KanikoCard) {
  const router = useRouter()

  const { mutate: userDeleteKaniko } = useMutation({
    mutationFn: (id: number) => apiUserDeleteKaniko(id),
    onSuccess: async () => {
      router.history.back()
      toast.success('镜像已删除')
    },
  })

  if (!kanikoInfo) {
    return null
  }

  return (
    <DetailPage
      {...props}
      header={
        <PageTitle
          title={kanikoInfo.description}
          // description={shortestImageName(kanikoInfo.imageLink)}
        >
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <div>
                <Button variant="destructive" title="删除镜像">
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
                <AlertDialogCancel>取消</AlertDialogCancel>
                <AlertDialogAction
                  variant="destructive"
                  onClick={() => {
                    userDeleteKaniko(kanikoInfo.ID)
                  }}
                >
                  删除
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
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

  return <RegistryInfo kanikoInfo={kanikoInfo} {...props} />
}

export default RegistryDetail
