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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useCallback, useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { apiAdminResourceReset } from '@/services/api/resource'
import { IResponse } from '@/services/types'

import useIsAdmin from '@/hooks/use-admin'

interface ResourceBadgesProps {
  namespace?: string
  podName?: string
  nodeName?: string // 添加 nodeName 用于刷新节点pods列表
  resources?: Record<string, string>
  requestResources?: Record<string, string> // 添加 requests 资源
  showEdit?: boolean
}

type UpdateResourceVars = {
  namespace: string
  podName: string
  key: 'cpu' | 'memory'
  numericValue: string
}

// 提取单独的 ResourceBadge 子组件，提高渲染效率
const ResourceBadge = ({
  keyName,
  value,
  editable,
  onUpdate,
}: {
  keyName: string
  value: string
  editable: boolean
  namespace?: string
  podName?: string
  onUpdate: (key: 'cpu' | 'memory', value: string) => void
}) => {
  const [editValue, setEditValue] = useState(value)

  useEffect(() => {
    setEditValue(value)
  }, [value])

  const display =
    keyName === 'cpu' ? `${value}c` : keyName === 'memory' ? `${value}` : `${keyName}: ${value}`

  // 验证内存格式是否为 xxGi
  const isValidMemoryFormat = useMemo(() => {
    if (keyName !== 'memory') return true
    return /^\d+(\.\d+)?Gi$/.test(editValue)
  }, [keyName, editValue])

  const isSaveDisabled = !editValue || !isValidMemoryFormat

  if (!editable) {
    return (
      <Badge variant="secondary" className="font-mono">
        {display}
      </Badge>
    )
  }

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Badge
          className="hover:bg-primary hover:text-primary-foreground cursor-pointer font-mono select-none"
          variant="secondary"
          title="Click to edit resource"
        >
          {display}
        </Badge>
      </PopoverTrigger>
      <PopoverContent className="w-64 space-y-3 p-4">
        <h4 className="font-medium">Configure {keyName.toUpperCase()}</h4>
        <div className="flex items-center gap-2">
          <Input
            type="text"
            value={editValue}
            onChange={(e) => setEditValue(e.target.value)}
            className="flex-1"
          />
        </div>
        <TooltipProvider>
          <Tooltip>
            <TooltipTrigger asChild>
              <div>
                <Button
                  size="sm"
                  className="w-full"
                  disabled={isSaveDisabled}
                  onClick={() => onUpdate(keyName as 'cpu' | 'memory', editValue)}
                >
                  Save
                </Button>
              </div>
            </TooltipTrigger>
            {!isValidMemoryFormat && (
              <TooltipContent>
                <p>内存格式必须为 xGi（例如：7Gi）</p>
              </TooltipContent>
            )}
          </Tooltip>
        </TooltipProvider>
      </PopoverContent>
    </Popover>
  )
}

export default function ResourceBadges({
  namespace,
  podName,
  nodeName,
  resources = {},
  requestResources = {},
  showEdit = false,
}: ResourceBadgesProps) {
  const isAdmin = useIsAdmin()
  const queryClient = useQueryClient()

  const { mutate: updateResource } = useMutation<IResponse<string>, Error, UpdateResourceVars>({
    mutationFn: ({ namespace, podName, key, numericValue }) =>
      apiAdminResourceReset(namespace, podName, key, numericValue),
    onSuccess: (_res, { podName, key, numericValue }) => {
      // 刷新节点的pods列表（管理员和普通用户视图都刷新）
      if (nodeName) {
        queryClient.invalidateQueries({ queryKey: ['nodes', nodeName, 'pods'] })
      }
      queryClient.invalidateQueries({ queryKey: ['podResources', podName] })
      toast.success(`${podName} ${key} updated to ${numericValue}`)
    },
    onError: (_err, { podName, key }) => {
      toast.error(`Failed to update ${key} for ${podName}`)
    },
  })

  // 使用 useCallback 缓存更新函数
  const handleUpdateResource = useCallback(
    (key: 'cpu' | 'memory', value: string) => {
      if (podName && namespace) {
        updateResource({ namespace, podName, key, numericValue: value })
      }
    },
    [namespace, podName, updateResource]
  )

  // 检测是否为绑核模式（requests == limits）
  const isCpuPinned = useMemo(() => {
    const cpuLimit = resources['cpu']
    const cpuRequest = requestResources['cpu']
    return cpuLimit && cpuRequest && cpuLimit === cpuRequest
  }, [resources, requestResources])

  const isMemoryPinned = useMemo(() => {
    const memLimit = resources['memory']
    const memRequest = requestResources['memory']
    return memLimit && memRequest && memLimit === memRequest
  }, [resources, requestResources])

  // 使用 useMemo 缓存排序结果 - 显示 requestResources 的值
  const sortedEntries = useMemo(() => {
    // 优先使用 requestResources 来显示，如果没有则使用 resources
    const displayResources = Object.keys(requestResources).length > 0 ? requestResources : resources
    return Object.entries(displayResources).sort(([a], [b]) => {
      if (a === 'cpu') return -1
      if (b === 'cpu') return 1
      if (a === 'memory') return b === 'cpu' ? 1 : -1
      if (b === 'memory') return a === 'cpu' ? -1 : 1
      return a.localeCompare(b)
    })
  }, [resources, requestResources])

  return (
    <div className="flex flex-col flex-wrap gap-1 lg:flex-row">
      {sortedEntries.map(([rawKey, rawValue]) => {
        const key = rawKey.includes('/') ? rawKey.split('/').slice(1).join('') : rawKey
        // 检查是否可编辑：管理员、CPU/Memory资源、非绑核状态
        let editable = showEdit && isAdmin && (key === 'cpu' || key === 'memory')
        if (editable && key === 'cpu' && isCpuPinned) {
          editable = false
        }
        if (editable && key === 'memory' && isMemoryPinned) {
          editable = false
        }
        return (
          <ResourceBadge
            key={key}
            keyName={key}
            value={rawValue}
            editable={editable}
            namespace={namespace}
            podName={podName}
            onUpdate={handleUpdateResource}
          />
        )
      })}
    </div>
  )
}
