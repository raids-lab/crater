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
import { ColumnDef } from '@tanstack/react-table'
import { useAtomValue } from 'jotai'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import {
  AlertTriangle,
  CheckCheck,
  DatabaseIcon,
  InfoIcon,
  PackagePlusIcon,
  RedoDotIcon,
  SquareCheckBig,
  Trash2Icon,
} from 'lucide-react'
import { type FC, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Dialog, DialogContent } from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'

import ImagePhaseBadge from '@/components/badge/image-phase-badge'
import DocsButton from '@/components/button/docs-button'
import ListedButton from '@/components/button/listed-button'
import { TimeDistance } from '@/components/custom/time-distance'
import ImageLabel from '@/components/label/image-label'
import TooltipLink from '@/components/label/tooltip-link'
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
  ImageLinkPair,
  ImagePackSource,
  ImagePackStatus,
  KanikoInfoResponse,
  ListKanikoResponse,
  getHeader,
  imagepackStatuses,
} from '@/services/api/imagepack'
import { IResponse } from '@/services/types'

import useIsAdmin from '@/hooks/use-admin'

import { formatBytes } from '@/utils/formatter'
import { logger } from '@/utils/loglevel'
import { atomUserInfo } from '@/utils/store'

import { ValidDialog } from '../images/valid-dialog'
import { CudaBaseImageSheet } from './cuda-base-image-sheet'
import { DockerfileSheet } from './dockerfile-sheet'
import { EnvdRawSheet } from './envd-raw-sheet'
import { EnvdSheet } from './envd-sheet'
import { PipAptSheet } from './pip-apt-sheet'
import { ProjectDetail } from './project-detail'

const toolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: '搜索镜像',
    key: 'image',
  },
  filterOptions: [
    {
      key: 'status',
      title: '状态',
      option: imagepackStatuses,
    },
  ],
  getHeader: getHeader,
}

interface KanikoListTableProps {
  apiListKaniko: () => Promise<IResponse<ListKanikoResponse>>
  apiDeleteKanikoList: (idList: number[]) => Promise<IResponse<string>>
  isAdminMode: boolean
}

export const KanikoListTable: FC<KanikoListTableProps> = ({
  apiListKaniko,
  apiDeleteKanikoList,
  isAdminMode,
}) => {
  const queryClient = useQueryClient()
  const [openPipAptSheet, setOpenPipAptSheet] = useState(false)
  const [openDockerfileSheet, setOpenDockerfileSheet] = useState(false)
  const [openEnvdSheet, setOpenEnvdSheet] = useState(false)
  const [openEnvdRawSheet, setOpenEnvdRawSheet] = useState(false)
  const [openCudaBaseImageSheet, setOpenCudaBaseImageSheet] = useState(false)
  const [imagePackName, setImagePackName] = useState<string>('')
  const navigate = useNavigate()
  const isAdmin = useIsAdmin()
  const [openCheckDialog, setCheckOpenDialog] = useState(false)
  const user = useAtomValue(atomUserInfo)
  const [selectedLinkPairs, setSelectedLinkPairs] = useState<ImageLinkPair[]>([])

  const imageQuery = useQuery({
    queryKey: ['imagepack', 'list'],
    queryFn: () => apiListKaniko(),
    select: (res) =>
      res.data.kanikoList.map((i) => ({
        ...i,
        image: `${i.imageLink} (${i.description})`,
      })),
  })
  const refetchImagePackList = async () => {
    try {
      // 并行发送所有异步请求
      await Promise.all([queryClient.invalidateQueries({ queryKey: ['imagepack', 'list'] })])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }

  const { mutate: deleteKanikoList } = useMutation({
    mutationFn: (idList: number[]) => apiDeleteKanikoList(idList),
    onSuccess: async () => {
      await refetchImagePackList()
      toast.success('镜像已删除')
    },
  })

  let columns: ColumnDef<KanikoInfoResponse>[] = [
    {
      accessorKey: 'image',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('image')} />,
      cell: ({ row }) => (
        <TooltipLink
          name={
            <ImageLabel
              description={row.original.description}
              url={row.original.imageLink}
              tags={row.original.tags}
            />
          }
          to={`${row.original.imagepackName}`}
          tooltip={`查看镜像详情`}
        />
      ),
    },
    {
      id: 'archs',
      accessorKey: 'archs',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('archs')} />,
      cell: ({ row }) => (
        <div className="flex flex-wrap gap-1">
          {(row.original.archs || []).map((arch, index) => (
            <Badge key={index} variant="outline" className="text-xs">
              {arch}
            </Badge>
          ))}
        </div>
      ),
    },
    {
      id: 'userInfo',
      accessorKey: 'userInfo',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader('userInfo')} />
      ),
      cell: ({ row }) => <UserLabel info={row.original.userInfo} />,
    },
    {
      accessorKey: 'createdAt',
      header: ({ column }) => (
        <DataTableColumnHeader column={column} title={getHeader('createdAt')} />
      ),
      cell: ({ row }) => {
        return <TimeDistance date={row.getValue('createdAt')}></TimeDistance>
      },
      sortingFn: 'datetime',
    },
    {
      accessorKey: 'status',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('status')} />,
      cell: ({ row }) => {
        return <ImagePhaseBadge status={row.getValue<ImagePackStatus>('status')} />
      },
      filterFn: (row, id, value) => {
        return (value as string[]).includes(row.getValue(id))
      },
    },
    {
      accessorKey: 'size',
      header: ({ column }) => <DataTableColumnHeader column={column} title={getHeader('size')} />,
      cell: ({ row }) => {
        return (
          <>
            {row.getValue<number>('size') > 0 && <span>{formatBytes(row.getValue('size'))}</span>}
          </>
        )
      },
    },
    {
      id: 'actions',
      enableHiding: false,
      cell: ({ row }) => {
        const kanikoInfo = row.original
        return (
          <div className="flex flex-row space-x-1">
            <AlertDialog>
              <DropdownMenu>
                <DropdownMenuTrigger asChild>
                  <Button variant="ghost" className="h-8 w-8 p-0">
                    <span className="sr-only">操作</span>
                    <DotsHorizontalIcon className="size-4" />
                  </Button>
                </DropdownMenuTrigger>
                <DropdownMenuContent align="end">
                  <DropdownMenuLabel className="text-muted-foreground text-xs">
                    操作
                  </DropdownMenuLabel>
                  <DropdownMenuItem
                    onClick={() => {
                      // Navigate to registry detail page using proper route structure
                      if (isAdmin) {
                        navigate({
                          to: '/admin/env/registry/$name',
                          params: { name: kanikoInfo.imagepackName },
                        })
                      } else {
                        navigate({
                          to: '/portal/env/registry/$name',
                          params: { name: kanikoInfo.imagepackName },
                        })
                      }
                    }}
                  >
                    <InfoIcon className="text-highlight-emerald" />
                    详情
                  </DropdownMenuItem>

                  <DropdownMenuItem
                    onClick={() => {
                      setImagePackName(kanikoInfo.imagepackName)
                      if (kanikoInfo.buildSource === ImagePackSource.EnvdAdvanced) {
                        setOpenEnvdSheet(true)
                      } else if (kanikoInfo.buildSource === ImagePackSource.EnvdRaw) {
                        setOpenEnvdRawSheet(true)
                      } else if (kanikoInfo.buildSource === ImagePackSource.Dockerfile) {
                        setOpenDockerfileSheet(true)
                      } else if (kanikoInfo.buildSource === ImagePackSource.PipApt) {
                        setOpenPipAptSheet(true)
                      } else {
                        toast.error('该镜像不支持克隆')
                      }
                    }}
                  >
                    <RedoDotIcon className="text-highlight-purple size-4" />
                    克隆
                  </DropdownMenuItem>

                  <AlertDialogTrigger asChild>
                    <DropdownMenuItem>
                      <Trash2Icon className="text-destructive" />
                      删除
                    </DropdownMenuItem>
                  </AlertDialogTrigger>
                </DropdownMenuContent>
              </DropdownMenu>
              <AlertDialogContent>
                <AlertDialogHeader>
                  <AlertDialogTitle>删除镜像</AlertDialogTitle>
                  <AlertDialogDescription>
                    镜像「{kanikoInfo?.imageLink}
                    」将删除
                  </AlertDialogDescription>
                </AlertDialogHeader>
                <AlertDialogFooter>
                  <AlertDialogCancel>取消</AlertDialogCancel>
                  <AlertDialogAction
                    variant="destructive"
                    onClick={() => {
                      deleteKanikoList([kanikoInfo.ID])
                    }}
                  >
                    删除
                  </AlertDialogAction>
                </AlertDialogFooter>
              </AlertDialogContent>
            </AlertDialog>
          </div>
        )
      },
    },
  ]
  columns = columns.filter((column) => column.id !== 'nickName' || isAdminMode)
  return (
    <>
      <DataTable
        info={
          isAdminMode
            ? {
                title: '镜像制作',
                description: '支持 Dockerfile 、低代码、快照等方式制作镜像',
              }
            : {
                title: '镜像制作',
                description: '支持 Dockerfile 、低代码、快照等方式制作镜像',
              }
        }
        storageKey="image_registry"
        query={imageQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        className="lg:col-span-2"
        multipleHandlers={[
          {
            title: (rows) => `删除 ${rows.length} 个镜像创建任务，以及对应镜像链接`,
            description: (rows) => (
              <div className="border-destructive/20 bg-destructive/5 rounded-md border px-4 py-3">
                <div className="flex items-start gap-3">
                  <div className="w-full overflow-hidden">
                    <p className="flex items-center gap-2 font-medium">
                      <AlertTriangle className="text-destructive h-4 w-4" />
                      <span className="text-destructive text-sm leading-4">
                        以下镜像创建任务和对应镜像链接将被删除，确认要继续吗？
                      </span>
                    </p>
                    <Separator className="my-2" />
                    <ScrollArea className="h-[200px] w-full overflow-hidden">
                      <div className="p-4">
                        {rows.map((row, index) => (
                          <p key={index} className="text-muted-foreground text-sm">
                            {`『${row.original.description}』`}
                          </p>
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                </div>
              </div>
            ),
            icon: <Trash2Icon className="text-destructive" />,
            handleSubmit: (rows) => {
              const ids = rows.map((row) => row.original.ID)
              deleteKanikoList(ids)
            },
            isDanger: true,
          },
          {
            title: (rows) => `检测 ${rows.length} 个镜像链接`,
            description: (rows) => (
              <div className="rounded-md border border-green-600/20 bg-green-600/5 px-4 py-3">
                <div className="flex items-start gap-3">
                  <div className="w-full overflow-hidden">
                    <p className="flex items-center gap-2 font-medium">
                      <SquareCheckBig className="h-4 w-4 text-green-600" />
                      <span className="text-sm leading-4 text-green-600">以下镜像链接将被检测</span>
                    </p>
                    <Separator className="my-2" />
                    <ScrollArea className="h-[200px] w-full overflow-hidden">
                      <div className="p-4">
                        {rows.map((row, index) => (
                          <p key={index} className="text-muted-foreground text-sm">
                            {`『${row.original.description}』`}
                          </p>
                        ))}
                      </div>
                    </ScrollArea>
                  </div>
                </div>
              </div>
            ),
            icon: <CheckCheck className="text-green-600" />,
            handleSubmit: (rows) => {
              setSelectedLinkPairs(
                rows.map((row) => ({
                  id: row.original.ID,
                  imageLink: row.original.imageLink,
                  description: row.original.description,
                  creator: row.original.userInfo,
                }))
              )
              setCheckOpenDialog(true)
            },
            isDanger: false,
          },
        ]}
        briefChildren={
          !isAdminMode ? (
            <ProjectDetail
              successImageNumber={
                imageQuery.data?.filter((c) => c.status == 'Finished').length ?? 0
              }
            />
          ) : null
        }
      >
        {!isAdminMode ? (
          <div className="flex flex-row gap-3">
            <DocsButton title="查看文档" url="image/imageCreate" />
            <ListedButton
              icon={<PackagePlusIcon />}
              renderTitle={(title) => title}
              itemTitle="构建方式"
              items={[
                {
                  key: 'envd',
                  title: 'Python + CUDA 自定义构建',
                  action: () => {
                    setOpenEnvdSheet(true)
                  },
                },
                {
                  key: 'pip-apt',
                  title: '基于现有镜像构建',
                  action: () => {
                    setOpenPipAptSheet(true)
                  },
                },
                {
                  key: 'dockerfile',
                  title: '基于 Dockerfile 构建',
                  action: () => {
                    setOpenDockerfileSheet(true)
                  },
                },
                {
                  key: 'envd-raw',
                  title: '基于 Envd 构建',
                  action: () => {
                    setOpenEnvdRawSheet(true)
                  },
                },
              ]}
              cacheKey="imagepack"
            />
          </div>
        ) : (
          <div className="flex flex-row gap-3">
            <Button onClick={() => setOpenCudaBaseImageSheet(true)} variant="default">
              <DatabaseIcon />
              导入CUDA Base镜像
            </Button>
          </div>
        )}
      </DataTable>
      {!isAdminMode ? (
        <div>
          <EnvdSheet
            isOpen={openEnvdSheet}
            onOpenChange={setOpenEnvdSheet}
            title="Python + CUDA 自定义构建"
            description="Python+CUDA指定版本构建"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenEnvdSheet(false)
              setImagePackName('')
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <EnvdRawSheet
            isOpen={openEnvdRawSheet}
            onOpenChange={setOpenEnvdRawSheet}
            title="高级 Envd 构建脚本"
            description="直接编写 Envd 构建脚本，实现更复杂的定制化"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenEnvdRawSheet(false)
              setImagePackName('')
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <PipAptSheet
            isOpen={openPipAptSheet}
            onOpenChange={setOpenPipAptSheet}
            title="基于现有镜像构建"
            description="基于平台提供的基础镜像，快速制作自定义镜像"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenPipAptSheet(false)
              setImagePackName('')
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
          <DockerfileSheet
            isOpen={openDockerfileSheet}
            onOpenChange={setOpenDockerfileSheet}
            title="基于 Dockerfile 构建镜像"
            description="基于 Dockerfile 制作镜像"
            className="sm:max-w-3xl"
            closeSheet={() => {
              setOpenDockerfileSheet(false)
              setImagePackName('')
            }}
            imagePackName={imagePackName}
            setImagePackName={setImagePackName}
          />
        </div>
      ) : null}

      {/* CUDA Base镜像管理Sheet - 只在管理员模式下显示 */}
      {isAdminMode && (
        <CudaBaseImageSheet
          isOpen={openCudaBaseImageSheet}
          onOpenChange={setOpenCudaBaseImageSheet}
          title="CUDA Base镜像管理"
          description="管理系统中的CUDA Base镜像列表"
          className="sm:max-w-2xl"
          closeSheet={() => {
            setOpenCudaBaseImageSheet(false)
          }}
        />
      )}

      <Dialog open={openCheckDialog} onOpenChange={setCheckOpenDialog}>
        <DialogContent>
          <ValidDialog
            linkPairs={selectedLinkPairs}
            onDeleteLinks={(invalidPairs: ImageLinkPair[]) => {
              deleteKanikoList(
                invalidPairs
                  .filter((pair) => pair.creator.username === user?.name)
                  .map((pair) => pair.id)
              )
            }}
          />
        </DialogContent>
      </Dialog>
    </>
  )
}
