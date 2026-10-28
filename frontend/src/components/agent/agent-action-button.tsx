import { useSetAtom } from 'jotai'
import { BotIcon } from 'lucide-react'
import type { ComponentProps } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import type { AgentActionContext, AgentActionKind } from '@/components/agent/agent-context'

import { atomAgentActionContext, atomAgentEntryRequest, atomAgentPanelOpen } from '@/utils/store'

interface AgentActionButtonProps extends Omit<ComponentProps<typeof Button>, 'onClick'> {
  context: AgentActionContext
  initialMessage?: string
  label?: string
  labelOverride?: string
}

const defaultInitialMessageKeys: Record<AgentActionKind, string> = {
  'choose-job-type': 'craterAgent.action.choose-job-type.initialMessage',
  'choose-template': 'craterAgent.action.choose-template.initialMessage',
  'assist-form': 'craterAgent.action.assist-form.initialMessage',
  'find-platform-resource': 'craterAgent.action.find-platform-resource.initialMessage',
  'recreate-job': 'craterAgent.action.recreate-job.initialMessage',
  'diagnose-job': 'craterAgent.action.diagnose-job.initialMessage',
}

export function AgentActionButton({
  context,
  initialMessage,
  label,
  labelOverride,
  ...props
}: AgentActionButtonProps) {
  const { t } = useTranslation()
  const setContext = useSetAtom(atomAgentActionContext)
  const setEntryRequest = useSetAtom(atomAgentEntryRequest)
  const setOpen = useSetAtom(atomAgentPanelOpen)

  return (
    <Button
      type="button"
      variant="outline"
      {...props}
      onClick={() => {
        const translatedContext = {
          ...context,
          title: t(`craterAgent.action.${context.action}.title`, { defaultValue: context.title }),
          prompt: t(`craterAgent.action.${context.action}.prompt`, {
            defaultValue: context.prompt,
          }),
        }
        setContext(translatedContext)
        setEntryRequest({
          id: crypto.randomUUID(),
          initialMessage: initialMessage ?? t(defaultInitialMessageKeys[context.action]),
        })
        setOpen(true)
      }}
    >
      <BotIcon />
      {labelOverride ??
        t(`craterAgent.action.${context.action}.label`, {
          defaultValue: label ?? t('craterAgent.assist'),
        })}
    </Button>
  )
}
