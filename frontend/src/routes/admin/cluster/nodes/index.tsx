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
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { BanIcon, RotateCcw, Users, ZapIcon } from 'lucide-react'
import { useState } from 'react'
import { useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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

import { useAccountNameLookup } from '@/components/node/getaccountnickname'
import { getNodeColumns, nodesToolbarConfig } from '@/components/node/node-list'
import { DataTable } from '@/components/query-table'

import {
  CraterArmTaint,
  IClusterNodeTaint,
  INodeBriefInfo,
  JoinTaint,
  NodeStatus,
  apiAddNodeTaint,
  apiDeleteNodeTaint,
  apichangeNodeScheduling,
} from '@/services/api/cluster'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import { logger } from '@/utils/loglevel'

import AccountSelect from './-components/account-list'

// 恢复ARM污点Dialog组件
interface RestoreArmTaintDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  armNodes: INodeBriefInfo[]
  onConfirm: (nodeNames: string[]) => void
}

function RestoreArmTaintDialog({
  open,
  onOpenChange,
  armNodes,
  onConfirm,
}: RestoreArmTaintDialogProps) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>同步 ARM 污点</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <p className="text-muted-foreground text-sm">
            检测到以下 ARM 架构节点缺少污点，是否为这些节点添加{' '}
            <code className="bg-muted mx-1 rounded px-1.5 py-0.5 text-xs">
              {JoinTaint(CraterArmTaint)}
            </code>{' '}
            污点？
          </p>

          <div className="bg-muted/30 max-h-48 overflow-y-auto rounded-md border p-3">
            <div className="space-y-2">
              {armNodes.map((node, index) => (
                <div
                  key={index}
                  className="bg-background flex items-center gap-2 rounded-sm px-3 py-2 text-sm"
                >
                  <div className="h-2 w-2 rounded-full bg-orange-500"></div>
                  <span className="font-mono">{node.name}</span>
                </div>
              ))}
            </div>
          </div>
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">取消</Button>
          </DialogClose>
          <Button
            onClick={() => {
              const nodeNames = armNodes.map((node) => node.name)
              onConfirm(nodeNames)
            }}
          >
            确认
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export const Route = createFileRoute('/admin/cluster/nodes/')({
  component: NodesForAdmin,
})

function NodesForAdmin() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [open, setOpen] = useState(false)
  const [selectedAccount, setSelectedAccount] = useState('')
  const [selectedNode, setSelectedNode] = useState('')
  const [isOccupation, setIsOccupation] = useState(true)

  // 恢复Arm污点相关状态
  const [restoreArmDialogOpen, setRestoreArmDialogOpen] = useState(false)
  const [armNodesToRestore, setArmNodesToRestore] = useState<INodeBriefInfo[]>([])

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
    (nodeId: string) => {
      // 调用 mutation
      apichangeNodeScheduling(nodeId)
        .then(() => {
          refetchTaskList()
          toast.success(t('nodeManagement.operationSuccess'))
        })
        .catch((error) => {
          toast.error(t('nodeManagement.operationFailed', { error: error.message }))
        })
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

  // 批量恢复ARM污点的mutation
  const { mutate: batchRestoreArmTaint } = useMutation({
    mutationFn: async (nodeNames: string[]) => {
      const promises = nodeNames.map((nodeName) => apiAddNodeTaint(nodeName, CraterArmTaint))
      return Promise.all(promises)
    },
    onSuccess: async (_, nodeNames) => {
      await refetchTaskList()
      toast.success(`成功为 ${nodeNames.length} 个 ARM 节点恢复污点`)
      setRestoreArmDialogOpen(false)
      setArmNodesToRestore([])
    },
    onError: (error) => {
      toast.error(`ARM 污点恢复失败: ${error.message}`)
    },
  })

  // 检测需要恢复ARM污点的节点
  const handleRestoreArmTaint = useCallback(() => {
    if (!nodeQuery.data) {
      toast.warning('节点数据尚未加载')
      return
    }

    // 过滤出ARM64架构且没有ARM污点的节点
    const armNodesWithoutTaint = nodeQuery.data.filter((node: INodeBriefInfo) => {
      // 检查节点名称中是否包含arm相关信息 (简化判断逻辑)
      const isArmArchitecture = node.arch.match('arm64')
      const hasArmTaint = node.taints?.some((taint) => taint.startsWith(JoinTaint(CraterArmTaint)))
      return isArmArchitecture && !hasArmTaint
    })

    if (armNodesWithoutTaint.length === 0) {
      toast.info('没有发现需要恢复 ARM 污点的节点')
      return
    }

    setArmNodesToRestore(armNodesWithoutTaint)
    setRestoreArmDialogOpen(true)
  }, [nodeQuery.data])

  const handleOccupation = useCallback(() => {
    const taintContent: IClusterNodeTaint = {
      key: 'crater.raids.io/account',
      value: selectedAccount,
      effect: 'NoSchedule',
    }
    if (isOccupation) {
      addNodeTaint({ nodeName: selectedNode, taintContent })
    } else {
      deleteNodeTaint({ nodeName: selectedNode, taintContent })
    }
    setOpen(false)
  }, [selectedAccount, selectedNode, isOccupation, addNodeTaint, deleteNodeTaint])

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
  const nodeColumns = useMemo(
    () => getNodeColumns((name: string) => getNicknameByName(name) || '', resources, true),
    [getNicknameByName, resources] // 依赖项确保列定义稳定
  )

  const columns = useMemo(
    () => [
      ...nodeColumns,
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => {
          const nodeId = row.original.name
          const nodeStatus = row.original.status
          const taints = row.original.taints
          const unscheduleTaint =
            taints?.some((t) => t === 'node.kubernetes.io/unschedulable:NoSchedule') || false
          const occupiedaccount = taints
            ?.find((t) => t.startsWith('crater.raids.io/account'))
            ?.split('=')[1]
            .split(':')[0]
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
                      setSelectedAccount(occupiedaccount || '')
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
                <DropdownMenuItem onClick={() => handleNodeScheduling(nodeId)}>
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
    [handleNodeScheduling, nodeColumns, t]
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
      >
        <div className="mb-4">
          <Button
            onClick={handleRestoreArmTaint}
            variant="default"
            className="flex items-center gap-2"
          >
            <RotateCcw className="h-4 w-4" />
            同步 ARM 污点
          </Button>
        </div>
      </DataTable>
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
              <AccountSelect value={selectedAccount} onChange={setSelectedAccount} />
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

      {/* 恢复ARM污点Dialog */}
      <RestoreArmTaintDialog
        open={restoreArmDialogOpen}
        onOpenChange={setRestoreArmDialogOpen}
        armNodes={armNodesToRestore}
        onConfirm={(nodeNames) => {
          batchRestoreArmTaint(nodeNames)
        }}
      />
    </>
  )
}
