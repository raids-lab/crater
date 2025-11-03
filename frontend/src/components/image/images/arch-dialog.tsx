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
import { Check, Loader2, X } from 'lucide-react'
import { FC, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'

import { ImageDefaultArchs, UpdateImageArch } from '@/services/api/imagepack'

interface ArchDialogProps {
  initialArchs: string[]
  imageID: number
  description: string
  imageLink: string
  onSaveArchs: (updateData: UpdateImageArch) => void
}

export const ArchDialog: FC<ArchDialogProps> = ({
  initialArchs,
  imageID: id,
  description,
  imageLink,
  onSaveArchs,
}) => {
  const [selectedArchs, setSelectedArchs] = useState<string[]>(initialArchs || [])
  const [isSaving, setIsSaving] = useState(false)

  const handleArchToggle = (arch: string) => {
    setSelectedArchs((prev) => {
      if (prev.includes(arch)) {
        return prev.filter((a) => a !== arch)
      } else {
        return [...prev, arch]
      }
    })
  }

  const handleSave = async () => {
    try {
      setIsSaving(true)
      await onSaveArchs({ id, archs: selectedArchs })
    } catch (error) {
      toast.error('保存架构设置失败: ' + error)
    } finally {
      setIsSaving(false)
    }
  }

  return (
    <>
      <DialogHeader>
        <DialogTitle>设置架构</DialogTitle>
      </DialogHeader>
      <DialogDescription className="space-y-4 pt-2">
        <div className="bg-muted/50 rounded-md px-4 py-3">
          <p className="text-foreground text-base font-medium">{description}</p>
          <p className="text-muted-foreground mt-1 font-mono text-sm break-all">『{imageLink}』</p>
        </div>
      </DialogDescription>

      <div className="my-4 space-y-3">
        <p className="text-sm font-medium">选择支持的架构:</p>
        <div className="grid max-h-[300px] grid-cols-2 gap-3 overflow-y-auto">
          {ImageDefaultArchs.map((archOption) => {
            const isSelected = selectedArchs.includes(archOption.value)
            return (
              <div
                key={archOption.value}
                className={`flex cursor-pointer items-center space-x-2 rounded-md border px-2 py-2 transition-colors ${
                  isSelected
                    ? 'border-primary bg-primary/5'
                    : 'border-muted-foreground/20 bg-muted/30'
                }`}
                onClick={() => handleArchToggle(archOption.value)}
              >
                <Checkbox
                  id={archOption.value}
                  checked={isSelected}
                  onCheckedChange={() => handleArchToggle(archOption.value)}
                  className="h-4 w-4"
                />
                <label
                  htmlFor={archOption.value}
                  className={`cursor-pointer ${
                    isSelected ? 'text-foreground' : 'text-muted-foreground'
                  }`}
                  onClick={(e) => e.preventDefault()}
                >
                  {archOption.value}
                </label>
              </div>
            )
          })}
        </div>

        {selectedArchs.length === 0 && (
          <p className="text-destructive text-sm">请至少选择一个架构</p>
        )}
      </div>

      <DialogFooter className="mt-4">
        <DialogClose asChild>
          <Button variant="outline">
            <X className="h-4 w-4" />
            取消
          </Button>
        </DialogClose>
        <Button onClick={handleSave} disabled={isSaving || selectedArchs.length === 0}>
          {isSaving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              保存中...
            </>
          ) : (
            <>
              <Check className="h-4 w-4" />
              确认
            </>
          )}
        </Button>
      </DialogFooter>
    </>
  )
}
