'use client'

import {
  CheckCircle,
  ChevronDown,
  AlertTriangle,
  HelpCircle,
  ArrowRight,
  Loader2,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'

import { cn } from '@/lib/utils'

import { ToolCallCard } from './ToolCallCard'

// ── Types ──────────────────────────────────────────────────────────────────────

export interface TimelineEvent {
  id: string
  eventType: 'run_started' | 'handoff' | 'status' | 'tool_call' | 'final_answer'
  agentRole: string
  agentId?: string
  targetAgentRole?: string
  summary?: string
  status?: string
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolStatus?: 'executing' | 'awaiting_confirmation' | 'done' | 'error' | 'cancelled'
  toolResult?: string
  verificationResult?: string
  timestamp: Date
}

export interface AgentTimelineProps {
  turnId: string
  orchestrationMode: 'single_agent' | 'multi_agent'
  events: TimelineEvent[]
  verifierVerdict: 'pass' | 'risk' | 'missing_evidence' | null
  isStreaming: boolean
  defaultOpen?: boolean
}

// ── Role color mapping ─────────────────────────────────────────────────────────

const ROLE_COLORS: Record<string, string> = {
  coordinator: 'bg-violet-100 text-violet-700 dark:bg-violet-900/40 dark:text-violet-300',
  planner: 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300',
  explorer: 'bg-cyan-100 text-cyan-700 dark:bg-cyan-900/40 dark:text-cyan-300',
  executor: 'bg-orange-100 text-orange-700 dark:bg-orange-900/40 dark:text-orange-300',
  verifier: 'bg-purple-100 text-purple-700 dark:bg-purple-900/40 dark:text-purple-300',
  guide: 'bg-gray-100 text-gray-600 dark:bg-gray-800/40 dark:text-gray-400',
  general: 'bg-gray-100 text-gray-600 dark:bg-gray-800/40 dark:text-gray-400',
}

const ROLE_BORDER_COLORS: Record<string, string> = {
  coordinator: 'border-violet-300 dark:border-violet-700',
  planner: 'border-blue-300 dark:border-blue-700',
  explorer: 'border-cyan-300 dark:border-cyan-700',
  executor: 'border-orange-300 dark:border-orange-700',
  verifier: 'border-purple-300 dark:border-purple-700',
  guide: 'border-gray-300 dark:border-gray-600',
  general: 'border-gray-300 dark:border-gray-600',
}

function getRoleColor(role: string): string {
  return ROLE_COLORS[role] || 'bg-gray-100 text-gray-600 dark:bg-gray-800/40 dark:text-gray-400'
}

function getRoleBorderColor(role: string): string {
  return ROLE_BORDER_COLORS[role] || 'border-gray-300 dark:border-gray-600'
}

// ── Verdict badge ──────────────────────────────────────────────────────────────

function VerdictBadge({ verdict }: { verdict: 'pass' | 'risk' | 'missing_evidence' }) {
  const { t } = useTranslation()

  switch (verdict) {
    case 'pass':
      return (
        <Badge className="gap-1 border-green-300 bg-green-50 text-green-700 dark:border-green-700 dark:bg-green-900/30 dark:text-green-300">
          <CheckCircle className="h-3 w-3" />
          {t('aiops.agent.timeline.verdictPass', { defaultValue: '验证通过' })}
        </Badge>
      )
    case 'risk':
      return (
        <Badge className="gap-1 border-orange-300 bg-orange-50 text-orange-700 dark:border-orange-700 dark:bg-orange-900/30 dark:text-orange-300">
          <AlertTriangle className="h-3 w-3" />
          {t('aiops.agent.timeline.verdictRisk', { defaultValue: '有风险' })}
        </Badge>
      )
    case 'missing_evidence':
      return (
        <Badge className="gap-1 border-gray-300 bg-gray-50 text-gray-600 dark:border-gray-600 dark:bg-gray-800/30 dark:text-gray-400">
          <HelpCircle className="h-3 w-3" />
          {t('aiops.agent.timeline.verdictMissing', { defaultValue: '证据不足' })}
        </Badge>
      )
  }
}

// ── Group events into role blocks ──────────────────────────────────────────────

/**
 * A block represents a contiguous segment of stream events emitted by one role.
 * This preserves the actual streaming order:
 *
 *   coordinator(started, handoff->planner)
 *   planner(status, handoff->explorer)
 *   explorer(tool, tool, status, handoff->executor)
 *   executor(status, handoff->verifier)
 *   verifier(status, handoff->coordinator)
 *   coordinator(final_answer)
 */
interface AgentRoleBlock {
  ownerRole: string
  events: TimelineEvent[]
}

function groupIntoBlocks(events: TimelineEvent[]): AgentRoleBlock[] {
  const blocks: AgentRoleBlock[] = []
  let current: AgentRoleBlock | null = null

  for (const event of events) {
    if (!current || current.ownerRole !== event.agentRole) {
      current = {
        ownerRole: event.agentRole,
        events: [],
      }
      blocks.push(current)
    }
    current.events.push(event)
  }

  return blocks
}

function getBlockStatus(block: AgentRoleBlock): string | undefined {
  const displayableStatuses = new Set(['running', 'completed', 'failed', 'awaiting_confirmation'])
  for (let i = block.events.length - 1; i >= 0; i--) {
    const event = block.events[i]
    if (event.eventType === 'final_answer') {
      return 'completed'
    }
    if (event.status && displayableStatuses.has(event.status)) {
      return event.status
    }
  }
  return undefined
}

function getBlockVerificationResult(
  block: AgentRoleBlock
): 'pass' | 'risk' | 'missing_evidence' | undefined {
  for (let i = block.events.length - 1; i >= 0; i--) {
    const verificationResult = block.events[i].verificationResult
    if (
      verificationResult === 'pass' ||
      verificationResult === 'risk' ||
      verificationResult === 'missing_evidence'
    ) {
      return verificationResult
    }
  }
  return undefined
}

// ── Get current active stage ───────────────────────────────────────────────────

function getCurrentStage(events: TimelineEvent[]): string | null {
  for (let i = events.length - 1; i >= 0; i--) {
    const event = events[i]
    if (event.eventType === 'final_answer' && event.agentRole) {
      return event.agentRole
    }
    if (event.eventType === 'handoff' && event.targetAgentRole) {
      return event.targetAgentRole
    }
    if (event.eventType === 'status' && event.agentRole) {
      return event.agentRole
    }
  }
  return 'coordinator'
}

// ── Component ──────────────────────────────────────────────────────────────────

export function AgentTimeline({
  events,
  verifierVerdict,
  isStreaming,
  defaultOpen = false,
}: AgentTimelineProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(defaultOpen)

  const currentStage = getCurrentStage(events)
  const blocks = groupIntoBlocks(events)

  return (
    <div className="w-full max-w-[95%] min-w-0">
      <Collapsible open={open} onOpenChange={setOpen}>
        {/* Collapsed: current stage pill */}
        <CollapsibleTrigger asChild>
          <button
            className={cn(
              'bg-muted/40 hover:bg-muted/60 flex w-full items-center gap-2 rounded-lg border px-3 py-2 text-left text-xs transition-colors',
              isStreaming && 'border-primary/30'
            )}
          >
            <ChevronDown
              className={cn('h-3.5 w-3.5 shrink-0 transition-transform', open && 'rotate-180')}
            />

            {isStreaming && currentStage ? (
              <>
                <Loader2 className="text-primary h-3 w-3 shrink-0 animate-spin" />
                <span
                  className={cn(
                    'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                    getRoleColor(currentStage)
                  )}
                >
                  {currentStage}
                </span>
                <span className="text-muted-foreground text-[10px]">
                  {t('aiops.agent.timeline.running', { defaultValue: '处理中...' })}
                </span>
              </>
            ) : (
              <>
                <span className="text-muted-foreground text-[10px]">
                  {t('aiops.agent.timeline.label', { defaultValue: '协作执行' })}
                </span>
                {currentStage && (
                  <span
                    className={cn(
                      'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                      getRoleColor(currentStage)
                    )}
                  >
                    {currentStage}
                  </span>
                )}
              </>
            )}

            <span className="flex-1" />
            {verifierVerdict && <VerdictBadge verdict={verifierVerdict} />}
          </button>
        </CollapsibleTrigger>

        {/* Expanded: grouped by agent phase */}
        <CollapsibleContent>
          <div className="mt-1 space-y-1.5 pl-1">
            {blocks.map((block, blockIdx) => {
              const blockStatus = getBlockStatus(block)
              const verificationResult = getBlockVerificationResult(block)

              return (
                <div key={blockIdx} className="space-y-1">
                  <div className="flex flex-wrap items-center gap-1.5 py-0.5 text-xs">
                    <span
                      className={cn(
                        'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                        getRoleColor(block.ownerRole)
                      )}
                    >
                      {block.ownerRole}
                    </span>
                    {blockStatus && (
                      <Badge variant="outline" className="h-4 px-1 text-[10px]">
                        {blockStatus}
                      </Badge>
                    )}
                    {verificationResult && <VerdictBadge verdict={verificationResult} />}
                  </div>

                  <div
                    className={cn(
                      'ml-2 space-y-1 border-l-2 pl-3',
                      getRoleBorderColor(block.ownerRole)
                    )}
                  >
                    {block.events.map((event) => {
                      if (event.eventType === 'run_started') {
                        return event.summary ? (
                          <p
                            key={event.id}
                            className="text-muted-foreground py-0.5 text-[11px] leading-relaxed"
                          >
                            {event.summary}
                          </p>
                        ) : null
                      }

                      if (event.eventType === 'status') {
                        return event.summary ? (
                          <p
                            key={event.id}
                            className="text-muted-foreground py-0.5 text-[11px] leading-relaxed"
                          >
                            {event.summary}
                          </p>
                        ) : null
                      }

                      if (event.eventType === 'tool_call') {
                        return (
                          <div key={event.id} className="py-0.5">
                            <ToolCallCard
                              toolName={event.toolName ?? 'unknown'}
                              args={event.toolArgs ?? {}}
                              status={event.toolStatus ?? 'executing'}
                              resultSummary={event.toolResult}
                            />
                          </div>
                        )
                      }

                      if (event.eventType === 'handoff') {
                        return (
                          <div
                            key={event.id}
                            className="bg-muted/30 space-y-1 rounded-md border px-2.5 py-2"
                          >
                            <div className="flex flex-wrap items-center gap-1.5 text-xs">
                              <span
                                className={cn(
                                  'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                                  getRoleColor(event.agentRole)
                                )}
                              >
                                {event.agentRole}
                              </span>
                              {event.targetAgentRole && (
                                <>
                                  <ArrowRight className="text-muted-foreground h-3 w-3 shrink-0" />
                                  <span
                                    className={cn(
                                      'rounded px-1.5 py-0.5 text-[10px] font-medium capitalize',
                                      getRoleColor(event.targetAgentRole)
                                    )}
                                  >
                                    {event.targetAgentRole}
                                  </span>
                                </>
                              )}
                            </div>
                            {event.summary && (
                              <p className="text-muted-foreground text-[11px] leading-relaxed">
                                {event.summary}
                              </p>
                            )}
                          </div>
                        )
                      }

                      if (event.eventType === 'final_answer') {
                        return (
                          <div
                            key={event.id}
                            className="rounded-md border border-green-300 bg-green-50/70 px-2.5 py-2"
                          >
                            <div className="flex items-center gap-1.5 text-[11px] text-green-700">
                              <CheckCircle className="h-3.5 w-3.5 shrink-0" />
                              <span>
                                {t('aiops.agent.timeline.finalDelivered', {
                                  defaultValue: '已整理最终答复并返回给用户',
                                })}
                              </span>
                            </div>
                          </div>
                        )
                      }

                      return null
                    })}
                  </div>
                </div>
              )
            })}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  )
}
