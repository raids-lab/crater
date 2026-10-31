import { useSetAtom } from 'jotai'
import { BotIcon } from 'lucide-react'
import type { ComponentProps } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { AgentActionContext } from '@/components/agent/agent-context'

import { atomAgentActionContext, atomAgentPanelOpen } from '@/utils/store'

interface AgentActionButtonProps extends Omit<ComponentProps<typeof Button>, 'onClick'> {
  context: AgentActionContext
  label?: string
}

export function AgentActionButton({ context, label, ...props }: AgentActionButtonProps) {
  const { t } = useTranslation()
  const setContext = useSetAtom(atomAgentActionContext)
  const setOpen = useSetAtom(atomAgentPanelOpen)

  return (
    <Button
      type="button"
      variant="outline"
      {...props}
      onClick={() => {
        setContext({
          ...context,
          title: t(`craterAgent.action.${context.action}.title`, { defaultValue: context.title }),
          prompt: t(`craterAgent.action.${context.action}.prompt`, {
            defaultValue: context.prompt,
          }),
        })
        setOpen(true)
      }}
    >
      <BotIcon />
      {t(`craterAgent.action.${context.action}.label`, {
        defaultValue: label ?? t('craterAgent.assist'),
      })}
    </Button>
  )
}
