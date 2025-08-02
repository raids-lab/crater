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
import { LucideIcon } from 'lucide-react'
import { ReactNode } from 'react'

import { cn } from '@/lib/utils'

interface DetailTitleProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: string
  description?: string
  children?: ReactNode
  className?: string
  icon?: LucideIcon
}

const DetailTitle = ({ title, description, children, className, ...props }: DetailTitleProps) => {
  return (
    <div className={cn('flex flex-row items-center justify-between gap-3', className)}>
      <div className="flex items-center space-x-4">
        {props.icon && (
          <div className="bg-muted flex h-20 w-20 items-center justify-center rounded-md border">
            <props.icon className="text-muted-foreground size-10" />
          </div>
        )}
        <div>
          <h1 className="flex items-center gap-2 text-3xl font-bold">{title}</h1>
          <p className="text-muted-foreground">{description}</p>
        </div>
      </div>
      {children}
    </div>
  )
}

export default DetailTitle
