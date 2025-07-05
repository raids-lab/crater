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

import { TooltipProvider, Tooltip, TooltipTrigger, TooltipContent } from '@/components/ui/tooltip'
import { LucideIcon } from 'lucide-react'
import LoadingCircleIcon from '../icon/LoadingCircleIcon'
import { Card, CardHeader, CardTitle } from '../ui/card'
import { cn } from '@/lib/utils'

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
    <Card className={cn('relative pb-0', className)}>
      {isLoading && (
        <div className="absolute top-0 right-0 bottom-0 left-0 z-10 flex items-center justify-center">
          <LoadingCircleIcon />
        </div>
      )}
      <CardHeader>
        <CardTitle className="flex flex-row items-center">
          <TooltipProvider delayDuration={100}>
            <Tooltip>
              <TooltipTrigger className="flex cursor-help flex-row items-center">
                <props.icon className="text-primary mr-1.5 size-5" />
                {cardTitle}
              </TooltipTrigger>
              <TooltipContent>{cardDescription}</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </CardTitle>
      </CardHeader>
      <div className="relative h-52 overflow-hidden px-2">{children}</div>
    </Card>
  )
}

export default PieCard
