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
import type { ChatStatus, UIMessage } from 'ai'

export type AgentResponseOutcome = 'complete' | 'stopped' | 'failed'
export type AgentProgressPhase =
  | AgentResponseOutcome
  | 'waiting'
  | 'processing'
  | 'tools'
  | 'responding'
  | 'confirmation'

export type AgentResponseProgress = {
  phase: AgentProgressPhase
  messageID?: string
}

const confirmationTools = new Set([
  'tool-navigate_to_page',
  'tool-navigate_to_job_detail',
  'tool-propose_form_patch',
])

export function getAgentResponseProgress(
  messages: readonly UIMessage[],
  status: ChatStatus,
  outcome?: AgentResponseOutcome
): AgentResponseProgress | undefined {
  const generating = status === 'submitted' || status === 'streaming'
  if (!generating && !outcome && status !== 'error') return undefined

  // Never attach a new request's progress to an earlier user turn.
  let response: UIMessage | undefined
  if (status !== 'submitted') {
    for (let index = messages.length - 1; index >= 0; index--) {
      const message = messages[index]
      if (message.role === 'user') break
      if (message.role === 'assistant') {
        response = message
        break
      }
    }
  }
  const parts = response?.parts ?? []
  const tools = parts.filter((part) => part.type.startsWith('tool-'))
  let runningTools = 0
  let pendingConfirmation = false
  for (const part of tools) {
    if (!('state' in part)) continue
    if (part.state === 'output-available' || part.state === 'output-error') continue
    if (part.state === 'input-available' && confirmationTools.has(part.type)) {
      pendingConfirmation = true
    } else {
      runningTools++
    }
  }
  const streamingText = parts.some((part) => part.type === 'text' && part.state === 'streaming')
  let phase: AgentProgressPhase
  if (status === 'error' || (!generating && outcome === 'failed')) phase = 'failed'
  else if (!generating && outcome === 'stopped') phase = 'stopped'
  else if (status === 'submitted') phase = 'waiting'
  else if (status === 'streaming') {
    phase = runningTools > 0 ? 'tools' : streamingText ? 'responding' : 'processing'
  } else phase = pendingConfirmation ? 'confirmation' : 'complete'

  return {
    phase,
    messageID: response?.id,
  }
}
