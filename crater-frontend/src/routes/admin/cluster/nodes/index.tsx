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
import { BanIcon, Users, ZapIcon } from 'lucide-react'
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
  NodeStatus,
  apiAddNodeTaint,
  apiDeleteNodeTaint,
  apichangeNodeScheduling,
} from '@/services/api/cluster'
import { queryNodes } from '@/services/query/node'
import { queryResources } from '@/services/query/resource'

import { logger } from '@/utils/loglevel'

import AccountSelect from './-components/AccountList'

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
    mutationFn: apiAddNodeTaint,
    onSuccess: async () => {
      await refetchTaskList()
      toast.success(t('nodeManagement.occupationSuccess'))
    },
    onError: (error) => {
      toast.error(t('nodeManagement.occupationFailed', { error: error.message }))
    },
  })

  const { mutate: deleteNodeTaint } = useMutation({
    mutationFn: apiDeleteNodeTaint,
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
    const taintcontent = `crater.raids.io/account=${selectedAccount}:NoSchedule`
    const taint = {
      name: selectedNode,
      taint: taintcontent,
    }
    if (isOccupation) {
      addNodeTaint(taint)
    } else {
      deleteNodeTaint(taint)
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
                  {nodeStatus === NodeStatus.Ready ? (
                    <BanIcon className="size-4" />
                  ) : (
                    <ZapIcon className="size-4" />
                  )}
                  {nodeStatus === NodeStatus.Ready
                    ? t('nodeManagement.disableScheduling')
                    : t('nodeManagement.enableScheduling')}
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
      />
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
    </>
  )
}
