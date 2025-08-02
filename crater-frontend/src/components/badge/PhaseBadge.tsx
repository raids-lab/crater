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
import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

export interface PhaseBadgeData {
  label: string
  color: string
  description: string
}

interface PhaseBadgeProps<T> {
  phase: T
  getPhaseLabel: (phase: T) => PhaseBadgeData
}

export const PhaseBadge = <T,>({ phase, getPhaseLabel }: PhaseBadgeProps<T>) => {
  const data = getPhaseLabel(phase)

  return (
    <TooltipProvider delayDuration={100}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge className={cn('cursor-help border-none', data.color)} variant="outline">
            <div>{data.label}</div>
          </Badge>
        </TooltipTrigger>
        <TooltipContent>
          <p>{data.description}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
