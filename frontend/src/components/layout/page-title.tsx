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
import { FC, ReactNode } from 'react'

import { cn } from '@/lib/utils'

import TipBadge from '../badge/tip-badge'
import { CopyButton } from '../button/copy-button'

interface PageTitleProps {
  title?: string
  description?: string
  descriptionCopiable?: boolean
  children?: ReactNode
  className?: string
  tipComponent?: ReactNode
  tipContent?: ReactNode
}

const PageTitle: FC<PageTitleProps> = ({
  description,
  descriptionCopiable,
  title,
  children,
  className,
  tipComponent,
  tipContent,
}) => {
  return (
    <div
      className={cn(
        'flex min-w-0 flex-col items-start gap-3 sm:flex-row sm:items-center sm:justify-between',
        title ? 'sm:h-12' : '',
        className
      )}
    >
      <div className="min-w-0">
        {title && (
          <div className="flex min-w-0 items-center gap-1.5 text-xl font-bold">
            <p className="min-w-0 truncate">{title}</p>
            {tipComponent}
            {tipContent && <TipBadge title={tipContent} />}
          </div>
        )}
        {description && (
          <p
            className={cn(
              'text-muted-foreground hidden items-center gap-1 md:flex',
              title ? 'text-sm' : 'text-base'
            )}
          >
            {description}
            {descriptionCopiable && <CopyButton content={description} />}
          </p>
        )}
      </div>
      {children && (
        <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto sm:flex-nowrap sm:justify-end">
          {children}
        </div>
      )}
    </div>
  )
}

export default PageTitle
