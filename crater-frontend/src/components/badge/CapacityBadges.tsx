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
import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import {
  IQuota,
  IUserInAccountUpdateReq,
  IUserInAccountUpdateResp,
  apiUpdateUserOutOfProjectList,
} from '@/services/api/account'
import { IResponse } from '@/services/types'

export default function CapacityBadges({
  aid,
  uid,
  role,
  accessmode,
  quota,
  editable,
}: IUserInAccountUpdateReq & { editable: boolean }) {
  const queryClient = useQueryClient()

  const { mutate: updateCapacity, isPending } = useMutation<
    IResponse<IUserInAccountUpdateResp>,
    Error,
    IUserInAccountUpdateReq
  >({
    mutationFn: ({ aid, uid, role, accessmode, quota }) =>
      apiUpdateUserOutOfProjectList({ aid, uid, role, accessmode, quota }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['account', aid, 'users'] })
      toast.success('User updated')
    },
    onError: () => {
      toast.error('Update user failed')
      queryClient.invalidateQueries({ queryKey: ['account', aid, 'users'] })
    },
  })

  const [capability, setCapability] = useState<Record<string, string>>(quota?.capability ?? {})
  const capabilityRef = useRef<Record<string, string>>(capability)

  useEffect(() => {
    setCapability(quota?.capability ?? {})
  }, [quota])

  useEffect(() => {
    capabilityRef.current = capability
  }, [capability])

  // 更新时使用原始 key（包含命名空间）
  const handleUpdateCapacity = useCallback(
    (key: string, value: string) => {
      // 在回调外计算 next，避免在 setState 回调里做副作用
      const next = { ...capabilityRef.current, [key]: value }
      setCapability(next)

      const quotaUpdate: IQuota = {
        guaranteed: quota?.guaranteed,
        deserved: quota?.deserved,
        capability: next,
      }
      updateCapacity({ aid, uid, role, accessmode, quota: quotaUpdate })
    },
    [aid, uid, role, accessmode, quota?.guaranteed, quota?.deserved, updateCapacity]
  )

  // 使用本地状态进行展示，确保立即刷新
  const sortedEntries = useMemo(
    () =>
      Object.entries(capability ?? {}).sort(([a], [b]) => {
        if (a === 'cpu') return -1
        if (b === 'cpu') return 1
        if (a === 'memory') return b === 'cpu' ? 1 : -1
        if (b === 'memory') return a === 'cpu' ? -1 : 1
        return a.localeCompare(b)
      }),
    [capability]
  )

  const CapacityBadge = ({
    keyName,
    displayKey,
    value,
    editable,
    onUpdate,
  }: {
    keyName: string
    displayKey: string
    value: string
    editable: boolean
    onUpdate: (key: string, value: string) => void
  }) => {
    const [editValue, setEditValue] = useState(value)
    const [isOpen, setIsOpen] = useState(false)

    useEffect(() => {
      setEditValue(value)
    }, [value])

    const display =
      displayKey === 'cpu'
        ? `${value}c`
        : displayKey === 'memory'
          ? `${value}`
          : `${displayKey}: ${value}`

    if (!editable) {
      return (
        <Badge variant="secondary" className="font-mono">
          {display}
        </Badge>
      )
    }

    const handleSave = () => {
      onUpdate(keyName, editValue) // 使用原始 key 更新
      setIsOpen(false)
    }

    return (
      <Popover open={isOpen} onOpenChange={setIsOpen}>
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
          <h4 className="font-medium">Configure {displayKey.toUpperCase()}</h4>
          <div className="flex items-center gap-2">
            <Input
              type="text"
              value={editValue}
              onChange={(e) => setEditValue(e.target.value)}
              className="flex-1"
            />
          </div>
          <Button
            size="sm"
            className="w-full"
            disabled={!editValue || editValue === value || isPending}
            onClick={handleSave}
          >
            Save
          </Button>
        </PopoverContent>
      </Popover>
    )
  }

  return (
    <div className="flex flex-col flex-wrap gap-1 lg:flex-row">
      {sortedEntries.map(([rawKey, rawValue]) => {
        const displayKey = rawKey.includes('/') ? rawKey.split('/').slice(1).join('') : rawKey
        return (
          <CapacityBadge
            key={rawKey}
            keyName={rawKey} // 用原始 key 进行更新
            displayKey={displayKey} // 用于展示
            value={rawValue}
            editable={editable}
            onUpdate={handleUpdateCapacity}
          />
        )
      })}
    </div>
  )
}
