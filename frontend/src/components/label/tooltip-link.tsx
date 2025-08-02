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
import { LinkComponent, createLink } from '@tanstack/react-router'
import { ReactNode } from 'react'
import React from 'react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

interface BasicLinkProps extends React.AnchorHTMLAttributes<HTMLAnchorElement> {
  name: ReactNode
  tooltip: ReactNode
}

const BasicLinkComponent = ({ name, tooltip, className, ...props }: BasicLinkProps) => {
  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        {props.href?.startsWith('http') ? (
          <TooltipTrigger asChild>
            <a
              className={cn('hover:text-primary font-normal no-underline', className)}
              target="_blank"
              rel="noopener noreferrer"
              {...props}
            >
              {name}
            </a>
          </TooltipTrigger>
        ) : (
          <TooltipTrigger>
            <a {...props} className={cn('hover:text-primary font-normal', className)}>
              {name}
            </a>
          </TooltipTrigger>
        )}
        <TooltipContent>{tooltip}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

const CreatedLinkComponent = createLink(BasicLinkComponent)

const TooltipLink: LinkComponent<typeof BasicLinkComponent> = (props) => {
  return <CreatedLinkComponent preload={'intent'} {...props} />
}

export default TooltipLink
