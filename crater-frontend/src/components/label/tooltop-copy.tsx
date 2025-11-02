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
import { Copy } from 'lucide-react'

import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'

import { cn } from '@/lib/utils'

interface TooltipCopyProps {
  name: string
  className?: string
  copyMessage?: string
  showIcon?: boolean
}

export default function TooltipCopy({
  name,
  className,
  copyMessage,
  showIcon = false,
}: TooltipCopyProps) {
  const { handleCopy } = useCopyToClipboard({
    text: name,
    copyMessage: copyMessage || `已复制 "${name}" 到剪贴板`,
  })

  return (
    <TooltipProvider delayDuration={50}>
      <Tooltip>
        <TooltipTrigger asChild>
          <button
            onClick={handleCopy}
            className={cn(
              'hover:text-primary inline-flex cursor-pointer items-center gap-1 font-normal',
              className
            )}
          >
            {name}
            {showIcon && <Copy className="h-3.5 w-3.5" />}
          </button>
        </TooltipTrigger>
        <TooltipContent>点击以复制</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
