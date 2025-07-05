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
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { ImageInfoResponse } from '@/services/api/imagepack'
import { ComboboxItem } from './Combobox'
import { BoxIcon } from 'lucide-react'
import { shortenImageName } from '@/utils/formatter'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { TimeDistance } from '../custom/TimeDistance'

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
            <div className="flex flex-col items-start gap-0.5">
              {item.detail?.description && (
                <p className="text-foreground">{item.detail?.description ?? item.label}</p>
              )}
              <p className="truncate font-mono text-xs" data-description>
                {shortenImageName(item.detail?.imageLink ?? item.value)}
              </p>
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
