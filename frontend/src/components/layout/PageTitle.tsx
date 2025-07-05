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
import TipBadge from '../badge/TipBadge'
import { cn } from '@/lib/utils'
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
    <div className={cn('flex h-12 flex-row items-center justify-between gap-3', className)}>
      <div>
        <div className="flex items-center gap-1.5 text-xl font-bold">
          <p>{title}</p>
          {tipComponent}
          {tipContent && <TipBadge title={tipContent} />}
        </div>
        {description && (
          <p className="text-muted-foreground hidden items-center gap-1 text-sm md:flex">
            {description}
            {descriptionCopiable && <CopyButton content={description} />}
          </p>
        )}
      </div>
      {children}
    </div>
  )
}

export default PageTitle
