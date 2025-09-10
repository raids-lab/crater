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
import { BoxIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { ImageInfoResponse } from '@/services/api/imagepack'

import { shortenImageName } from '@/utils/formatter'

import { cn } from '@/lib/utils'

import { TimeDistance } from '../custom/TimeDistance'
import { ComboboxItem } from './Combobox'

export default function ImageItem({ item }: { item: ComboboxItem<ImageInfoResponse> }) {
  const { t } = useTranslation()

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <div className="text-muted-foreground flex w-full items-center gap-3">
            <div
              className={cn('flex size-8 items-center justify-center rounded-full font-normal', {
                'bg-primary/15': item.detail?.isPublic,
                'bg-purple-500/15': !item.detail?.isPublic,
              })}
            >
              <BoxIcon
                className={cn('size-5', {
                  'text-primary': item.detail?.isPublic,
                  'text-purple-500': !item.detail?.isPublic,
                })}
              />
            </div>
            <div className="flex min-w-0 flex-1 flex-col items-start gap-0.5">
              {item.detail?.description && (
                <div className="flex w-full items-center gap-2">
                  {item.detail?.archs && item.detail?.archs.length > 0 && (
                    <div className="flex flex-wrap gap-1">
                      {item.detail?.archs.map((arch) => {
                        const archName = arch.includes('/')
                          ? arch.split('/').slice(1).join('/')
                          : arch
                        const isArm = archName.toLowerCase().includes('arm')
                        return (
                          <span
                            key={arch}
                            className={cn(
                              'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 font-mono text-xs font-medium',
                              {
                                'border-orange-200 bg-orange-50 text-orange-700 dark:border-orange-800 dark:bg-orange-950 dark:text-orange-300':
                                  isArm,
                                'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-800 dark:bg-blue-950 dark:text-blue-300':
                                  !isArm,
                              }
                            )}
                          >
                            <span
                              className={cn('h-1.5 w-1.5 rounded-full', {
                                'bg-orange-500': isArm,
                                'bg-blue-500': !isArm,
                              })}
                            />
                            {archName}
                          </span>
                        )
                      })}
                    </div>
                  )}
                  <p className="text-foreground">{item.detail?.description ?? item.label}</p>
                </div>
              )}
              <p className="mt-1 truncate font-mono text-xs" data-description>
                {shortenImageName(item.detail?.imageLink ?? item.value)}
              </p>
              {/* 显示镜像标签 */}
              {item.tags && item.tags.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {item.tags.map((tag) => (
                    <Badge key={tag} variant="outline" className="text-xs">
                      {tag}
                    </Badge>
                  ))}
                </div>
              )}
            </div>
          </div>
        </TooltipTrigger>
        <TooltipContent className="flex flex-row gap-0.5">
          <p>{item.detail?.userInfo.nickname}</p>
          {t('imageItem.tooltip.createdOn')}
          <TimeDistance date={item.detail?.createdAt} className="text-xs" />
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
