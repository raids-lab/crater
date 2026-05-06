'use client'

import { MessageCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

interface FloatingAssistantButtonProps {
  onClick: () => void
  hasNewMessage?: boolean
}

export function FloatingAssistantButton({
  onClick,
  hasNewMessage = false,
}: FloatingAssistantButtonProps) {
  const { t } = useTranslation()

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            onClick={onClick}
            size="icon"
            className="fixed right-6 bottom-6 z-50 h-14 w-14 rounded-full shadow-lg transition-all hover:shadow-xl"
          >
            <MessageCircle className="h-6 w-6" />
            {hasNewMessage && (
              <span className="bg-destructive absolute top-0 right-0 h-3 w-3 animate-pulse rounded-full" />
            )}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="left">
          <p>{t('aiops.chat.assistantName')}</p>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
