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
import React, { useEffect, type FC } from 'react'
import { Card } from '@/components/ui/card'
import { useNavigate, useParams } from 'react-router-dom'
import useBreadcrumb from '@/hooks/useBreadcrumb'
import { apiUserDeleteKaniko, apiUserGetKaniko, KanikoInfoResponse } from '@/services/api/imagepack'
import { Clock, Hash, Link2, LogsIcon, CodeIcon, ActivityIcon, Trash2Icon } from 'lucide-react'
import { TimeDistance } from '@/components/custom/TimeDistance'
import { DetailPage } from '@/components/layout/DetailPage'
import PageTitle from '@/components/layout/PageTitle'
import { shortenImageName } from '@/utils/formatter'
import { CodeContent } from '@/components/codeblock/ConfigDialog'
import DetailPageLog from '@/components/codeblock/DetailPageLog'
import ImagePhaseBadge from '@/components/badge/ImagePhaseBadge'
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
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'

type KanikoCard = React.ComponentProps<typeof Card> & {
  kanikoInfo?: KanikoInfoResponse
}

function KanikoInfo({ kanikoInfo }: KanikoCard) {
  const navigate = useNavigate()

  const { mutate: userDeleteKaniko } = useMutation({
    mutationFn: (id: number) => apiUserDeleteKaniko(id),
    onSuccess: async () => {
      navigate(-1)
      toast.success('镜像已删除')
    },
  })

  if (!kanikoInfo) {
    return null
  }

  return (
    <>
      <DetailPage
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
    </>
  )
}

export const KanikoDetail: FC = () => {
  const { name: ImagePackName } = useParams()
  const { data: kanikoInfo } = useQuery({
    queryKey: ['imagepack', 'get', ImagePackName],
    queryFn: () => apiUserGetKaniko(`${ImagePackName}`),
    select: (res) => res.data.data,
    enabled: !!ImagePackName,
  })

  const setBreadcrumb = useBreadcrumb() // 修改 BreadCrumb
  useEffect(() => {
    setBreadcrumb([{ title: '构建详情' }])
  }, [setBreadcrumb])

  return <KanikoInfo kanikoInfo={kanikoInfo} />
}

export default KanikoDetail
