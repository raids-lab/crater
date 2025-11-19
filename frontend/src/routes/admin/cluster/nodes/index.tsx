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
// Modified code
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { type Locale, enUS, ja, ko, zhCN } from 'date-fns/locale'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { BanIcon, Users, ZapIcon } from 'lucide-react'
import { useState } from 'react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import { useAccountNameLookup } from '@/components/node/getaccountnickname'
import { getNodeColumns, nodesToolbarConfig } from '@/components/node/node-list'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import {
  IClusterNodeTaint,
  NodeStatus,
  apiAddNodeTaint,
  apiDeleteNodeTaint,
  apichangeNodeScheduling,
} from '@/services/api/cluster'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import { logger } from '@/utils/loglevel'

import AccountSelect from './-components/account-list'

export const Route = createFileRoute('/admin/cluster/nodes/')({
  component: NodesForAdmin,
})

function NodesForAdmin() {
  const { t, i18n } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [selectedAccount, setSelectedAccount] = useState('')
  const [selectedNode, setSelectedNode] = useState('')
  const [isOccupation, setIsOccupation] = useState(true)

  // 占有原因、调度弹窗和原因
  const [occupationReason, setOccupationReason] = useState('')
  const [occupationReasonError, setOccupationReasonError] = useState('')
  const [schedulingDialogOpen, setSchedulingDialogOpen] = useState(false)
  const [schedulingNodeId, setSchedulingNodeId] = useState('')
  const [schedulingIsCurrentlyUnschedule, setSchedulingIsCurrentlyUnschedule] = useState(false)
  const [schedulingReason, setSchedulingReason] = useState('')
  const [schedulingReasonError, setSchedulingReasonError] = useState('')

  // 常用的禁止调度原因
  const commonSchedulingReasons = ['内核/驱动升级', 'GPU 驱动升级', 'GPU 故障']

  const refetchTaskList = useCallback(async () => {
    try {
      await Promise.all([
        new Promise((resolve) => setTimeout(resolve, 200)).then(() =>
          queryClient.invalidateQueries({ queryKey: ['overview', 'nodes'] })
        ),
      ])
    } catch (error) {
      logger.error('更新查询失败', error)
    }
  }, [queryClient])

  const handleNodeScheduling = useCallback(
    async (nodeId: string, reason?: string) => {
      try {
        await apichangeNodeScheduling(nodeId, { reason })
        await refetchTaskList()
        toast.success(t('nodeManagement.operationSuccess'))
      } catch (error: unknown) {
        if (error instanceof Error) {
          toast.error(t('nodeManagement.operationFailed', { error: error.message }))
        }
      } finally {
        setSchedulingDialogOpen(false)
        setSchedulingNodeId('')
        setSchedulingReason('')
        setSchedulingReasonError('')
      }
    },
    [refetchTaskList, t]
  )

  const { mutate: addNodeTaint } = useMutation({
    mutationFn: ({
      nodeName,
      taintContent,
    }: {
      nodeName: string
      taintContent: IClusterNodeTaint
    }) => apiAddNodeTaint(nodeName, taintContent),
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('nodeManagement.occupationSuccess'))
    },
    onError: (error) => {
      toast.error(t('nodeManagement.occupationFailed', { error: error.message }))
    },
  })

  const { mutate: deleteNodeTaint } = useMutation({
    mutationFn: ({
      nodeName,
      taintContent,
    }: {
      nodeName: string
      taintContent: IClusterNodeTaint
    }) => apiDeleteNodeTaint(nodeName, taintContent),
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('nodeManagement.releaseSuccess'))
    },
    onError: (error) => {
      toast.error(t('nodeManagement.releaseFailed', { error: error.message }))
    },
  })

  const nodeQuery = useQuery(queryNodes())

  const handleOccupation = useCallback(() => {
    // 在占有时验证原因
    if (isOccupation) {
      if (!occupationReason || occupationReason.trim() === '') {
        setOccupationReasonError('请填写占有原因')
        return
      }
    }

    const taintContent: IClusterNodeTaint = {
      key: 'crater.raids.io/account',
      value: selectedAccount,
      effect: 'NoSchedule',
      reason: occupationReason || '',
    }
    if (isOccupation) {
      addNodeTaint({ nodeName: selectedNode, taintContent })
    } else {
      deleteNodeTaint({ nodeName: selectedNode, taintContent })
    }
    setOpen(false)
    setOccupationReason('')
    setOccupationReasonError('')
  }, [selectedAccount, selectedNode, isOccupation, occupationReason, addNodeTaint, deleteNodeTaint])

  const { getNicknameByName } = useAccountNameLookup()

  const { data: resources } = useQuery(
    queryResources(
      true,
      (resource) => {
        return resource.type == 'gpu'
      },
      (resource) => {
        return resource.name
      }
    )
  )

  // 生成稳定的列定义
  const nodeColumns = useMemo(() => {
    // 获取当前语言对应的 date-fns locale
    const getDateLocale = (): Locale => {
      switch (i18n.language) {
        case 'en':
          return enUS
        case 'ja':
          return ja
        case 'ko':
          return ko
        default:
          return zhCN
      }
    }

    return getNodeColumns(
      (name: string) => getNicknameByName(name) || '',
      resources,
      true,
      getDateLocale()
    )
  }, [getNicknameByName, resources, i18n.language]) // 依赖项确保列定义稳定

  const columns = useMemo(
    () => [
      ...nodeColumns,
      {
        accessorKey: 'gpuDriver',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('nodeManagement.gpuDriver')} />
        ),
        cell: ({ row }) => {
          const driver = row.getValue<string>('gpuDriver')
          if (!driver) {
            return <></>
          }
          return (
            <Badge variant="outline" className="font-mono text-xs font-normal">
              {driver}
            </Badge>
          )
        },
        enableSorting: true,
      },
      {
        accessorKey: 'kernelVersion',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('nodeManagement.kernelVersion')} />
        ),
        cell: ({ row }) => {
          const version = row.getValue<string>('kernelVersion')
          return version ? (
            <Badge variant="outline" className="font-mono text-xs font-normal">
              {version}
            </Badge>
          ) : (
            <span className="text-muted-foreground">-</span>
          )
        },
        enableSorting: true,
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const nodeId = row.original.name
          const nodeStatus = row.original.status
          const taints = row.original.taints
          const unscheduleTaint =
            taints?.some(
              (t) => t.key === 'node.kubernetes.io/unschedulable' && t.effect === 'NoSchedule'
            ) || false
          const occupiedTaint = taints?.find((t) => t.key === 'crater.raids.io/account')
          const occupiedaccount = occupiedTaint?.value
          const occupiedAccountNickname = occupiedaccount
            ? getNicknameByName(occupiedaccount) || occupiedaccount
            : ''
          return (
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" className="h-8 w-8 p-0">
                  <span className="sr-only">操作</span>
                  <DotsHorizontalIcon className="size-4" />
                </Button>
              </DropdownMenuTrigger>
              <DropdownMenuContent align="end">
                <DropdownMenuLabel className="text-muted-foreground text-xs">
                  {t('nodeManagement.actionsLabel')}
                </DropdownMenuLabel>
                {nodeStatus === NodeStatus.Occupied ? (
                  <DropdownMenuItem
                    onClick={() => {
                      setSelectedNode(nodeId)
                      setIsOccupation(false)
                      setSelectedAccount(occupiedAccountNickname)
                      setOpen(true)
                    }}
                  >
                    <Users size={16} strokeWidth={2} />
                    {t('nodeManagement.releaseOccupation')}
                  </DropdownMenuItem>
                ) : (
                  <DropdownMenuItem
                    onClick={() => {
                      setSelectedNode(nodeId)
                      setIsOccupation(true)
                      setOpen(true)
                    }}
                  >
                    <Users size={16} strokeWidth={2} />
                    {t('nodeManagement.accountOccupation')}
                  </DropdownMenuItem>
                )}
                <DropdownMenuItem
                  onClick={() => {
                    setSchedulingNodeId(nodeId)
                    setSchedulingIsCurrentlyUnschedule(unscheduleTaint)
                    if (unscheduleTaint) {
                      handleNodeScheduling(nodeId, '')
                      return
                    }
                    setSchedulingReason('')
                    setSchedulingDialogOpen(true)
                  }}
                >
                  {unscheduleTaint ? (
                    <ZapIcon className="size-4" />
                  ) : (
                    <BanIcon className="size-4" />
                  )}
                  {unscheduleTaint
                    ? t('nodeManagement.enableScheduling')
                    : t('nodeManagement.disableScheduling')}
                </DropdownMenuItem>
              </DropdownMenuContent>
            </DropdownMenu>
          )
        },
      },
    ],
    [handleNodeScheduling, getNicknameByName, nodeColumns, t]
  ) // 确保依赖项是稳定的

  return (
    <>
      <DataTable
        info={{
          title: t('nodeManagement.title'),
          description: t('nodeManagement.description'),
        }}
        storageKey="admin_node_management"
        query={nodeQuery}
        columns={columns}
        toolbarConfig={nodesToolbarConfig}
        initialColumnVisibility={{
          gpuDriver: false, // 默认隐藏加速卡驱动版本列
          kernelVersion: false,
        }}
      />
      {/* 占有 / 释放 弹窗：在占有分支中增加 reason 输入 */}
      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {isOccupation
                ? t('nodeManagement.occupationDialogTitle')
                : t('nodeManagement.releaseDialogTitle')}
            </DialogTitle>
          </DialogHeader>
          {isOccupation ? (
            <div className="grid w-full gap-4 py-4">
              {/* 账号选择 */}
              <div className="grid gap-2">
                <Label className="text-muted-foreground text-sm">账户</Label>
                <AccountSelect value={selectedAccount} onChange={setSelectedAccount} />
              </div>
              {/* 占有原因输入 */}
              <div className="grid gap-2">
                <Label htmlFor="occupation-reason" className="text-muted-foreground text-sm">
                  占有原因
                </Label>
                <Input
                  id="occupation-reason"
                  value={occupationReason}
                  onChange={(e) => {
                    setOccupationReason(e.target.value)
                    // 清除错误提示
                    if (occupationReasonError) {
                      setOccupationReasonError('')
                    }
                  }}
                  placeholder="请输入占有原因"
                  className={occupationReasonError ? 'border-red-500' : ''}
                />
                {occupationReasonError && (
                  <p className="text-sm text-red-500">{occupationReasonError}</p>
                )}
              </div>
            </div>
          ) : (
            <div className="grid gap-4 py-4">
              <div className="flex items-center gap-4">
                <span>
                  {t('nodeManagement.confirmRelease', {
                    account: selectedAccount,
                  })}
                </span>
              </div>
            </div>
          )}
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">{t('nodeManagement.cancel')}</Button>
            </DialogClose>
            <Button variant="default" onClick={handleOccupation}>
              {t('nodeManagement.submit')}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      {/* 禁止调度确认 Diglog */}
      <Dialog open={schedulingDialogOpen} onOpenChange={setSchedulingDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>
              {schedulingIsCurrentlyUnschedule ? '允许节点调度' : '禁止节点调度'}
            </DialogTitle>
          </DialogHeader>
          <div className="grid gap-4 py-4">
            {/* 禁止调度原因输入 */}
            <div className="grid gap-2">
              <Label htmlFor="scheduling-reason" className="text-muted-foreground text-sm">
                原因
              </Label>
              <Input
                id="scheduling-reason"
                value={schedulingReason}
                onChange={(e) => {
                  setSchedulingReason(e.target.value)
                  // 清除错误提示
                  if (schedulingReasonError) {
                    setSchedulingReasonError('')
                  }
                }}
                placeholder="请输入原因"
                className={schedulingReasonError ? 'border-red-500' : ''}
              />
              {schedulingReasonError && (
                <p className="text-sm text-red-500">{schedulingReasonError}</p>
              )}
              {/* 常用原因快捷按钮 */}
              {!schedulingIsCurrentlyUnschedule && (
                <div className="flex gap-2 overflow-x-auto pt-2 pb-1">
                  {commonSchedulingReasons.map((reason) => (
                    <Badge
                      key={reason}
                      variant="outline"
                      className="shrink-0 cursor-pointer border-orange-600 bg-orange-50 text-orange-600 transition-colors hover:bg-orange-100 dark:border-orange-500 dark:bg-orange-950 dark:text-orange-400 dark:hover:bg-orange-900"
                      onClick={() => {
                        setSchedulingReason(reason)
                        if (schedulingReasonError) {
                          setSchedulingReasonError('')
                        }
                      }}
                    >
                      {reason}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          </div>
          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">取消</Button>
            </DialogClose>
            <Button
              onClick={() => {
                // 验证原因不得为空
                if (!schedulingReason || schedulingReason.trim() === '') {
                  setSchedulingReasonError('请填写禁止调度原因')
                  return
                }
                // 将 reason 作为 body 发送；handleNodeScheduling 已实现相应逻辑
                handleNodeScheduling(schedulingNodeId, schedulingReason)
              }}
            >
              确认
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
