/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { SendIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import { pickAgentWelcomeExamples } from '@/components/agent/agent-welcome-examples'

import type { AgentIdentity } from '@/services/api/agent'

const capabilities = ['guide', 'jobs', 'resources'] as const

export function AgentWelcome({
  identity,
  disabled,
  onSelectExample,
}: {
  identity: AgentIdentity
  disabled: boolean
  onSelectExample: (prompt: string) => void
}) {
  const { t } = useTranslation()
  const [examples] = useState(() => pickAgentWelcomeExamples())
  const visibleCapabilities = identity === 'user' ? [...capabilities, 'forms'] : capabilities
  return (
    <div className="bg-muted min-w-0 flex-1 space-y-3 rounded-2xl p-4 text-sm leading-6 wrap-anywhere">
      <p>{t('craterAgent.welcome')}</p>
      <ul className="list-disc space-y-1 pl-4">
        {visibleCapabilities.map((capability) => (
          <li key={capability}>{t(`craterAgent.intro.capability.${capability}`)}</li>
        ))}
      </ul>
      <div className="space-y-2 border-t pt-3">
        <p className="text-muted-foreground text-xs">{t('craterAgent.intro.examples')}</p>
        <div className="flex flex-wrap gap-2">
          {examples.map((example) => (
            <Button
              key={example}
              type="button"
              variant="outline"
              size="sm"
              className="h-auto max-w-full min-w-0 justify-start py-1.5 text-left text-xs whitespace-normal"
              disabled={disabled}
              onClick={() => onSelectExample(t(`craterAgent.intro.prompt.${example}.${identity}`))}
            >
              <SendIcon className="size-3.5 shrink-0" />
              <span className="min-w-0 wrap-anywhere">
                {t(`craterAgent.intro.example.${example}`)}
              </span>
            </Button>
          ))}
        </div>
      </div>
    </div>
  )
}
