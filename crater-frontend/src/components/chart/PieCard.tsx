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

import { Card, CardHeader, CardTitle } from '@/components/ui/card'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

import LoadingCircleIcon from '../icon/LoadingCircleIcon'

interface PieCardProps extends React.HTMLAttributes<HTMLDivElement> {
  icon: LucideIcon
  cardTitle: string
  cardDescription: string
  isLoading?: boolean
}

const PieCard = ({
  children,
  cardTitle,
  cardDescription,
  isLoading,
  className,
  ...props
}: PieCardProps) => {
  return (
    <Card className={cn('relative gap-0 pb-0', className)}>
      {isLoading && (
        <div className="absolute top-0 right-0 bottom-0 left-0 z-10 flex items-center justify-center">
          <LoadingCircleIcon />
        </div>
      )}
      <CardHeader className="mb-0 pb-0">
        <CardTitle className="flex flex-row items-center">
          <TooltipProvider delayDuration={100}>
            <Tooltip>
              <TooltipTrigger className="text-muted-foreground flex cursor-help flex-row items-center text-sm font-normal">
                <props.icon className="text-muted-foreground mr-1 size-4" />
                {cardTitle}
              </TooltipTrigger>
              <TooltipContent>{cardDescription}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </CardTitle>
      </CardHeader>
      <div className="relative h-56 overflow-hidden px-2">{children}</div>
    </Card>
  )
}

export default PieCard
