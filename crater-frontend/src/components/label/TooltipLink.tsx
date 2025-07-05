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

import { ReactNode } from 'react'
import { cn } from '@/lib/utils'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { Link } from 'react-router-dom'

interface TooltipLinkProps {
  name: ReactNode
  to: string
  tooltip: ReactNode
  className?: string
}

export default function TooltipLink({ name, to, tooltip, className }: TooltipLinkProps) {
  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        {to.startsWith('http') ? (
          <TooltipTrigger asChild>
            <a
              href={to}
              className={cn('hover:text-primary font-normal no-underline', className)}
              target="_blank"
              rel="noopener noreferrer"
            >
              {name}
            </a>
          </TooltipTrigger>
        ) : (
          <TooltipTrigger>
            <Link className={cn('hover:text-primary font-normal', className)} to={to}>
              {name}
            </Link>
          </TooltipTrigger>
        )}
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
