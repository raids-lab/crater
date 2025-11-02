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
import { Check, Globe, Lock, X } from 'lucide-react'
import { type FC } from 'react'

import { Button } from '@/components/ui/button'
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import VisibilityBadge, { Visibility } from '@/components/badge/visibility-badge'

import { cn } from '@/lib/utils'

interface StatusDialogProps {
  imageLink: string
  isPublic: boolean
  onChange: () => void
}

export const StatusDialog: FC<StatusDialogProps> = ({ imageLink, isPublic, onChange }) => {
  const currentVisibility = isPublic ? Visibility.Public : Visibility.Private
  const newVisibility = isPublic ? Visibility.Private : Visibility.Public

  // Choose icon based on the new status
  const StatusIcon = isPublic ? Lock : Globe
  const statusColor = isPublic ? 'text-amber-600' : 'text-green-600'
  const bgColor = isPublic ? 'bg-amber-50' : 'bg-green-50'
  const darkBgColor = isPublic ? 'dark:bg-amber-950/30' : 'dark:bg-green-950/30'
  return (
    <>
      <DialogHeader>
        <DialogTitle className="flex items-center gap-2 text-xl">
          <StatusIcon className={`h-5 w-5 ${statusColor}`} />
          <span>更新镜像访问权限</span>
        </DialogTitle>
      </DialogHeader>
      <DialogDescription className="space-y-4 pt-2">
        <div className="bg-muted/50 rounded-md px-4 py-3">
          <p className="text-muted-foreground text-sm">镜像链接</p>
          <p className="mt-1 font-medium break-all">『{imageLink}』</p>
        </div>

        <div className={`rounded-md ${bgColor} ${darkBgColor} p-4`}>
          <div className="flex items-center justify-between">
            <div className="space-y-1">
              <p className="text-sm font-medium">状态变更</p>
              <div className="flex items-center gap-6">
                <div className="flex items-center gap-1.5">
                  <VisibilityBadge visibility={currentVisibility} />
                </div>
                <div className="text-muted-foreground">→</div>
                <div className="flex items-center gap-1.5">
                  <VisibilityBadge visibility={newVisibility} />
                </div>
              </div>
            </div>
            <StatusIcon className={`h-8 w-8 ${statusColor} opacity-20`} />
          </div>
        </div>

        <p className="text-muted-foreground text-sm">
          {isPublic
            ? '设为私有后，只有您可以访问此镜像。'
            : '设为公共后，任何人都可以通过链接访问此镜像。'}
        </p>
      </DialogDescription>
      <DialogFooter>
        <DialogClose asChild>
          <Button variant="outline">
            <X className="size-4" />
            取消
          </Button>
        </DialogClose>
        <Button
          className={cn('flex items-center gap-2', {
            'bg-highlight-amber hover:bg-highlight-amber/90': isPublic,
            'bg-highlight-green hover:bg-highlight-green/90': !isPublic,
          })}
          onClick={onChange}
        >
          <Check />
          确认
        </Button>
      </DialogFooter>
    </>
  )
}
