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

import { useQuery } from '@tanstack/react-query'
import { type FC } from 'react'
import { apiUserCheckImageValid, ImageLinkPair } from '@/services/api/imagepack'
import {
  DialogClose,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { AlertCircle, CheckCheck, CheckCircle, Loader2, Trash2, X } from 'lucide-react'

interface ValidDialogProps {
  linkPairs: ImageLinkPair[]
  onDeleteLinks: (invalidPairs: ImageLinkPair[]) => void
}

export const ValidDialog: FC<ValidDialogProps> = ({ linkPairs, onDeleteLinks }) => {
  const { data, isLoading } = useQuery({
    queryKey: ['checkImageValid', linkPairs],
    queryFn: () => apiUserCheckImageValid({ linkPairs }),
    select: (res) => res.data.data.linkPairs,
  })

  const invalidPairs = data
  const isValid = !isLoading && invalidPairs?.length === 0

  return (
    <>
      {isLoading ? (
        <div className="bg-background/80 flex flex-col items-center justify-center backdrop-blur-xs">
          <Loader2 className="text-primary h-8 w-8 animate-spin" />
          <span className="mt-2 text-sm font-medium">验证镜像中...</span>
        </div>
      ) : (
        <>
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-xl">
              {isValid ? (
                <>
                  <CheckCircle className="h-5 w-5 text-green-500" />
                  <span>所选镜像有效</span>
                </>
              ) : (
                <>
                  <AlertCircle className="text-destructive h-5 w-5" />
                  <span>无效镜像链接</span>
                </>
              )}
            </DialogTitle>
          </DialogHeader>

          <DialogDescription className="pt-2">
            {isValid ? (
              <div className="flex h-16 items-center justify-center rounded-md bg-green-50 px-4 py-3 dark:bg-green-950/30">
                <span className="flex items-center gap-2 text-center text-sm text-green-700 dark:text-green-400">
                  <CheckCheck className="h-4 w-4" />
                  所选镜像链接有效，可以继续操作。
                </span>
              </div>
            ) : (
              <div className="max-h-[240px] space-y-3 overflow-y-auto pr-1">
                {invalidPairs?.map((pair) => (
                  <div className="border-destructive/20 bg-destructive/5 hover:bg-destructive/10 rounded-md border px-4 py-3 transition-all">
                    <div className="space-y-1 text-sm">
                      <div className="group text-muted-foreground relative overflow-hidden text-ellipsis">
                        <span className="text-foreground font-medium">链接: </span>
                        <span className="break-all">{pair.imageLink}</span>
                      </div>
                      <div className="text-muted-foreground">
                        <span className="text-foreground font-medium">描述: </span>
                        {pair.description || (
                          <span className="text-muted-foreground italic">无描述</span>
                        )}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </DialogDescription>

          <DialogFooter>
            <DialogClose asChild>
              <Button variant="outline">
                <X />
                关闭
              </Button>
            </DialogClose>
            {(invalidPairs?.length ?? 0) > 0 && (
              <Button variant="destructive" onClick={() => onDeleteLinks(invalidPairs ?? [])}>
                <Trash2 />
                删除链接
              </Button>
            )}
          </DialogFooter>
        </>
      )}
    </>
  )
}
