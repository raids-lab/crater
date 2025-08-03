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
import { useNavigate } from '@tanstack/react-router'
import { InfoIcon } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { DropdownMenuLabel } from '@/components/ui/dropdown-menu'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

const NodeBadges = ({ nodes }: { nodes?: string[] }) => {
  const navigate = useNavigate()

  if (!nodes || nodes.length === 0 || nodes[0] === '') {
    return <></>
  }

  const isSingleNode = nodes.length === 1
  const handleBadgeClick = () => {
    if (isSingleNode) {
      navigate({ to: `/portal/overview/${nodes[0]}` })
    }
  }

  return (
    <TooltipProvider delayDuration={200}>
      <Tooltip>
        <TooltipTrigger asChild>
          <Badge
            variant="secondary"
            className={cn(
              'font-mono font-normal select-none',
              isSingleNode ? 'cursor-pointer' : 'cursor-help'
            )}
            onClick={handleBadgeClick}
          >
            {isSingleNode ? (
              nodes[0]
            ) : (
              <p>
                {nodes.length}
                <span className="ml-0.5 font-sans">节点</span>
              </p>
            )}
          </Badge>
        </TooltipTrigger>
        <TooltipContent className="p-0">
          {isSingleNode ? (
            <div className="p-2 text-xs">查看节点 {nodes[0]} 详情</div>
          ) : (
            <div className="flex flex-col">
              {nodes
                .sort((a, b) => a.localeCompare(b))
                .map((node, i) => (
                  <div
                    key={node}
                    className={cn('flex flex-col p-1', {
                      'border-t dark:border-slate-700': i > 0,
                    })}
                  >
                    <DropdownMenuLabel className="text-xs">{node}</DropdownMenuLabel>
                    <Button
                      variant="ghost"
                      className="z-10 cursor-pointer justify-start bg-transparent px-2 py-1"
                      onClick={() => navigate({ to: `/portal/overview/${node}` })}
                    >
                      <InfoIcon className="mr-2 h-4 w-4" />
                      <span className="truncate font-normal">节点详情</span>
                    </Button>
                  </div>
                ))}
            </div>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}

export default NodeBadges
