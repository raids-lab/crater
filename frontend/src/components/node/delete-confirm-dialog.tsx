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
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { DialogType } from './node-mark'

interface DeleteConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onConfirm: () => void
  title: string
  itemInfo: {
    key: string
    value: string
    effect?: string
  }
  type: DialogType
}

export function DeleteConfirmDialog({
  open,
  onOpenChange,
  onConfirm,
  title,
  itemInfo,
  type,
}: DeleteConfirmDialogProps) {
  const handleConfirm = () => {
    onConfirm()
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
        </DialogHeader>

        <div className="space-y-3">
          <div className="bg-muted space-y-2 rounded-lg p-3">
            <div className="text-sm">
              <span className="font-medium">Key: </span>
              <span className="font-mono text-sm">{itemInfo.key}</span>
            </div>
            <div className="text-sm">
              <span className="font-medium">Value: </span>
              <span className="font-mono text-sm">{itemInfo.value}</span>
            </div>
            {type === 'taint' && itemInfo.effect && (
              <div className="text-sm">
                <span className="font-medium">Effect: </span>
                <span className="font-mono text-sm">{itemInfo.effect}</span>
              </div>
            )}
          </div>

          <p className="text-muted-foreground text-sm">
            确认删除{' 『 '}
            <span className="font-mono">
              {type === DialogType.LABEL
                ? itemInfo.key + '=' + itemInfo.value
                : type === DialogType.ANNOTATION
                  ? itemInfo.key + ': ' + itemInfo.value
                  : type === DialogType.TAINT
                    ? itemInfo.key + '=' + itemInfo.value + ':' + itemInfo.effect
                    : ''}
            </span>
            {' 』 '}
            的数据项吗？
          </p>
        </div>

        <DialogFooter className="flex gap-2">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            取消
          </Button>
          <Button variant="destructive" onClick={handleConfirm}>
            删除
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
