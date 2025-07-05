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

import { Check, Copy } from 'lucide-react'

import { cn } from '@/lib/utils'
import { useCopyToClipboard } from '@/hooks/use-copy-to-clipboard'
import TooltipButton from '../custom/TooltipButton'
import { VariantProps } from 'class-variance-authority'
import { buttonVariants } from '../ui/button'

type CopyButtonProps = React.ComponentProps<'button'> &
  VariantProps<typeof buttonVariants> & {
    content: string
    copyMessage?: string
  }

export function CopyButton({ className, variant, size, content, copyMessage }: CopyButtonProps) {
  const { isCopied, handleCopy } = useCopyToClipboard({
    text: content,
    copyMessage,
  })

  return (
    <TooltipButton
      variant={variant ?? 'ghost'}
      size={size ?? 'icon'}
      className={cn('relative h-6 w-6 cursor-pointer', className)}
      type="button"
      onFocus={(e) => e.preventDefault()}
      aria-label="Copy to clipboard"
      tooltipContent={'复制'}
      onClick={handleCopy}
    >
      <div className="absolute inset-0 flex items-center justify-center">
        <Check
          className={cn(
            'h-4 w-4 transition-transform ease-in-out',
            isCopied ? 'scale-100' : 'scale-0'
          )}
        />
      </div>
      <Copy
        className={cn(
          'h-4 w-4 transition-transform ease-in-out',
          isCopied ? 'scale-0' : 'scale-100'
        )}
      />
    </TooltipButton>
  )
}
