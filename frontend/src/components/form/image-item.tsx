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
import { Badge } from '@/components/ui/badge'
import { HoverCard, HoverCardContent, HoverCardTrigger } from '@/components/ui/hover-card'

import { ImageInfoResponse } from '@/services/api/imagepack'

import { shortenImageName } from '@/utils/formatter'

import { cn } from '@/lib/utils'

import { TimeDistance } from '../custom/time-distance'
import { ComboboxItem } from './combobox'

export default function ImageItem({ item }: { item: ComboboxItem<ImageInfoResponse> }) {
  const renderContent = () => (
    <div className="w-full">
      {/* 标题行：描述 + 公共/私有标签 */}
      <div className="flex w-full flex-row items-center justify-between gap-2">
        <Badge
          variant={item.detail?.isPublic ? 'default' : 'secondary'}
          className={cn('shrink-0 text-xs', {
            'bg-primary/15 text-primary hover:bg-primary/20': item.detail?.isPublic,
            'bg-purple-500/15 text-purple-600 hover:bg-purple-500/20 dark:text-purple-400':
              !item.detail?.isPublic,
          })}
        >
          {item.detail?.isPublic ? '公共' : '私有'}
        </Badge>
        <h4 className="text-foreground flex-1 truncate font-medium">
          {item.detail?.description || '未命名镜像'}
        </h4>
        {/* 副标题行：上传者 + 时间 */}
        <div className="text-muted-foreground flex items-center gap-1 text-xs">
          <span className="font-medium">{item.detail?.userInfo.nickname}</span>
          <span>•</span>
          <TimeDistance date={item.detail?.createdAt} className="text-xs" />
        </div>
      </div>

      {/* 镜像地址 + 架构 */}
      <div className="flex items-center gap-2">
        <code className="text-muted-foreground flex-1 truncate rounded px-2 py-1 font-mono text-xs">
          {shortenImageName(item.detail?.imageLink ?? item.value)}
        </code>
        {item.detail?.archs && item.detail?.archs.length > 0 && (
          <div className="flex shrink-0 gap-1">
            {item.detail?.archs.map((arch) => {
              const archName = arch.includes('/') ? arch.split('/').slice(1).join('/') : arch
              const isArm = archName.toLowerCase().includes('arm')
              return (
                <span
                  key={arch}
                  className={cn(
                    'inline-flex items-center gap-1 rounded border px-1.5 py-0 font-mono text-xs',
                    {
                      'border-highlight-orange bg-highlight-orange text-highlight-orange dark:border-highlight-orange dark:bg-highlight-orange dark:text-highlight-orange':
                        isArm,
                      'text-highlight-sky border-highlight-sky bg-blue-50 dark:bg-blue-950': !isArm,
                    }
                  )}
                >
                  <span
                    className={cn('size-1 rounded-full', {
                      'bg-highlight-orange': isArm,
                      'bg-highlight-sky': !isArm,
                    })}
                  />
                  {archName}
                </span>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )

  if (item.tags && item.tags.length > 0) {
    return (
      <div>
        {/* 标签 */}
        {item.tags && item.tags.length > 0 && (
          <HoverCard openDelay={100} closeDelay={50}>
            <HoverCardTrigger asChild>{renderContent()}</HoverCardTrigger>
            <HoverCardContent className="w-80">
              <h4 className="text-primary text-sm font-semibold">{item.detail?.description}</h4>
              <div className="mt-2 flex flex-wrap gap-1">
                {item.tags.map((tag) => (
                  <Badge key={tag} variant="outline" className="text-xs">
                    {tag}
                  </Badge>
                ))}
              </div>
            </HoverCardContent>
          </HoverCard>
        )}
      </div>
    )
  }

  return renderContent()
}
