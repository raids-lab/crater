'use client'

import { useMutation, useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  CheckCircle,
  ChevronDown,
  HelpCircle,
  History,
  Loader2,
  PanelLeftClose,
  Pin,
  Plus,
  Send,
  Sparkles,
  Trash2,
  Users,
  X,
  Zap,
} from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'

import {
  apiDeleteSession,
  apiGetAgentConfigSummary,
  apiGetSessionMessages,
  apiGetSessionTurns,
  apiGetSessionToolCalls,
  apiGetTurnEvents,
  apiListSessions,
  apiParameterUpdate,
  apiPinSession,
  connectAgentChat,
  connectAgentResume,
} from '@/services/api/agent'
import type {
  AgentConfigSummary,
  AgentEvent,
  AgentConfirmationForm,
  AgentMessage,
  AgentSession,
  AgentToolCall,
  AgentTurn,
  BatchConfirmationPayload,
  ParameterReviewPayload,
  PipelineReportPayload,
  ResourceSuggestionPayload,
} from '@/services/api/agent'
import type { AgentSSEEvent } from '@/services/api/agent'
import {
  apiAdminChatMessage,
  apiAdminChatMessageLLM,
  apiChatMessage,
  apiChatMessageLLM,
} from '@/services/api/aiops'
import type { IChatRequest, IChatResponse, IDiagnosis } from '@/services/api/aiops'

import { cn } from '@/lib/utils'

import { AgentTimeline } from './AgentTimeline'
import type { TimelineEvent } from './AgentTimeline'
import { BatchConfirmCard } from './BatchConfirmCard'
import { ConfirmActionCard } from './ConfirmActionCard'
import { ParameterReviewCard } from './ParameterReviewCard'
import { PipelineReportCard } from './PipelineReportCard'
import { ResourceSuggestionCard } from './ResourceSuggestionCard'
import { ThinkingIndicator } from './ThinkingIndicator'
import { ToolCallCard } from './ToolCallCard'

const AGENT_INPUT_MAX_LENGTH = 4000
const AGENT_LAST_SESSION_STORAGE_KEY = 'crater-agent-last-session-id'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  type?: 'text' | 'diagnosis' | 'suggestion'
  data?: IChatResponse['data']
  timestamp: Date
}

type AgentEntryPoint = 'default' | 'node_analysis'

// ── Agent-mode: Two-layer conversation model ──────────────────────────────────

/**
 * Top-level conversation items. In multi_agent mode, MAS execution events are
 * grouped inside a 'timeline' item rather than being flat.
 */
type ConversationItemKind =
  | 'user'
  | 'timeline'
  | 'thinking'
  | 'message'
  | 'tool_call'
  | 'confirmation_required'
  | 'parameter_review'
  | 'resource_suggestion'
  | 'pipeline_report'
  | 'batch_confirmation'
  | 'error'

interface ConversationItem {
  id: string
  kind: ConversationItemKind
  /** For 'user' and 'message' kinds */
  text?: string
  requestId?: string
  requestState?: 'running' | 'done' | 'awaiting_confirmation' | 'failed'
  requestError?: string
  requestSessionId?: string | null
  requestOrchestrationMode?: 'single_agent' | 'multi_agent'
  /** For 'thinking' — may be partial/streaming */
  thinkingContent?: string
  /** For 'timeline' — contains the MAS execution trace */
  turnId?: string
  timelineEvents?: TimelineEvent[]
  timelineVerdict?: 'pass' | 'risk' | 'missing_evidence' | null
  timelineComplete?: boolean
  /** For 'tool_call' (single_agent mode only) */
  toolName?: string
  toolArgs?: Record<string, unknown>
  toolStatus?: 'executing' | 'awaiting_confirmation' | 'done' | 'error' | 'cancelled'
  toolResult?: string
  /** For 'confirmation_required' */
  confirmId?: string
  confirmToolCallId?: string
  confirmAction?: string
  confirmDescription?: string
  confirmInteraction?: string
  confirmForm?: AgentConfirmationForm
  retryRequestId?: string
  /** For agent_event in single_agent (thinking update) */
  agentRole?: string
  /** For 'parameter_review' */
  parameterReview?: ParameterReviewPayload
  /** For 'resource_suggestion' */
  resourceSuggestion?: ResourceSuggestionPayload
  /** For 'pipeline_report' */
  pipelineReport?: PipelineReportPayload
  /** For 'batch_confirmation' */
  batchConfirmation?: BatchConfirmationPayload
  timestamp: Date
}

interface AgentPendingRequest {
  requestId: string
  sessionId: string | null
  message: string
  orchestrationMode: 'single_agent' | 'multi_agent'
  pageContext: {
    route?: string
    url: string
    jobName?: string
    jobStatus?: string
    nodeName?: string
    entryPoint?: AgentEntryPoint
  }
  clientContext?: {
    locale?: string
    timezone?: string
  }
}

interface ActiveAgentRequestState {
  requestId: string
  hasFinalResponse: boolean
  awaitingConfirmation: boolean
}

interface InterruptConfirmState {
  title: string
  description: string
  confirmLabel: string
}

interface SessionDeleteConfirmState {
  sessionId: string
  title: string
}

interface AgentEventPayload {
  turnId?: string
  agentId?: string
  agentRole?: string
  parentAgentId?: string
  targetAgentId?: string
  targetAgentRole?: string
  summary?: string
  status?: string
  title?: string
  eventType?: string
  verificationResult?: string
  content?: string
  toolName?: string
  name?: string
  tool?: string
  toolArgs?: Record<string, unknown>
  args?: Record<string, unknown>
  arguments?: Record<string, unknown>
  toolCallId?: string
  id?: string
  resultSummary?: string
  result?: unknown
  isError?: boolean
  sessionId?: string
  confirmId?: string
  confirm_id?: string
  action?: string
  tool_name?: string
  description?: string
  interaction?: string
  form?: AgentConfirmationForm
}

// ─────────────────────────────────────────────────────────────────────────────

interface AIChatDrawerProps {
  isOpen: boolean
  onClose: () => void
  currentJobName?: string
}

function getAssistantContentForDisplay(
  content: string,
  data: IChatResponse['data'],
  adminHintText: string
) {
  const hintFromData =
    !!data && typeof data === 'object' && 'adminHint' in data && data.adminHint === true
  const showAdminHint = hintFromData || content.includes(adminHintText)
  const cleanedContent = content.replace(adminHintText, '').trim()
  return { showAdminHint, cleanedContent }
}

function isDiagnosisData(data: IChatResponse['data']): data is IDiagnosis {
  return !!data && typeof data === 'object' && 'jobName' in data && 'category' in data
}

function parseAgentJSON(value: unknown): Record<string, unknown> | null {
  if (!value) return null
  if (typeof value === 'object') return value as Record<string, unknown>
  if (typeof value !== 'string') return null
  try {
    return JSON.parse(value) as Record<string, unknown>
  } catch {
    return null
  }
}

function generateAgentRequestId() {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') {
    return crypto.randomUUID()
  }
  return `agent-req-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`
}

function inferAgentEntryPoint(pathname: string): AgentEntryPoint {
  if (/\/admin\/nodes(\/|$)/.test(pathname)) {
    return 'node_analysis'
  }
  return 'default'
}

function getToolStatusFromResult(resultStatus: string): ConversationItem['toolStatus'] {
  switch (resultStatus) {
    case 'success':
      return 'done'
    case 'cancelled':
    case 'rejected':
      return 'cancelled'
    case 'await_confirm':
    case 'confirmation_required':
      return 'awaiting_confirmation'
    default:
      return 'error'
  }
}

function stringifyAgentValue(value: unknown): string | undefined {
  if (value == null) return undefined
  if (typeof value === 'string') return value
  try {
    return JSON.stringify(value)
  } catch {
    return String(value)
  }
}

function getMessageRequestId(message: AgentMessage): string | undefined {
  const metadata = parseAgentJSON(message.metadata)
  const requestId = metadata?.requestId
  return typeof requestId === 'string' ? requestId : undefined
}

function getRequestStateFromTurnStatus(
  status: string | undefined
): ConversationItem['requestState'] | undefined {
  switch (status) {
    case 'running':
      return 'running'
    case 'completed':
      return 'done'
    case 'awaiting_confirmation':
      return 'awaiting_confirmation'
    case 'failed':
      return 'failed'
    default:
      return undefined
  }
}

function getFailedRequestMessage() {
  return 'Agent 未返回最终答复，执行可能已中断。'
}

function getTurnFailureMessage(turn: AgentTurn | undefined): string | undefined {
  if (!turn) return undefined
  const metadata = parseAgentJSON(turn.metadata)
  const errorMessage = metadata?.errorMessage
  return typeof errorMessage === 'string' && errorMessage.trim()
    ? errorMessage
    : undefined
}

function getConversationItemSortWeight(item: ConversationItem): number {
  switch (item.kind) {
    case 'user':
      return 0
    case 'thinking':
      return 1
    case 'timeline':
      return 2
    case 'tool_call':
      return 3
    case 'confirmation_required':
      return 4
    case 'message':
      return 5
    case 'error':
      return 6
    default:
      return 99
  }
}

function formatAgentSessionDate(dateString: string): string {
  const date = new Date(dateString)
  if (Number.isNaN(date.getTime())) {
    return dateString
  }
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
}

function mapRunEventToTimelineEvent(event: AgentEvent): TimelineEvent | null {
  const metadata = parseAgentJSON(event.metadata)
  const timestamp = new Date(event.startedAt || event.createdAt)
  const summary = event.content || event.title || event.eventType

  if (event.eventType === 'agent_run_started') {
    return {
      id: `history-te-${event.id}`,
      eventType: 'run_started',
      agentRole: event.agentRole,
      agentId: event.agentId,
      summary,
      status: event.eventStatus,
      timestamp,
    }
  }

  if (event.eventType === 'agent_handoff') {
    const targetAgentRole =
      typeof metadata?.targetAgentRole === 'string' ? metadata.targetAgentRole : undefined

    return {
      id: `history-te-${event.id}`,
      eventType: 'handoff',
      agentRole: event.agentRole,
      agentId: event.agentId,
      targetAgentRole,
      summary,
      status: event.eventStatus,
      timestamp,
    }
  }

  if (event.eventType === 'agent_status') {
    const verificationResult =
      typeof metadata?.verificationResult === 'string' ? metadata.verificationResult : undefined

    return {
      id: `history-te-${event.id}`,
      eventType: 'status',
      agentRole: event.agentRole,
      agentId: event.agentId,
      summary,
      status: event.eventStatus,
      verificationResult,
      timestamp,
    }
  }

  if (event.eventType === 'final_answer') {
    return {
      id: `history-te-${event.id}`,
      eventType: 'final_answer',
      agentRole: event.agentRole,
      agentId: event.agentId,
      summary,
      status: event.eventStatus,
      timestamp,
    }
  }

  return null
}

function mapToolCallToTimelineEvent(toolCall: AgentToolCall): TimelineEvent {
  return {
    id: toolCall.toolCallId ? `toolcall-${toolCall.toolCallId}` : `history-tool-${toolCall.id}`,
    eventType: 'tool_call',
    agentRole: toolCall.agentRole ?? 'single_agent',
    agentId: toolCall.agentId,
    toolName: toolCall.toolName,
    toolArgs: parseAgentJSON(toolCall.toolArgs) ?? {},
    toolStatus: getToolStatusFromResult(toolCall.resultStatus),
    toolResult: stringifyAgentValue(toolCall.toolResult),
    timestamp: new Date(toolCall.createdAt),
  }
}

function isTimelineEvent(value: TimelineEvent | null): value is TimelineEvent {
  return value !== null
}

function mapSessionHistoryToConversationItems(
  messages: AgentMessage[],
  toolCalls: AgentToolCall[],
  turns: AgentTurn[],
  runEventsByTurn: Record<string, AgentEvent[]>
): ConversationItem[] {
  const latestTurnByRequestId = new Map<string, AgentTurn>()
  for (const turn of turns) {
    if (!turn.requestId) continue
    const existing = latestTurnByRequestId.get(turn.requestId)
    if (
      !existing ||
      new Date(turn.startedAt).getTime() >= new Date(existing.startedAt).getTime()
    ) {
      latestTurnByRequestId.set(turn.requestId, turn)
    }
  }

  const messageItems = messages.map((message) => {
    const requestId = message.role === 'user' ? getMessageRequestId(message) : undefined
    const turn = requestId ? latestTurnByRequestId.get(requestId) : undefined
    const requestState =
      message.role === 'user' ? getRequestStateFromTurnStatus(turn?.status) : undefined
    const failureMessage = turn ? getTurnFailureMessage(turn) : undefined

    return {
      id: `history-${message.id}`,
      kind: message.role === 'user' ? 'user' : 'message',
      text: message.content,
      requestId,
      requestState,
      requestError:
        requestState === 'failed' ? failureMessage ?? getFailedRequestMessage() : undefined,
      requestSessionId: message.sessionId ?? turn?.sessionId ?? null,
      requestOrchestrationMode: turn?.orchestrationMode,
      timestamp: new Date(message.createdAt),
    } satisfies ConversationItem
  })

  // Group multi_agent turns into timeline items
  const timelineItems: ConversationItem[] = turns
    .filter((turn) => turn.orchestrationMode === 'multi_agent')
    .map((turn) => {
      const turnEvents = runEventsByTurn[turn.turnId] ?? []
      const turnToolCalls = toolCalls.filter((toolCall) => toolCall.turnId === turn.turnId)

      const timelineEvents: TimelineEvent[] = [
        ...turnEvents.map(mapRunEventToTimelineEvent).filter(isTimelineEvent),
        ...turnToolCalls.map(mapToolCallToTimelineEvent),
      ].sort((a, b) => a.timestamp.getTime() - b.timestamp.getTime())

      // Extract verifier verdict from events
      const verifierEvent = turnEvents.find((e) => e.agentRole === 'verifier')
      const verifierMetadata = parseAgentJSON(verifierEvent?.metadata)
      const verdict = (typeof verifierMetadata?.verificationResult === 'string'
        ? verifierMetadata.verificationResult
        : verifierEvent?.eventStatus) as 'pass' | 'risk' | 'missing_evidence' | undefined

      return {
        id: `history-timeline-${turn.turnId}`,
        kind: 'timeline',
        turnId: turn.turnId,
        timelineEvents,
        timelineVerdict: verdict ?? null,
        timelineComplete: true,
        timestamp: new Date(turn.startedAt),
      } satisfies ConversationItem
    })

  // Single agent tool calls (not associated with a multi_agent turn) stay flat
  const multiAgentTurnIds = new Set(
    turns.filter((t) => t.orchestrationMode === 'multi_agent').map((t) => t.turnId)
  )
  const singleAgentToolCalls = toolCalls.filter(
    (toolCall) => !toolCall.turnId || !multiAgentTurnIds.has(toolCall.turnId)
  )

  const toolItems = singleAgentToolCalls.flatMap((toolCall) => {
    const toolArgs = parseAgentJSON(toolCall.toolArgs) ?? {}
    const baseTimestamp = new Date(toolCall.createdAt)
    const itemId = `history-tool-${toolCall.id}`
    const resultSummary =
      typeof toolCall.toolResult === 'string'
        ? toolCall.toolResult
        : JSON.stringify(toolCall.toolResult ?? '')
    const items: ConversationItem[] = [
      {
        id: itemId,
        kind: 'tool_call',
        toolName: toolCall.toolName,
        toolArgs,
        toolStatus: getToolStatusFromResult(toolCall.resultStatus),
        toolResult: resultSummary,
        timestamp: baseTimestamp,
      },
    ]
    return items
  })

  const confirmationItems = toolCalls.flatMap((toolCall) => {
    if (
      toolCall.resultStatus !== 'await_confirm' &&
      toolCall.resultStatus !== 'confirmation_required'
    ) {
      return []
    }

    const toolResult = parseAgentJSON(toolCall.toolResult)

    return [
      {
        id: `history-confirm-${toolCall.id}`,
        kind: 'confirmation_required' as const,
        confirmId: String(toolCall.id),
        confirmToolCallId: toolCall.turnId
          ? toolCall.toolCallId
            ? `toolcall-${toolCall.toolCallId}`
            : `history-tool-${toolCall.id}`
          : `history-tool-${toolCall.id}`,
        confirmAction: toolCall.toolName,
        confirmDescription: (toolResult?.description as string) ?? `等待确认 ${toolCall.toolName}`,
        confirmInteraction: (toolResult?.interaction as string) ?? 'approval',
        confirmForm: (toolResult?.form as AgentConfirmationForm) ?? undefined,
        timestamp: new Date(new Date(toolCall.createdAt).getTime() + 1),
      } satisfies ConversationItem,
    ]
  })

  return [...timelineItems, ...messageItems, ...toolItems, ...confirmationItems].sort((a, b) => {
    const timestampDiff = a.timestamp.getTime() - b.timestamp.getTime()
    if (timestampDiff !== 0) return timestampDiff
    return getConversationItemSortWeight(a) - getConversationItemSortWeight(b)
  })
}

// ── Markdown renderer components ──────────────────────────────────────────────

const markdownComponents = {
  p: ({ children }: { children?: React.ReactNode }) => (
    <p className="mb-2 [overflow-wrap:anywhere] break-words last:mb-0">
      {children}
    </p>
  ),
  strong: ({ children }: { children?: React.ReactNode }) => (
    <strong className="text-foreground font-semibold">
      {children}
    </strong>
  ),
  ul: ({ children }: { children?: React.ReactNode }) => (
    <ul className="my-2 list-inside list-disc space-y-1 [overflow-wrap:anywhere] break-words">
      {children}
    </ul>
  ),
  ol: ({ children }: { children?: React.ReactNode }) => (
    <ol className="my-2 list-inside list-decimal space-y-1 [overflow-wrap:anywhere] break-words">
      {children}
    </ol>
  ),
  li: ({ children }: { children?: React.ReactNode }) => (
    <li className="text-sm [overflow-wrap:anywhere] break-words">
      {children}
    </li>
  ),
  pre: ({ children }: { children?: React.ReactNode }) => (
    <pre className="bg-background my-2 max-w-full overflow-x-auto rounded p-2">
      {children}
    </pre>
  ),
  code: ({ children, className }: { children?: React.ReactNode; className?: string }) => {
    const isInline = !className
    return isInline ? (
      <code className="bg-background rounded px-1 py-0.5 font-mono text-xs [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
        {children}
      </code>
    ) : (
      <code className="block min-w-max font-mono text-xs whitespace-pre">
        {children}
      </code>
    )
  },
  h1: ({ children }: { children?: React.ReactNode }) => (
    <h1 className="mb-2 text-lg font-bold [overflow-wrap:anywhere] break-words">{children}</h1>
  ),
  h2: ({ children }: { children?: React.ReactNode }) => (
    <h2 className="mb-2 text-base font-bold [overflow-wrap:anywhere] break-words">{children}</h2>
  ),
  h3: ({ children }: { children?: React.ReactNode }) => (
    <h3 className="mb-1 text-sm font-semibold [overflow-wrap:anywhere] break-words">{children}</h3>
  ),
  h4: ({ children }: { children?: React.ReactNode }) => (
    <h4 className="mb-1 text-sm font-semibold [overflow-wrap:anywhere] break-words">{children}</h4>
  ),
  blockquote: ({ children }: { children?: React.ReactNode }) => (
    <blockquote className="border-primary my-2 border-l-2 pl-3 [overflow-wrap:anywhere] break-words italic">
      {children}
    </blockquote>
  ),
  table: ({ children }: { children?: React.ReactNode }) => (
    <div className="my-2 max-w-full overflow-x-auto">
      <table className="w-full min-w-max border-collapse text-left text-sm">
        {children}
      </table>
    </div>
  ),
  th: ({ children }: { children?: React.ReactNode }) => (
    <th className="border-border bg-background px-3 py-2 font-semibold whitespace-nowrap">
      {children}
    </th>
  ),
  td: ({ children }: { children?: React.ReactNode }) => (
    <td className="border-border border-t px-3 py-2 align-top [overflow-wrap:anywhere] break-words">
      {children}
    </td>
  ),
}

// ── Main Component ────────────────────────────────────────────────────────────

export function AIChatDrawer({ isOpen, onClose, currentJobName }: AIChatDrawerProps) {
  const { t } = useTranslation()
  const adminHintText = t('aiops.chat.adminHint', {
    defaultValue: '管理员账号可前往 Admin 页面使用 Chat 诊断（/admin/aiops）。',
  })

  const smartPrompts = [
    {
      id: 'failure-query',
      category: t('aiops.chat.prompt.category.failureQuery'),
      prompts: [
        { text: t('aiops.chat.prompt.failureReason'), icon: '📊' },
        { text: t('aiops.chat.prompt.whyFail'), icon: '❓' },
        { text: t('aiops.chat.prompt.reduceFailure'), icon: '📉' },
      ],
    },
    {
      id: 'top-issues',
      category: t('aiops.chat.prompt.category.topIssues'),
      prompts: [
        { text: t('aiops.chat.prompt.exit1'), icon: '🐛', hint: t('aiops.chat.prompt.hint.exit1') },
        {
          text: t('aiops.chat.prompt.evicted'),
          icon: '🔧',
          hint: t('aiops.chat.prompt.hint.evicted'),
        },
        {
          text: t('aiops.chat.prompt.mountFailed'),
          icon: '💾',
          hint: t('aiops.chat.prompt.hint.mountFailed'),
        },
        {
          text: t('aiops.chat.prompt.exit127'),
          icon: '⚙️',
          hint: t('aiops.chat.prompt.hint.exit127'),
        },
        { text: t('aiops.chat.prompt.oom'), icon: '💥', hint: t('aiops.chat.prompt.hint.oom') },
      ],
    },
    {
      id: 'job-diagnosis',
      category: t('aiops.chat.prompt.category.jobDiagnosis'),
      prompts: [
        {
          text: t('aiops.chat.prompt.diagnoseJob'),
          icon: '🔍',
          hint: t('aiops.chat.prompt.hint.jobName'),
        },
        { text: t('aiops.chat.prompt.logTips'), icon: '📝' },
      ],
    },
  ]

  const [messages, setMessages] = useState<ChatMessage[]>([
    {
      id: 'initial',
      role: 'assistant',
      content: t('aiops.chat.initialMessage'),
      type: 'text',
      timestamp: new Date(),
    },
  ])
  const [input, setInput] = useState('')
  const [showHelp, setShowHelp] = useState(false)
  const [chatMode, setChatMode] = useState<'rule' | 'llm' | 'agent'>('rule')
  const isAdminRoute =
    typeof window !== 'undefined' && window.location.pathname.startsWith('/admin')
  const [expandedCategories, setExpandedCategories] = useState<string[]>(['top-issues'])
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // ── Agent mode state ──────────────────────────────────────────────────────
  const [conversationItems, setConversationItems] = useState<ConversationItem[]>([])
  const [agentStreaming, setAgentStreaming] = useState(false)
  const [pendingConfirmIds, setPendingConfirmIds] = useState<string[]>([])
  const [agentHistoryLoading, setAgentHistoryLoading] = useState(false)
  const [agentHistoryError, setAgentHistoryError] = useState<string | null>(null)
  const [selectedAgentSessionId, setSelectedAgentSessionId] = useState<string | null>(null)
  const [orchestrationMode, setOrchestrationMode] = useState<'single_agent' | 'multi_agent'>(
    'single_agent'
  )
  const [retryableAgentRequest, setRetryableAgentRequest] = useState<AgentPendingRequest | null>(
    null
  )
  const [failedAgentRequests, setFailedAgentRequests] = useState<
    Record<string, AgentPendingRequest>
  >({})
  const [interruptConfirmState, setInterruptConfirmState] =
    useState<InterruptConfirmState | null>(null)
  const [sessionDeleteConfirmState, setSessionDeleteConfirmState] =
    useState<SessionDeleteConfirmState | null>(null)
  const [sessionActionLoading, setSessionActionLoading] = useState<{
    sessionId: string
    action: 'pin' | 'delete'
  } | null>(null)
  const [sessionPanelOpen, setSessionPanelOpen] = useState(true)
  const agentAbortRef = useRef<AbortController | null>(null)
  const agentHistoryRequestIdRef = useRef(0)
  const agentInteractionVersionRef = useRef(0)
  const lastAgentHistorySessionIdRef = useRef<string | null>(null)
  const lastLoadedAgentSessionIdRef = useRef<string | null>(null)
  const lastAgentRequestRef = useRef<AgentPendingRequest | null>(null)
  const activeAgentRequestStateRef = useRef<ActiveAgentRequestState | null>(null)
  const pendingInterruptActionRef = useRef<(() => void) | null>(null)
  const agentSessionIdRef = useRef<string | null>(null)
  /** Ref to track the current timeline item id being streamed into */
  const currentTimelineIdRef = useRef<string | null>(null)
  const hasActiveAgentTask = agentStreaming || pendingConfirmIds.length > 0

  const { data: agentSessions = [], refetch: refetchAgentSessions } = useQuery<AgentSession[]>({
    queryKey: ['agent-sessions'],
    queryFn: async () => (await apiListSessions()).data,
    enabled: isOpen && chatMode === 'agent',
  })
  const { data: agentConfigSummary } = useQuery<AgentConfigSummary>({
    queryKey: ['agent-config-summary'],
    queryFn: async () => (await apiGetAgentConfigSummary()).data,
    enabled: isOpen && chatMode === 'agent',
  })

  const getPageContext = () => {
    const pathname = window.location.pathname
    const jobMatch = pathname.match(/\/jobs\/detail\/([^/?#]+)/)
    const nodeMatch = pathname.match(/\/nodes\/([^/?#]+)/)
    return {
      route: pathname,
      url: pathname,
      jobName: jobMatch?.[1] ?? currentJobName,
      nodeName: nodeMatch?.[1],
      entryPoint: inferAgentEntryPoint(pathname),
    }
  }

  const getClientContext = () => ({
    locale: typeof navigator !== 'undefined' ? navigator.language : undefined,
    timezone:
      typeof Intl !== 'undefined'
        ? Intl.DateTimeFormat().resolvedOptions().timeZone
        : undefined,
  })

  const cancelAgentStream = useCallback(() => {
    agentAbortRef.current?.abort()
    agentAbortRef.current = null
    setAgentStreaming(false)
  }, [])

  const clearThinkingItems = useCallback(() => {
    setConversationItems((prev) => prev.filter((item) => item.kind !== 'thinking'))
  }, [])

  const rememberFailedAgentRequest = useCallback((request: AgentPendingRequest) => {
    setFailedAgentRequests((prev) => ({
      ...prev,
      [request.requestId]: request,
    }))
  }, [])

  const clearFailedAgentRequest = useCallback((requestId: string | undefined) => {
    if (!requestId) return
    setFailedAgentRequests((prev) => {
      if (!prev[requestId]) return prev
      const next = { ...prev }
      delete next[requestId]
      return next
    })
  }, [])

  const markCurrentTimelineComplete = useCallback(() => {
    const timelineId = currentTimelineIdRef.current
    if (!timelineId) return
    setConversationItems((prev) =>
      prev.map((item) =>
        item.id === timelineId
          ? {
              ...item,
              timelineComplete: true,
            }
          : item
      )
    )
  }, [])

  const updateUserRequestState = useCallback(
    (
      requestId: string | undefined,
      requestState: ConversationItem['requestState'],
      requestError?: string
    ) => {
      if (!requestId) return
      setConversationItems((prev) =>
        prev.map((item) =>
          item.kind === 'user' && item.requestId === requestId
            ? {
                ...item,
                requestState,
                requestError,
              }
            : item
        )
      )
    },
    []
  )

  const resolveConfirmation = useCallback((confirmId: string) => {
    setPendingConfirmIds((prev) => prev.filter((id) => id !== confirmId))
  }, [])

  const bumpAgentInteractionVersion = useCallback(() => {
    agentInteractionVersionRef.current += 1
  }, [])

  const requestInterruptConfirmation = useCallback(
    (
      action: () => void,
      options?: Partial<InterruptConfirmState>
    ) => {
      if (!hasActiveAgentTask) {
        action()
        return
      }

      pendingInterruptActionRef.current = action
      setInterruptConfirmState({
        title: options?.title ?? '中断当前 Agent 执行？',
        description:
          options?.description ??
          '当前 Agent 仍在执行或等待确认。关闭助手、切换模式或切换会话都会中断这轮思考与工具调用。',
        confirmLabel: options?.confirmLabel ?? '中断并继续',
      })
    },
    [hasActiveAgentTask]
  )

  const cancelInterruptConfirmation = useCallback(() => {
    pendingInterruptActionRef.current = null
    setInterruptConfirmState(null)
  }, [])

  const confirmInterruptAndContinue = useCallback(() => {
    const action = pendingInterruptActionRef.current
    pendingInterruptActionRef.current = null
    setInterruptConfirmState(null)
    cancelAgentStream()
    action?.()
  }, [cancelAgentStream])

  const cancelSessionDeleteConfirmation = useCallback(() => {
    setSessionDeleteConfirmState(null)
  }, [])

  const handleToggleSessionPin = useCallback(
    async (session: AgentSession) => {
      if (sessionActionLoading || agentHistoryLoading) return
      const nextPinned = !session.pinnedAt
      setSessionActionLoading({ sessionId: session.sessionId, action: 'pin' })
      try {
        await apiPinSession(session.sessionId, nextPinned)
        await refetchAgentSessions()
      } finally {
        setSessionActionLoading((current) =>
          current?.sessionId === session.sessionId && current.action === 'pin' ? null : current
        )
      }
    },
    [agentHistoryLoading, refetchAgentSessions, sessionActionLoading]
  )

  const requestDeleteAgentSession = useCallback((session: AgentSession) => {
    setSessionDeleteConfirmState({
      sessionId: session.sessionId,
      title: session.title || '未命名',
    })
  }, [])

  const persistAgentSessionId = useCallback((sessionId: string | null) => {
    agentSessionIdRef.current = sessionId
    setSelectedAgentSessionId(sessionId)
    if (typeof window === 'undefined') return
    if (sessionId) {
      window.localStorage.setItem(AGENT_LAST_SESSION_STORAGE_KEY, sessionId)
    } else {
      window.localStorage.removeItem(AGENT_LAST_SESSION_STORAGE_KEY)
    }
  }, [])

  const resetAgentConversation = useCallback(() => {
    bumpAgentInteractionVersion()
    agentHistoryRequestIdRef.current += 1
    cancelAgentStream()
    setAgentHistoryLoading(false)
    setConversationItems([])
    setPendingConfirmIds([])
    setAgentHistoryError(null)
    setRetryableAgentRequest(null)
    setFailedAgentRequests({})
    lastAgentHistorySessionIdRef.current = null
    lastLoadedAgentSessionIdRef.current = null
    lastAgentRequestRef.current = null
    activeAgentRequestStateRef.current = null
    pendingInterruptActionRef.current = null
    setInterruptConfirmState(null)
    currentTimelineIdRef.current = null
    persistAgentSessionId(null)
  }, [bumpAgentInteractionVersion, cancelAgentStream, persistAgentSessionId])

  const performDeleteAgentSession = useCallback(
    async (sessionId: string) => {
      setSessionActionLoading({ sessionId, action: 'delete' })
      try {
        await apiDeleteSession(sessionId)
        const deletingCurrentSession =
          selectedAgentSessionId === sessionId || agentSessionIdRef.current === sessionId
        if (deletingCurrentSession) {
          resetAgentConversation()
        }
        await refetchAgentSessions()
      } finally {
        setSessionActionLoading((current) =>
          current?.sessionId === sessionId && current.action === 'delete' ? null : current
        )
      }
    },
    [refetchAgentSessions, resetAgentConversation, selectedAgentSessionId]
  )

  const confirmDeleteAgentSession = useCallback(() => {
    const pending = sessionDeleteConfirmState
    if (!pending) return
    setSessionDeleteConfirmState(null)

    const runDelete = () => {
      void performDeleteAgentSession(pending.sessionId)
    }

    const deletingCurrentSession =
      selectedAgentSessionId === pending.sessionId || agentSessionIdRef.current === pending.sessionId
    if (deletingCurrentSession && hasActiveAgentTask) {
      requestInterruptConfirmation(runDelete, {
        title: t('aiops.agent.interruptDeleteSessionTitle', {
          defaultValue: '删除当前会话会中断当前执行',
        }),
        description: t('aiops.agent.interruptDeleteSessionDescription', {
          defaultValue: '当前 Agent 仍在执行或等待确认。删除这个会话会立即中断本轮流程，并从历史中移除该会话。',
        }),
        confirmLabel: t('aiops.agent.interruptDeleteSessionConfirm', {
          defaultValue: '中断并删除',
        }),
      })
      return
    }

    runDelete()
  }, [
    hasActiveAgentTask,
    performDeleteAgentSession,
    requestInterruptConfirmation,
    selectedAgentSessionId,
    sessionDeleteConfirmState,
    t,
  ])

  const updateToolCallItem = useCallback(
    (toolCallId: string | undefined, updater: (item: ConversationItem) => ConversationItem) => {
      if (!toolCallId) return
      setConversationItems((prev) =>
        prev.map((item) =>
          item.id === toolCallId || item.id === `toolcall-${toolCallId}` ? updater(item) : item
        )
      )
    },
    []
  )

  const updateTimelineToolEvent = useCallback(
    (toolCallId: string | undefined, patch: Partial<TimelineEvent>) => {
      if (!toolCallId) return
      const candidateIds = new Set([toolCallId, `toolcall-${toolCallId}`])
      setConversationItems((prev) =>
        prev.map((item) => {
          if (item.kind !== 'timeline') return item
          return {
            ...item,
            timelineEvents: item.timelineEvents?.map((event) =>
              candidateIds.has(event.id) ? { ...event, ...patch } : event
            ),
          }
        })
      )
    },
    []
  )

  // ── SSE event handler ─────────────────────────────────────────────────────

  const handleAgentSSEEvent = useCallback(
    (event: AgentSSEEvent, thinkingId: string) => {
      const eventData: AgentEventPayload =
        typeof event.data === 'object' && event.data !== null
          ? (event.data as AgentEventPayload)
          : {}

      switch (event.event) {
        case 'agent_run_started': {
          if (orchestrationMode === 'multi_agent') {
            // Create a new timeline item for this turn
            const timelineId = `timeline-${eventData.turnId || Date.now()}`
            currentTimelineIdRef.current = timelineId
            clearThinkingItems()
            const newEvent: TimelineEvent = {
              id: `te-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
              eventType: 'run_started',
              agentRole: eventData.agentRole ?? 'coordinator',
              agentId: eventData.agentId,
              summary:
                eventData.summary ??
                eventData.content ??
                eventData.description ??
                eventData.title ??
                '',
              status: eventData.status,
              timestamp: new Date(),
            }
            setConversationItems((prev) => [
              ...prev,
              {
                id: timelineId,
                kind: 'timeline',
                turnId: eventData.turnId,
                timelineEvents: [newEvent],
                timelineVerdict: null,
                timelineComplete: false,
                timestamp: new Date(),
              },
            ])
          }
          break
        }

        case 'agent_status':
        case 'agent_handoff': {
          const summary =
            eventData.summary ?? eventData.content ?? eventData.description ?? eventData.title ?? ''
          const role = eventData.agentRole ?? 'single_agent'

          if (orchestrationMode === 'multi_agent') {
            // Append to current timeline
            const timelineId = currentTimelineIdRef.current
            if (timelineId) {
              const newEvent: TimelineEvent = {
                id: `te-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
                eventType: event.event === 'agent_handoff' ? 'handoff' : 'status',
                agentRole: role,
                agentId: eventData.agentId,
                targetAgentRole: eventData.targetAgentRole,
                summary,
                status: eventData.status,
                verificationResult: eventData.verificationResult,
                timestamp: new Date(),
              }
              setConversationItems((prev) =>
                prev.map((item) => {
                  if (item.id !== timelineId) return item
                  const verdict = eventData.verificationResult
                    ? (eventData.verificationResult as 'pass' | 'risk' | 'missing_evidence')
                    : item.timelineVerdict
                  return {
                    ...item,
                    timelineEvents: [...(item.timelineEvents || []), newEvent],
                    timelineVerdict: verdict,
                  }
                })
              )
            }
          } else {
            // Single agent: show as thinking indicator
            if (
              role === 'single_agent' &&
              event.event === 'agent_status' &&
              eventData.status === 'running'
            ) {
              setConversationItems((prev) => {
                const existing = prev.find((item) => item.id === thinkingId)
                if (existing) {
                  return prev.map((item) =>
                    item.id === thinkingId
                      ? { ...item, thinkingContent: summary || 'Agent 思考中...' }
                      : item
                  )
                }
                return [
                  ...prev,
                  {
                    id: thinkingId,
                    kind: 'thinking',
                    thinkingContent: summary || 'Agent 思考中...',
                    timestamp: new Date(),
                  },
                ]
              })
            }
          }
          break
        }

        case 'thinking': {
          const content: string =
            typeof event.data === 'string' ? event.data : (eventData.content ?? '')
          setConversationItems((prev) => {
            const existing = prev.find((i) => i.id === thinkingId)
            if (existing) {
              return prev.map((i) =>
                i.id === thinkingId
                  ? { ...i, thinkingContent: (i.thinkingContent ?? '') + content }
                  : i
              )
            }
            return [
              ...prev,
              {
                id: thinkingId,
                kind: 'thinking',
                thinkingContent: content,
                timestamp: new Date(),
              },
            ]
          })
          break
        }

        case 'tool_call':
        case 'tool_call_started': {
          const toolName: string =
            eventData.toolName ?? eventData.name ?? eventData.tool ?? 'unknown'
          const toolArgs: Record<string, unknown> =
            eventData.toolArgs ?? eventData.args ?? eventData.arguments ?? {}
          const toolCallId =
            eventData.toolCallId ?? eventData.id ?? `tool-${toolName}-${Date.now()}`

          if (orchestrationMode === 'multi_agent') {
            // Append to current timeline
            const timelineId = currentTimelineIdRef.current
            if (timelineId) {
              const newEvent: TimelineEvent = {
                id: `toolcall-${toolCallId}`,
                eventType: 'tool_call',
                agentRole: eventData.agentRole ?? 'explorer',
                toolName,
                toolArgs,
                toolStatus: 'executing',
                timestamp: new Date(),
              }
              setConversationItems((prev) =>
                prev.map((item) => {
                  if (item.id !== timelineId) return item
                  const exists = item.timelineEvents?.some((e) => e.id === newEvent.id)
                  if (exists) {
                    return {
                      ...item,
                      timelineEvents: item.timelineEvents!.map((e) =>
                        e.id === newEvent.id ? { ...e, toolName, toolArgs, toolStatus: 'executing' } : e
                      ),
                    }
                  }
                  return {
                    ...item,
                    timelineEvents: [...(item.timelineEvents || []), newEvent],
                  }
                })
              )
            }
          } else {
            // Single agent: flat tool_call item
            setConversationItems((prev) => {
              const itemId = `toolcall-${toolCallId}`
              const exists = prev.some((item) => item.id === itemId)
              if (exists) {
                return prev.map((item) =>
                  item.id === itemId
                    ? { ...item, toolName, toolArgs, toolStatus: 'executing' as const }
                    : item
                )
              }
              return [
                ...prev,
                {
                  id: itemId,
                  kind: 'tool_call',
                  toolName,
                  toolArgs,
                  toolStatus: 'executing',
                  timestamp: new Date(),
                },
              ]
            })
          }
          break
        }

        case 'tool_result':
        case 'tool_call_completed': {
          const toolName = eventData.toolName ?? eventData.name ?? eventData.tool ?? 'unknown'
          const toolCallId =
            eventData.toolCallId ?? eventData.id ?? `tool-${toolName}-${Date.now()}`
          const result: string =
            typeof eventData.resultSummary === 'string'
              ? eventData.resultSummary
              : typeof eventData.result === 'string'
                ? eventData.result
                : JSON.stringify(eventData.result ?? event.data ?? '')
          const isError: boolean = eventData.isError ?? false

          if (orchestrationMode === 'multi_agent') {
            const timelineId = currentTimelineIdRef.current
            if (timelineId) {
              setConversationItems((prev) =>
                prev.map((item) => {
                  if (item.id !== timelineId) return item
                  const eventId = `toolcall-${toolCallId}`
                  const exists = item.timelineEvents?.some((e) => e.id === eventId)
                  if (!exists) {
                    return {
                      ...item,
                      timelineEvents: [
                        ...(item.timelineEvents || []),
                        {
                          id: eventId,
                          eventType: 'tool_call' as const,
                          agentRole: eventData.agentRole ?? 'explorer',
                          toolName,
                          toolArgs: eventData.toolArgs ?? eventData.args ?? eventData.arguments ?? {},
                          toolStatus: isError ? ('error' as const) : ('done' as const),
                          toolResult: result,
                          timestamp: new Date(),
                        },
                      ],
                    }
                  }
                  return {
                    ...item,
                    timelineEvents: item.timelineEvents!.map((e) =>
                      e.id === eventId
                        ? { ...e, toolStatus: isError ? ('error' as const) : ('done' as const), toolResult: result }
                        : e
                    ),
                  }
                })
              )
            }
          } else {
            // Single agent: update flat tool_call item
            setConversationItems((prev) => {
              const itemId = `toolcall-${toolCallId}`
              const exists = prev.some((item) => item.id === itemId)
              if (!exists) {
                return [
                  ...prev,
                  {
                    id: itemId,
                    kind: 'tool_call',
                    toolName,
                    toolArgs: eventData.toolArgs ?? eventData.args ?? eventData.arguments ?? {},
                    toolStatus: isError ? 'error' : 'done',
                    toolResult: result,
                    timestamp: new Date(),
                  },
                ]
              }
              return prev.map((item) =>
                item.id === itemId
                  ? { ...item, toolStatus: isError ? ('error' as const) : ('done' as const), toolResult: result }
                  : item
              )
            })
          }
          break
        }

        case 'message':
        case 'final_answer': {
          const text: string =
            typeof event.data === 'string' ? event.data : (eventData.content ?? '')
          if (!text.trim()) {
            clearThinkingItems()
            break
          }
          if (activeAgentRequestStateRef.current) {
            activeAgentRequestStateRef.current = {
              ...activeAgentRequestStateRef.current,
              hasFinalResponse: true,
            }
          }
          if (eventData.sessionId) {
            lastLoadedAgentSessionIdRef.current = eventData.sessionId
            persistAgentSessionId(eventData.sessionId)
          }
          clearThinkingItems()
          if (orchestrationMode === 'multi_agent' && currentTimelineIdRef.current) {
            const finalEvent: TimelineEvent = {
              id: `te-${Date.now()}-${Math.random().toString(36).slice(2, 6)}`,
              eventType: 'final_answer',
              agentRole: eventData.agentRole ?? 'coordinator',
              agentId: eventData.agentId,
              summary: text,
              status: 'completed',
              timestamp: new Date(),
            }
            setConversationItems((prev) =>
              prev.map((item) =>
                item.id === currentTimelineIdRef.current
                  ? {
                      ...item,
                      timelineEvents: [...(item.timelineEvents || []), finalEvent],
                      timelineComplete: true,
                    }
                  : item
              )
            )
          }
          // Mark current timeline as complete
          setConversationItems((prev) => [
            ...prev,
            {
              id: `msg-${Date.now()}`,
              kind: 'message',
              text,
              timestamp: new Date(),
            },
          ])
          break
        }

        case 'confirmation_required':
        case 'tool_call_confirmation_required': {
          const confirmId = eventData.confirmId ?? eventData.confirm_id ?? eventData.id ?? ''
          const confirmToolCallId = eventData.toolCallId
          clearThinkingItems()
          if (activeAgentRequestStateRef.current) {
            activeAgentRequestStateRef.current = {
              ...activeAgentRequestStateRef.current,
              awaitingConfirmation: true,
            }
          }
          if (confirmId) {
            setPendingConfirmIds((prev) => (prev.includes(confirmId) ? prev : [...prev, confirmId]))
          }
          updateToolCallItem(confirmToolCallId, (item) => ({
            ...item,
            toolStatus: 'awaiting_confirmation',
            toolResult: eventData.description ?? '等待用户确认后继续执行',
          }))
          updateTimelineToolEvent(confirmToolCallId, {
            toolStatus: 'awaiting_confirmation',
            toolResult: eventData.description ?? '等待用户确认后继续执行',
          })
          const nextItem: ConversationItem = {
            id: `confirm-${Date.now()}`,
            kind: 'confirmation_required',
            confirmId,
            confirmToolCallId,
            confirmAction: eventData.action ?? eventData.toolName ?? eventData.tool_name ?? '',
            confirmDescription: eventData.description ?? '',
            confirmInteraction: eventData.interaction ?? 'approval',
            confirmForm: eventData.form,
            timestamp: new Date(),
          }
          setConversationItems((prev) => {
            const existing = confirmId
              ? prev.find(
                  (item) => item.kind === 'confirmation_required' && item.confirmId === confirmId
                )
              : undefined
            if (!existing) {
              return [...prev, nextItem]
            }
            return prev.map((item) =>
              item.id === existing.id
                ? {
                    ...item,
                    confirmToolCallId,
                    confirmAction: nextItem.confirmAction,
                    confirmDescription: nextItem.confirmDescription,
                    confirmInteraction: nextItem.confirmInteraction,
                    confirmForm: nextItem.confirmForm,
                  }
                : item
            )
          })
          break
        }

        case 'parameter_review': {
          const payload = eventData as unknown as ParameterReviewPayload
          clearThinkingItems()
          setConversationItems((prev) => [
            ...prev,
            {
              id: `param-review-${payload.reviewId || Date.now()}`,
              kind: 'parameter_review',
              parameterReview: payload,
              timestamp: new Date(),
            },
          ])
          break
        }

        case 'resource_suggestion': {
          const payload = eventData as unknown as ResourceSuggestionPayload
          clearThinkingItems()
          setConversationItems((prev) => [
            ...prev,
            {
              id: `resource-sug-${payload.suggestionId || Date.now()}`,
              kind: 'resource_suggestion',
              resourceSuggestion: payload,
              timestamp: new Date(),
            },
          ])
          break
        }

        case 'pipeline_report': {
          const payload = eventData as unknown as PipelineReportPayload
          clearThinkingItems()
          setConversationItems((prev) => [
            ...prev,
            {
              id: `pipeline-rpt-${payload.reportId || Date.now()}`,
              kind: 'pipeline_report',
              pipelineReport: payload,
              timestamp: new Date(),
            },
          ])
          break
        }

        case 'batch_confirmation': {
          const payload = eventData as unknown as BatchConfirmationPayload
          clearThinkingItems()
          setConversationItems((prev) => [
            ...prev,
            {
              id: `batch-confirm-${payload.batchId || Date.now()}`,
              kind: 'batch_confirmation',
              batchConfirmation: payload,
              timestamp: new Date(),
            },
          ])
          break
        }

        default:
          break
      }
    },
    [
      clearThinkingItems,
      orchestrationMode,
      persistAgentSessionId,
      updateTimelineToolEvent,
      updateToolCallItem,
    ]
  )

  const loadAgentSession = useCallback(
    async (sessionId: string) => {
      const requestId = agentHistoryRequestIdRef.current + 1
      const interactionVersion = agentInteractionVersionRef.current
      agentHistoryRequestIdRef.current = requestId
      lastAgentHistorySessionIdRef.current = sessionId
      setSelectedAgentSessionId(sessionId)
      setAgentHistoryLoading(true)
      setAgentHistoryError(null)
      setRetryableAgentRequest(null)
      setFailedAgentRequests({})
      cancelAgentStream()
      setPendingConfirmIds([])
      activeAgentRequestStateRef.current = null
      currentTimelineIdRef.current = null
      try {
        const [messages, toolCalls, turns] = await Promise.all([
          apiGetSessionMessages(sessionId).then((response) => response.data),
          apiGetSessionToolCalls(sessionId).then((response) => response.data),
          apiGetSessionTurns(sessionId).then((response) => response.data),
        ])
        const multiAgentTurns = turns.filter((turn) => turn.orchestrationMode === 'multi_agent')
        const runEventsByTurn = Object.fromEntries(
          await Promise.all(
            multiAgentTurns.map(async (turn) => [
              turn.turnId,
              await apiGetTurnEvents(turn.turnId).then((response) => response.data),
            ])
          )
        ) as Record<string, AgentEvent[]>
        if (
          requestId !== agentHistoryRequestIdRef.current ||
          interactionVersion !== agentInteractionVersionRef.current
        ) {
          return
        }
        const items = mapSessionHistoryToConversationItems(
          messages,
          toolCalls,
          turns,
          runEventsByTurn
        )
        setConversationItems(items)
        setPendingConfirmIds(
          items
            .filter((item) => item.kind === 'confirmation_required' && item.confirmId)
            .map((item) => item.confirmId as string)
        )
        const session = agentSessions.find((entry) => entry.sessionId === sessionId)
        if (session?.lastOrchestrationMode) {
          setOrchestrationMode(session.lastOrchestrationMode)
        }
        lastLoadedAgentSessionIdRef.current = sessionId
        persistAgentSessionId(sessionId)
      } catch (error) {
        if (requestId !== agentHistoryRequestIdRef.current) {
          return
        }
        const message =
          error instanceof Error
            ? error.message
            : t('aiops.common.unknownError', { defaultValue: '未知错误' })
        setAgentHistoryError(message)
      } finally {
        if (requestId === agentHistoryRequestIdRef.current) {
          setAgentHistoryLoading(false)
        }
      }
    },
    [agentSessions, cancelAgentStream, persistAgentSessionId, t]
  )

  const retryLoadAgentSession = useCallback(() => {
    const sessionId = lastAgentHistorySessionIdRef.current
    if (!sessionId || agentHistoryLoading || agentStreaming) return
    void loadAgentSession(sessionId)
  }, [agentHistoryLoading, agentStreaming, loadAgentSession])

  useEffect(() => {
    if (!isOpen) cancelAgentStream()
  }, [cancelAgentStream, isOpen])

  useEffect(() => {
    if (chatMode !== 'agent') cancelAgentStream()
  }, [cancelAgentStream, chatMode])

  useEffect(() => {
    if (!hasActiveAgentTask || typeof window === 'undefined') return

    const handleBeforeUnload = (event: BeforeUnloadEvent) => {
      event.preventDefault()
      event.returnValue = ''
    }

    window.addEventListener('beforeunload', handleBeforeUnload)
    return () => window.removeEventListener('beforeunload', handleBeforeUnload)
  }, [hasActiveAgentTask])

  useEffect(() => {
    if (!isOpen || chatMode !== 'agent' || typeof window === 'undefined') return
    if (agentSessionIdRef.current) return
    const storedSessionId = window.localStorage.getItem(AGENT_LAST_SESSION_STORAGE_KEY)
    if (storedSessionId) {
      persistAgentSessionId(storedSessionId)
    }
  }, [chatMode, isOpen, persistAgentSessionId])

  useEffect(() => {
    if (chatMode !== 'agent') return
    if (selectedAgentSessionId) return
    if (agentConfigSummary?.defaultOrchestrationMode) {
      setOrchestrationMode(agentConfigSummary.defaultOrchestrationMode)
    }
  }, [agentConfigSummary?.defaultOrchestrationMode, chatMode, selectedAgentSessionId])

  useEffect(() => {
    if (!isOpen || chatMode !== 'agent' || conversationItems.length > 0 || agentHistoryLoading)
      return
    const sessionId = agentSessionIdRef.current
    if (!sessionId || lastLoadedAgentSessionIdRef.current === sessionId) return
    if (!agentSessions.some((session) => session.sessionId === sessionId)) return
    void loadAgentSession(sessionId)
  }, [agentHistoryLoading, conversationItems.length, agentSessions, chatMode, isOpen, loadAgentSession])

  const startAgentRequest = useCallback(
    (request: AgentPendingRequest, options?: { appendUserBubble?: boolean }) => {
      if (agentStreaming || pendingConfirmIds.length > 0) return

      bumpAgentInteractionVersion()
      agentHistoryRequestIdRef.current += 1
      const appendUserBubble = options?.appendUserBubble ?? true
      setAgentHistoryLoading(false)
      setAgentHistoryError(null)
      setRetryableAgentRequest(null)
      clearFailedAgentRequest(request.requestId)
      activeAgentRequestStateRef.current = {
        requestId: request.requestId,
        hasFinalResponse: false,
        awaitingConfirmation: false,
      }
      currentTimelineIdRef.current = null

      if (appendUserBubble) {
        setConversationItems((prev) => [
          ...prev,
          {
            id: `user-${Date.now()}`,
            kind: 'user',
            text: request.message,
            requestId: request.requestId,
            requestState: 'running',
            requestError: undefined,
            requestSessionId: request.sessionId,
            requestOrchestrationMode: request.orchestrationMode,
            timestamp: new Date(),
          },
        ])
      }

      lastAgentRequestRef.current = request
      setAgentStreaming(true)

      const thinkingId = `thinking-${Date.now()}`

      const ctrl = connectAgentChat(
        agentSessionIdRef.current ?? request.sessionId,
        request.requestId,
        request.message,
        request.pageContext,
        request.orchestrationMode,
        request.clientContext,
        (event: AgentSSEEvent) => handleAgentSSEEvent(event, thinkingId),
        (err: Error) => {
          agentAbortRef.current = null
          clearThinkingItems()
          markCurrentTimelineComplete()
          updateUserRequestState(request.requestId, 'failed', err.message)
          rememberFailedAgentRequest(request)
          setRetryableAgentRequest(request)
          activeAgentRequestStateRef.current = null
          setAgentStreaming(false)
        },
        () => {
          agentAbortRef.current = null
          clearThinkingItems()
          markCurrentTimelineComplete()
          const activeState = activeAgentRequestStateRef.current
          const isCurrentRequest = activeState?.requestId === request.requestId
          const hasFinalResponse = isCurrentRequest && activeState?.hasFinalResponse
          const awaitingConfirmation = isCurrentRequest && activeState?.awaitingConfirmation

          if (hasFinalResponse) {
            updateUserRequestState(request.requestId, 'done', undefined)
            clearFailedAgentRequest(request.requestId)
            setRetryableAgentRequest(null)
          } else if (awaitingConfirmation) {
            updateUserRequestState(request.requestId, 'awaiting_confirmation', undefined)
            clearFailedAgentRequest(request.requestId)
            setRetryableAgentRequest(null)
          } else {
            updateUserRequestState(
              request.requestId,
              'failed',
              t('aiops.agent.missingFinalAnswer', {
                defaultValue: getFailedRequestMessage(),
              })
            )
            rememberFailedAgentRequest(request)
            setRetryableAgentRequest(request)
          }
          activeAgentRequestStateRef.current = null
          setAgentStreaming(false)
          void refetchAgentSessions()
        },
        (sessionId: string) => {
          lastLoadedAgentSessionIdRef.current = sessionId
          persistAgentSessionId(sessionId)
          if (lastAgentRequestRef.current?.requestId === request.requestId) {
            lastAgentRequestRef.current = {
              ...lastAgentRequestRef.current,
              sessionId,
            }
          }
        }
      )

      agentAbortRef.current = ctrl
    },
    [
      agentStreaming,
      pendingConfirmIds.length,
      bumpAgentInteractionVersion,
      clearFailedAgentRequest,
      persistAgentSessionId,
      rememberFailedAgentRequest,
      refetchAgentSessions,
      clearThinkingItems,
      handleAgentSSEEvent,
      markCurrentTimelineComplete,
      t,
      updateUserRequestState,
    ]
  )

  const startAgentResume = useCallback(
    (confirmId: string, fallbackText: string) => {
      if (!confirmId || agentStreaming) {
        if (fallbackText) {
          setConversationItems((prev) => [
            ...prev,
            {
              id: `confirm-result-${Date.now()}`,
              kind: 'message',
              text: fallbackText,
              timestamp: new Date(),
            },
          ])
        }
        return
      }

      bumpAgentInteractionVersion()
      agentHistoryRequestIdRef.current += 1
      setAgentHistoryLoading(false)
      setAgentHistoryError(null)
      setRetryableAgentRequest(null)
      setAgentStreaming(true)
      const thinkingId = `thinking-resume-${Date.now()}`

      const ctrl = connectAgentResume(
        confirmId,
        (event: AgentSSEEvent) => handleAgentSSEEvent(event, thinkingId),
        (err: Error) => {
          agentAbortRef.current = null
          clearThinkingItems()
          if (fallbackText) {
            setConversationItems((prev) => [
              ...prev,
              {
                id: `confirm-result-${Date.now()}`,
                kind: 'message',
                text: fallbackText,
                timestamp: new Date(),
              },
              {
                id: `confirm-error-${Date.now()}`,
                kind: 'error',
                text: err.message,
                timestamp: new Date(),
              },
            ])
          } else {
            setConversationItems((prev) => [
              ...prev,
              {
                id: `confirm-error-${Date.now()}`,
                kind: 'error',
                text: err.message,
                timestamp: new Date(),
              },
            ])
          }
          setAgentStreaming(false)
        },
        () => {
          agentAbortRef.current = null
          clearThinkingItems()
          setAgentStreaming(false)
          void refetchAgentSessions()
        },
        (sessionId: string) => {
          lastLoadedAgentSessionIdRef.current = sessionId
          persistAgentSessionId(sessionId)
        }
      )

      agentAbortRef.current = ctrl
    },
    [
      agentStreaming,
      bumpAgentInteractionVersion,
      clearThinkingItems,
      handleAgentSSEEvent,
      persistAgentSessionId,
      refetchAgentSessions,
    ]
  )

  const handleAgentSend = (messageText?: string) => {
    const textToSend = messageText ?? input.trim()
    if (!textToSend || agentStreaming || pendingConfirmIds.length > 0) return
    if (textToSend.length > AGENT_INPUT_MAX_LENGTH) {
      setConversationItems((prev) => [
        ...prev,
        {
          id: `err-${Date.now()}`,
          kind: 'error',
          text: `输入内容超过 ${AGENT_INPUT_MAX_LENGTH} 字，请精简后再发送。`,
          timestamp: new Date(),
        },
      ])
      return
    }

    setInput('')
    startAgentRequest(
      {
        requestId: generateAgentRequestId(),
        sessionId: agentSessionIdRef.current,
        message: textToSend,
        orchestrationMode,
        pageContext: getPageContext(),
        clientContext: getClientContext(),
      },
      { appendUserBubble: true }
    )
  }

  const retryAgentRequest = useCallback(
    (source?: string | ConversationItem) => {
      const requestId = typeof source === 'string' ? source : source?.requestId
      const requestFromMessage =
        source && typeof source !== 'string' && source.kind === 'user' && source.text
          ? {
              requestId: source.requestId ?? generateAgentRequestId(),
              sessionId: agentSessionIdRef.current ?? source.requestSessionId ?? null,
              message: source.text,
              orchestrationMode: source.requestOrchestrationMode ?? orchestrationMode,
              pageContext: getPageContext(),
              clientContext: getClientContext(),
            }
          : null
      const request =
        requestFromMessage ??
        (requestId ? failedAgentRequests[requestId] : undefined) ??
        retryableAgentRequest ??
        lastAgentRequestRef.current
      if (!request || agentStreaming || pendingConfirmIds.length > 0) return
      clearFailedAgentRequest(requestId ?? request.requestId)
      const retryRequest: AgentPendingRequest = {
        ...request,
        requestId: generateAgentRequestId(),
        sessionId: agentSessionIdRef.current ?? request.sessionId,
        pageContext: getPageContext(),
        clientContext: getClientContext(),
      }
      startAgentRequest(retryRequest, { appendUserBubble: true })
    },
    [
      agentStreaming,
      clearFailedAgentRequest,
      failedAgentRequests,
      orchestrationMode,
      pendingConfirmIds.length,
      retryableAgentRequest,
      startAgentRequest,
    ]
  )

  const retryAgentRequestInNewSession = useCallback(
    (source?: string | ConversationItem) => {
      const requestId = typeof source === 'string' ? source : source?.requestId
      const requestFromMessage =
        source && typeof source !== 'string' && source.kind === 'user' && source.text
          ? {
              requestId: source.requestId ?? generateAgentRequestId(),
              sessionId: null,
              message: source.text,
              orchestrationMode: source.requestOrchestrationMode ?? orchestrationMode,
              pageContext: getPageContext(),
              clientContext: getClientContext(),
            }
          : null
      const request =
        requestFromMessage ??
        (requestId ? failedAgentRequests[requestId] : undefined) ??
        retryableAgentRequest ??
        lastAgentRequestRef.current
      if (!request || agentStreaming || pendingConfirmIds.length > 0) return
      clearFailedAgentRequest(requestId ?? request.requestId)
      resetAgentConversation()
      startAgentRequest(
        {
          ...request,
          requestId: generateAgentRequestId(),
          sessionId: null,
          pageContext: getPageContext(),
          clientContext: getClientContext(),
        },
        { appendUserBubble: true }
      )
    },
    [
      agentStreaming,
      clearFailedAgentRequest,
      failedAgentRequests,
      orchestrationMode,
      pendingConfirmIds.length,
      resetAgentConversation,
      retryableAgentRequest,
      startAgentRequest,
    ]
  )

  const handleConfirmationSettled = useCallback(
    (
      item: ConversationItem,
      result: { status: string; result?: unknown; message?: string },
      nextStatus: ConversationItem['toolStatus'],
      fallbackText: string
    ) => {
      resolveConfirmation(item.confirmId ?? '')
      const nextToolResult =
        typeof result.result === 'string'
          ? result.result
          : JSON.stringify(result.result ?? result.message ?? fallbackText)
      updateToolCallItem(item.confirmToolCallId, (toolItem) => ({
        ...toolItem,
        toolStatus: nextStatus,
        toolResult: nextToolResult,
      }))
      updateTimelineToolEvent(item.confirmToolCallId, {
        toolStatus: nextStatus,
        toolResult: nextToolResult,
      })
      setConversationItems((prev) =>
        prev.map((ci) =>
          ci.kind === 'user' && ci.requestState === 'awaiting_confirmation'
            ? { ...ci, requestState: 'done' }
            : ci
        )
      )
      startAgentResume(item.confirmId ?? '', result.message ?? fallbackText)
    },
    [resolveConfirmation, startAgentResume, updateTimelineToolEvent, updateToolCallItem]
  )

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, conversationItems])

  // Chat mutation (rule/llm modes)
  const chatMutation = useMutation({
    mutationFn: async (request: IChatRequest) => {
      const response =
        chatMode === 'llm'
          ? await (isAdminRoute ? apiAdminChatMessageLLM(request) : apiChatMessageLLM(request))
          : await (isAdminRoute ? apiAdminChatMessage(request) : apiChatMessage(request))
      return response.data
    },
    onSuccess: (data: IChatResponse) => {
      const assistantMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'assistant',
        content: data.message,
        type: data.type,
        data: data.data,
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, assistantMessage])
    },
    onError: (error: unknown) => {
      let message = error instanceof Error ? error.message : t('aiops.common.unknownError')
      if (error && typeof error === 'object' && 'data' in error) {
        const backend = (error as { data?: { msg?: string } }).data
        if (backend?.msg) {
          message = backend.msg
        }
      }
      const errorMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'assistant',
        content: t('aiops.chat.errorMessage', { message }),
        type: 'text',
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMessage])
    },
  })

  const handleSend = (messageText?: string) => {
    const textToSend = messageText || input.trim()
    if (!textToSend || chatMutation.isPending) return

    let jobName = currentJobName
    const jobNameMatch = textToSend.match(/(?:作业|job)[：:]\s*([a-zA-Z0-9-]+)/i)
    if (jobNameMatch) {
      jobName = jobNameMatch[1]
    }

    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: textToSend,
      timestamp: new Date(),
    }
    setMessages((prev) => [...prev, userMessage])

    chatMutation.mutate({
      message: textToSend,
      jobName: jobName,
    })

    setInput('')
  }

  const toggleCategory = (categoryId: string) => {
    setExpandedCategories((prev) =>
      prev.includes(categoryId) ? prev.filter((c) => c !== categoryId) : [...prev, categoryId]
    )
  }

  const handleAgentDrawerClose = useCallback(() => {
    requestInterruptConfirmation(
      () => {
        cancelAgentStream()
        onClose()
      },
      {
        title: t('aiops.agent.interruptCloseTitle', {
          defaultValue: '关闭助手会中断当前执行',
        }),
        description: t('aiops.agent.interruptCloseDescription', {
          defaultValue:
            '当前 Agent 还在思考或执行工具。关闭助手会立刻中断这轮执行，未完成的答复不会保留。',
        }),
        confirmLabel: t('aiops.agent.interruptCloseConfirm', {
          defaultValue: '中断并关闭',
        }),
      }
    )
  }, [cancelAgentStream, onClose, requestInterruptConfirmation, t])

  const handleAgentModeSwitch = useCallback(
    (nextMode: 'rule' | 'llm' | 'agent') => {
      if (nextMode === chatMode) return
      if (chatMode !== 'agent') {
        setChatMode(nextMode)
        return
      }

      requestInterruptConfirmation(
        () => {
          cancelAgentStream()
          setChatMode(nextMode)
        },
        {
          title: t('aiops.agent.interruptModeTitle', {
            defaultValue: '切换模式会中断当前执行',
          }),
        description: t('aiops.agent.interruptModeDescription', {
          defaultValue:
              '当前 Agent 还在执行或等待确认。切换到其他聊天模式会立刻中断这轮 Agent 流程。',
        }),
          confirmLabel: t('aiops.agent.interruptModeConfirm', {
            defaultValue: '中断并切换',
          }),
        }
      )
    },
    [cancelAgentStream, chatMode, requestInterruptConfirmation, t]
  )

  const handleSelectAgentSession = useCallback(
    (sessionId: string) => {
      if (sessionId === selectedAgentSessionId && !agentHistoryLoading) return
      requestInterruptConfirmation(
        () => {
          void loadAgentSession(sessionId)
        },
        {
          title: t('aiops.agent.interruptSessionTitle', {
            defaultValue: '切换会话会中断当前执行',
          }),
          description: t('aiops.agent.interruptSessionDescription', {
            defaultValue:
              '当前 Agent 还在执行或等待确认。切换到其他历史会话会中断当前这一轮执行并载入新的会话内容。',
          }),
          confirmLabel: t('aiops.agent.interruptSessionConfirm', {
            defaultValue: '中断并切换',
          }),
        }
      )
    },
    [agentHistoryLoading, loadAgentSession, requestInterruptConfirmation, selectedAgentSessionId, t]
  )

  const handleCreateAgentSession = useCallback(() => {
    requestInterruptConfirmation(
      () => {
        resetAgentConversation()
      },
      {
        title: t('aiops.agent.interruptNewSessionTitle', {
          defaultValue: '新会话会中断当前执行',
        }),
        description: t('aiops.agent.interruptNewSessionDescription', {
          defaultValue:
            '当前 Agent 还在执行或等待确认。创建新会话会清空当前上下文并中断这轮执行。',
        }),
        confirmLabel: t('aiops.agent.interruptNewSessionConfirm', {
          defaultValue: '中断并新建',
        }),
      }
    )
  }, [requestInterruptConfirmation, resetAgentConversation, t])

  if (!isOpen) return null

  // ── Agent mode layout ─────────────────────────────────────────────────────

  if (chatMode === 'agent') {
    return (
      <div className="bg-background fixed inset-y-0 right-0 z-50 flex w-full min-w-0 flex-col border-l shadow-2xl sm:w-[540px]">
        <AlertDialog
          open={!!interruptConfirmState}
          onOpenChange={(open) => {
            if (!open) {
              cancelInterruptConfirmation()
            }
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {interruptConfirmState?.title ?? '中断当前 Agent 执行？'}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {interruptConfirmState?.description ??
                  '当前 Agent 仍在执行或等待确认，继续操作会中断这轮流程。'}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('common.cancel', { defaultValue: '取消' })}</AlertDialogCancel>
              <AlertDialogAction onClick={confirmInterruptAndContinue}>
                {interruptConfirmState?.confirmLabel ?? '中断并继续'}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        <AlertDialog
          open={!!sessionDeleteConfirmState}
          onOpenChange={(open) => {
            if (!open) {
              cancelSessionDeleteConfirmation()
            }
          }}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>
                {t('aiops.agent.deleteSessionTitle', {
                  defaultValue: '确认删除会话',
                })}
              </AlertDialogTitle>
              <AlertDialogDescription>
                {t('aiops.agent.deleteSessionDescription', {
                  defaultValue: '确认要删除「{{title}}」会话吗？删除后无法找回。',
                  title: sessionDeleteConfirmState?.title ?? '未命名',
                })}
              </AlertDialogDescription>
            </AlertDialogHeader>
            <AlertDialogFooter>
              <AlertDialogCancel>{t('common.cancel', { defaultValue: '取消' })}</AlertDialogCancel>
              <AlertDialogAction onClick={confirmDeleteAgentSession}>
                {t('aiops.agent.deleteSessionConfirm', {
                  defaultValue: '删除会话',
                })}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
        {/* Header — clean: title + mode tabs + help/close */}
        <div className="from-primary/5 to-primary/10 flex flex-none items-center justify-between border-b bg-gradient-to-r px-4 py-2.5">
          <div className="flex items-center gap-2">
            <Sparkles className="text-primary h-4 w-4" />
            <h3 className="text-sm font-semibold">{t('aiops.chat.assistantName')}</h3>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant="outline"
              size="sm"
              className="h-6 px-2 text-[11px]"
              onClick={() => handleAgentModeSwitch('rule')}
            >
              {t('aiops.chat.mode.rule')}
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-6 px-2 text-[11px]"
              onClick={() => handleAgentModeSwitch('llm')}
            >
              {t('aiops.chat.mode.llm')}
            </Button>
            <Button variant="default" size="sm" className="h-6 px-2 text-[11px]">
              {t('aiops.chat.mode.agent', { defaultValue: 'Agent' })}
            </Button>
            <Dialog open={showHelp} onOpenChange={setShowHelp}>
              <DialogTrigger asChild>
                <Button variant="ghost" size="icon" className="h-6 w-6" aria-label={t('aiops.chat.helpTitle')}>
                  <HelpCircle className="h-3.5 w-3.5" />
                </Button>
              </DialogTrigger>
              <DialogContent showCloseButton={false} className="max-h-[80vh] max-w-2xl overflow-hidden p-0">
                <div className="bg-background sticky top-0 z-10 border-b px-6 py-4 pr-12">
                  <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                      <Sparkles className="text-primary h-5 w-5" />
                      {t('aiops.chat.helpTitle')}
                    </DialogTitle>
                    <DialogDescription>{t('aiops.chat.helpDesc')}</DialogDescription>
                  </DialogHeader>
                  <DialogClose asChild>
                    <Button variant="ghost" size="icon" className="absolute top-4 right-4 h-8 w-8" aria-label={t('common.close')}>
                      <X className="h-4 w-4" />
                    </Button>
                  </DialogClose>
                </div>
                <div className="overflow-y-auto px-6 py-4">
                  <HelpContent />
                </div>
              </DialogContent>
            </Dialog>
            <Button
              variant="ghost"
              size="icon"
              className="h-6 w-6"
              onClick={handleAgentDrawerClose}
              aria-label={t('common.close')}
            >
              <X className="h-3.5 w-3.5" />
            </Button>
          </div>
        </div>
        {hasActiveAgentTask && (
          <div className="border-b bg-amber-50/70 px-4 py-2">
            <p className="text-[11px] leading-relaxed text-amber-800">
              {t('aiops.agent.interruptHint', {
                defaultValue:
                  '当前 Agent 正在执行或等待确认。关闭助手、切换模式或切换会话都会中断这轮流程。',
              })}
            </p>
          </div>
        )}

        {/* Agent body: left session panel + right conversation */}
        <div className="flex min-h-0 flex-1">
          {/* Left: Session list panel — narrow, scrollable */}
          {sessionPanelOpen && (
            <div className="flex w-[200px] shrink-0 flex-col border-r">
              <div className="flex items-center justify-between px-2 py-1.5">
                <span className="text-muted-foreground text-[10px] font-medium">
                  {t('aiops.agent.sessionHistory', { defaultValue: '历史' })}
                </span>
                <div className="flex items-center">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-5 w-5"
                    onClick={handleCreateAgentSession}
                    disabled={agentHistoryLoading}
                    title={t('aiops.agent.newSession', { defaultValue: '新会话' })}
                  >
                    <Plus className="h-3 w-3" />
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-5 w-5"
                    onClick={() => setSessionPanelOpen(false)}
                    title={t('aiops.agent.collapsePanel', { defaultValue: '收起面板' })}
                  >
                    <PanelLeftClose className="h-3 w-3" />
                  </Button>
                </div>
              </div>
              {hasActiveAgentTask && (
                <div className="px-2 pb-1">
                  <p className="rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-[9px] leading-relaxed text-amber-800">
                    {t('aiops.agent.sessionInterruptNotice', {
                      defaultValue: '切换会话或新建会话会中断当前执行。',
                    })}
                  </p>
                </div>
              )}
              <ScrollArea className="min-h-0 flex-1">
                <div className="flex flex-col gap-1 px-1 pb-2">
                  {agentSessions.map((session) => {
                    const isSelected = selectedAgentSessionId === session.sessionId
                    const isPinning =
                      sessionActionLoading?.sessionId === session.sessionId &&
                      sessionActionLoading.action === 'pin'
                    const isDeleting =
                      sessionActionLoading?.sessionId === session.sessionId &&
                      sessionActionLoading.action === 'delete'
                    const sessionTitle =
                      session.title ||
                      t('aiops.agent.untitledSession', { defaultValue: '未命名' })

                    // Truncate title: show first 10 chars + ellipsis if longer
                    const displayTitle =
                      [...sessionTitle].length > 10
                        ? [...sessionTitle].slice(0, 10).join('') + '…'
                        : sessionTitle

                    return (
                      <div
                        key={session.sessionId}
                        className={cn(
                          'group rounded-lg border px-2 py-1.5 transition-colors',
                          isSelected
                            ? 'border-primary/30 bg-primary/10'
                            : 'border-border/60 hover:bg-muted/60',
                          agentHistoryLoading && 'cursor-wait opacity-60',
                          hasActiveAgentTask && !isSelected && 'border-amber-200/70 opacity-60'
                        )}
                      >
                        <button
                          className="w-full min-w-0 text-left"
                          onClick={() => handleSelectAgentSession(session.sessionId)}
                          disabled={agentHistoryLoading || isDeleting}
                        >
                          <div className="flex items-center gap-1">
                            <span
                              className="min-w-0 flex-1 truncate text-[11px] font-medium"
                              title={sessionTitle}
                            >
                              {displayTitle}
                            </span>
                            {isDeleting && (
                              <Loader2 className="text-muted-foreground h-3 w-3 shrink-0 animate-spin" />
                            )}
                          </div>
                          {hasActiveAgentTask && !isSelected && (
                            <p className="mt-0.5 text-[9px] text-amber-700">
                              {t('aiops.agent.sessionSwitchInterruptHint', {
                                defaultValue: '点击后将提示中断当前执行',
                              })}
                            </p>
                          )}
                        </button>
                        <div className="text-muted-foreground mt-1 flex items-center gap-1 text-[9px]">
                          <span className="min-w-0 shrink truncate">
                            {formatAgentSessionDate(session.updatedAt)}
                          </span>
                          <span className="ml-auto flex shrink-0 items-center gap-0.5">
                            <Button
                              variant="ghost"
                              size="icon"
                              className={cn(
                                'h-4 w-4 hover:text-amber-700',
                                session.pinnedAt
                                  ? 'text-amber-600'
                                  : 'text-muted-foreground'
                              )}
                              onClick={(event) => {
                                event.preventDefault()
                                event.stopPropagation()
                                void handleToggleSessionPin(session)
                              }}
                              disabled={agentHistoryLoading || isDeleting}
                              title={
                                session.pinnedAt
                                  ? t('aiops.agent.unpinSession', { defaultValue: '取消置顶' })
                                  : t('aiops.agent.pinSession', { defaultValue: '置顶' })
                              }
                            >
                              {isPinning ? (
                                <Loader2 className="h-2.5 w-2.5 animate-spin" />
                              ) : (
                                <Pin className="h-2.5 w-2.5" />
                              )}
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              className="h-4 w-4 text-muted-foreground hover:text-red-600"
                              onClick={(event) => {
                                event.preventDefault()
                                event.stopPropagation()
                                requestDeleteAgentSession(session)
                              }}
                              disabled={agentHistoryLoading || isPinning || isDeleting}
                              title={t('aiops.agent.deleteSession', { defaultValue: '删除会话' })}
                            >
                              <Trash2 className="h-2.5 w-2.5" />
                            </Button>
                          </span>
                        </div>
                      </div>
                    )
                  })}
                  {agentSessions.length === 0 && (
                    <div className="text-muted-foreground px-2 py-3 text-center text-[10px]">
                      {t('aiops.agent.noSessions', { defaultValue: '暂无历史' })}
                    </div>
                  )}
                </div>
              </ScrollArea>
              {agentHistoryLoading && (
                <p className="text-muted-foreground border-t px-2 py-1 text-[9px]">
                  {t('aiops.agent.loadingSession', { defaultValue: '加载中…' })}
                </p>
              )}
              {agentHistoryError && (
                <div className="border-t px-2 py-1">
                  <p className="text-[9px] text-red-500">{agentHistoryError}</p>
                  {lastAgentHistorySessionIdRef.current && (
                    <Button
                      variant="outline"
                      size="sm"
                      className="mt-0.5 h-4 text-[9px]"
                      onClick={retryLoadAgentSession}
                      disabled={agentHistoryLoading || agentStreaming}
                    >
                      {t('aiops.agent.retryLoadSession', { defaultValue: '重试' })}
                    </Button>
                  )}
                </div>
              )}
            </div>
          )}

          {/* Right: Conversation area */}
          <div className="flex min-w-0 flex-1 flex-col">
            {/* Messages */}
            <ScrollArea className="chat-drawer-scroll-area min-h-0 min-w-0 flex-1 p-4">
              <div className="min-w-0 space-y-4">
                {conversationItems.length === 0 && (
                  <div className="flex min-w-0 justify-start">
                    <div className="bg-muted max-w-[95%] min-w-0 overflow-hidden rounded-lg px-4 py-3">
                      <p className="text-sm [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
                        {t('aiops.agent.initialMessage', {
                          defaultValue:
                            '你好！我是 Crater Agent，可以自主执行操作来帮助你管理作业。请告诉我你需要什么帮助。',
                        })}
                      </p>
                    </div>
                  </div>
                )}
                {conversationItems.map((item) => {
                  if (item.kind === 'user') {
                    const canRetry = item.requestState === 'failed' && !!item.text

                    return (
                      <div key={item.id} className="flex min-w-0 justify-end">
                        <div className="max-w-[85%] min-w-0 overflow-hidden">
                          <div className="bg-primary text-primary-foreground rounded-lg px-4 py-2">
                            <p className="text-sm [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
                              {item.text}
                            </p>
                          </div>
                          {item.requestState === 'failed' && (
                            <div className="mt-1 flex flex-wrap items-center justify-end gap-2">
                              <Badge variant="destructive" className="h-5 gap-1 px-1.5 text-[10px]">
                                <AlertCircle className="h-3 w-3" />
                                {t('aiops.agent.requestFailedShort', {
                                  defaultValue: 'error',
                                })}
                              </Badge>
                              {canRetry && (
                                <Button
                                  variant="outline"
                                  size="sm"
                                  className="h-6 px-2 text-[11px]"
                                  onClick={() => retryAgentRequest(item)}
                                  disabled={agentStreaming || pendingConfirmIds.length > 0}
                                >
                                  {t('aiops.agent.retryLast', {
                                    defaultValue: '重试',
                                  })}
                                </Button>
                              )}
                              {canRetry && (
                                  <Button
                                    variant="ghost"
                                    size="sm"
                                    className="h-6 px-2 text-[11px]"
                                    onClick={() => retryAgentRequestInNewSession(item)}
                                    disabled={agentStreaming || pendingConfirmIds.length > 0}
                                  >
                                    {t('aiops.agent.retryInNewSession', {
                                      defaultValue: '新会话重试',
                                    })}
                                  </Button>
                                )}
                            </div>
                          )}
                          {item.requestState === 'failed' && item.requestError && (
                            <p className="text-destructive mt-1 text-right text-[11px] leading-relaxed">
                              {item.requestError}
                            </p>
                          )}
                          {item.requestState === 'awaiting_confirmation' && (
                            <div className="mt-1 flex justify-end">
                              <Badge variant="outline" className="h-5 gap-1 px-1.5 text-[10px]">
                                <Loader2 className="h-3 w-3 animate-spin" />
                                {t('aiops.agent.awaitingConfirmation', {
                                  defaultValue: '等待确认',
                                })}
                              </Badge>
                            </div>
                          )}
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'thinking') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="bg-muted max-w-[95%] min-w-0 overflow-hidden rounded-lg px-4 py-3">
                          <ThinkingIndicator content={item.thinkingContent} />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'timeline') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <AgentTimeline
                          turnId={item.turnId ?? ''}
                          orchestrationMode="multi_agent"
                          events={item.timelineEvents ?? []}
                          verifierVerdict={item.timelineVerdict ?? null}
                          isStreaming={!item.timelineComplete && agentStreaming}
                        />
                      </div>
                    )
                  }

                  if (item.kind === 'tool_call') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <ToolCallCard
                            toolName={item.toolName ?? 'unknown'}
                            args={item.toolArgs ?? {}}
                            status={item.toolStatus ?? 'executing'}
                            resultSummary={item.toolResult}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'message') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="bg-muted max-w-[95%] min-w-0 overflow-hidden rounded-lg px-4 py-3">
                          <div className="markdown-content w-full min-w-0 text-sm">
                            <ReactMarkdown
                              remarkPlugins={[remarkGfm]}
                              components={markdownComponents}
                            >
                              {item.text ?? ''}
                            </ReactMarkdown>
                          </div>
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'confirmation_required') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <ConfirmActionCard
                            confirmId={item.confirmId ?? ''}
                            action={item.confirmAction ?? ''}
                            description={item.confirmDescription ?? ''}
                            interaction={item.confirmInteraction}
                            form={item.confirmForm}
                            onConfirmed={(result) => {
                              handleConfirmationSettled(
                                item,
                                result,
                                result.status === 'error' ? 'error' : 'done',
                                `${item.confirmAction ?? '操作'} 已执行`
                              )
                            }}
                            onRejected={(result) => {
                              handleConfirmationSettled(
                                item,
                                result,
                                'cancelled',
                                `${item.confirmAction ?? '操作'} 已取消`
                              )
                            }}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'parameter_review' && item.parameterReview) {
                    const pr = item.parameterReview
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <ParameterReviewCard
                            reviewId={pr.reviewId}
                            scenario={pr.scenario}
                            complexity={pr.complexity}
                            step={pr.step}
                            totalSteps={pr.totalSteps}
                            title={pr.title}
                            description={pr.description}
                            parameters={pr.parameters as ParameterReviewPayload['parameters']}
                            onConfirm={(reviewId, parameters) => {
                              apiParameterUpdate(
                                lastLoadedAgentSessionIdRef.current ?? '',
                                reviewId,
                                'confirm',
                                parameters
                              )
                              setConversationItems((prev) =>
                                prev.map((ci) =>
                                  ci.id === item.id
                                    ? {
                                        ...ci,
                                        parameterReview: ci.parameterReview
                                          ? { ...ci.parameterReview, _settled: 'confirmed' as const }
                                          : ci.parameterReview,
                                      }
                                    : ci
                                )
                              )
                            }}
                            onModify={(reviewId, parameters) => {
                              apiParameterUpdate(
                                lastLoadedAgentSessionIdRef.current ?? '',
                                reviewId,
                                'modify',
                                parameters
                              )
                              setConversationItems((prev) =>
                                prev.map((ci) =>
                                  ci.id === item.id
                                    ? {
                                        ...ci,
                                        parameterReview: ci.parameterReview
                                          ? { ...ci.parameterReview, _settled: 'confirmed' as const }
                                          : ci.parameterReview,
                                      }
                                    : ci
                                )
                              )
                            }}
                            settled={(pr as ParameterReviewPayload & { _settled?: string })._settled === 'confirmed' ? 'confirmed' : null}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'resource_suggestion' && item.resourceSuggestion) {
                    const rs = item.resourceSuggestion
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <ResourceSuggestionCard
                            suggestionId={rs.suggestionId}
                            context={rs.context}
                            recommendations={rs.recommendations}
                            tip={rs.tip}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'pipeline_report' && item.pipelineReport) {
                    const rpt = item.pipelineReport
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <PipelineReportCard
                            reportId={rpt.reportId}
                            reportType={rpt.reportType}
                            completedAt={rpt.completedAt}
                            summary={rpt.summary}
                            summaryLabels={rpt.summary_labels}
                            categories={rpt.categories}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'batch_confirmation' && item.batchConfirmation) {
                    const bc = item.batchConfirmation
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="w-full max-w-[95%] min-w-0">
                          <BatchConfirmCard
                            batchId={bc.batchId}
                            action={bc.action}
                            description={bc.description}
                            items={bc.items}
                            onConfirmSelected={(_batchId, _selectedJobNames) => {
                              setConversationItems((prev) =>
                                prev.map((ci) =>
                                  ci.id === item.id
                                    ? {
                                        ...ci,
                                        batchConfirmation: ci.batchConfirmation
                                          ? { ...ci.batchConfirmation, _settled: 'confirmed' as const }
                                          : ci.batchConfirmation,
                                      }
                                    : ci
                                )
                              )
                            }}
                            onRejectAll={(_batchId) => {
                              setConversationItems((prev) =>
                                prev.map((ci) =>
                                  ci.id === item.id
                                    ? {
                                        ...ci,
                                        batchConfirmation: ci.batchConfirmation
                                          ? { ...ci.batchConfirmation, _settled: 'rejected' as const }
                                          : ci.batchConfirmation,
                                      }
                                    : ci
                                )
                              )
                            }}
                            settled={
                              (bc as BatchConfirmationPayload & { _settled?: string })._settled === 'confirmed'
                                ? 'confirmed'
                                : (bc as BatchConfirmationPayload & { _settled?: string })._settled === 'rejected'
                                  ? 'rejected'
                                  : null
                            }
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'error') {
                    return (
                      <div key={item.id} className="flex min-w-0 justify-start">
                        <div className="bg-destructive/5 border-destructive/20 max-w-[82%] min-w-0 overflow-hidden rounded-md border px-3 py-2">
                          <p className="text-destructive text-[11px] [overflow-wrap:anywhere] break-words">
                            {t('aiops.agent.errorMessage', {
                              defaultValue: 'Agent 出错：{{message}}',
                              message: item.text,
                            })}
                          </p>
                        </div>
                      </div>
                    )
                  }

                  return null
                })}
                {agentStreaming &&
                  pendingConfirmIds.length === 0 &&
                  conversationItems[conversationItems.length - 1]?.kind !== 'thinking' && (
                    <div className="flex min-w-0 justify-start">
                      <div className="bg-muted min-w-0 overflow-hidden rounded-lg px-4 py-3">
                        <ThinkingIndicator />
                      </div>
                    </div>
                  )}
                <div ref={messagesEndRef} />
              </div>
            </ScrollArea>

            {/* Input bar with mode toggle */}
            <div className="bg-background flex-none border-t p-3">
              <div className="flex items-center gap-2">
                {/* Session panel toggle */}
                {!sessionPanelOpen && (
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-8 w-8 shrink-0"
                    onClick={() => setSessionPanelOpen(true)}
                    title={t('aiops.agent.showSessions', { defaultValue: '显示历史' })}
                  >
                    <History className="h-3.5 w-3.5" />
                  </Button>
                )}
                {/* Orchestration mode toggle — inline in input bar */}
                <Button
                  variant={orchestrationMode === 'single_agent' ? 'secondary' : 'ghost'}
                  size="icon"
                  className="h-8 w-8 shrink-0"
                  onClick={() => setOrchestrationMode('single_agent')}
                  disabled={agentStreaming || pendingConfirmIds.length > 0}
                  title={t('aiops.agent.singleModeDesc', { defaultValue: '标准模式：单 Agent 执行' })}
                >
                  <Zap className="h-3.5 w-3.5" />
                </Button>
                <Button
                  variant={orchestrationMode === 'multi_agent' ? 'secondary' : 'ghost'}
                  size="icon"
                  className="h-8 w-8 shrink-0"
                  onClick={() => setOrchestrationMode('multi_agent')}
                  disabled={agentStreaming || pendingConfirmIds.length > 0}
                  title={t('aiops.agent.multiModeDesc', { defaultValue: '协作模式：多 Agent 分工' })}
                >
                  <Users className="h-3.5 w-3.5" />
                </Button>
                {/* Input */}
                <Input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  maxLength={AGENT_INPUT_MAX_LENGTH}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter' && !e.shiftKey) {
                      e.preventDefault()
                      handleAgentSend()
                    }
                  }}
                  placeholder={
                    orchestrationMode === 'multi_agent'
                      ? t('aiops.agent.inputPlaceholderMulti', {
                          defaultValue: '协作模式 · 输入问题...',
                        })
                      : t('aiops.agent.inputPlaceholder', {
                          defaultValue: '输入问题...',
                        })
                  }
                  disabled={agentStreaming || pendingConfirmIds.length > 0}
                  className="min-w-0 flex-1"
                />
                <Button
                  onClick={() => handleAgentSend()}
                  disabled={agentStreaming || pendingConfirmIds.length > 0 || !input.trim()}
                  size="icon"
                  className="h-8 w-8 shrink-0"
                  aria-label={t('common.send')}
                >
                  {agentStreaming ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                </Button>
              </div>
              {currentJobName && (
                <div className="text-muted-foreground mt-1.5 flex items-center gap-1 text-[10px]">
                  <span>{t('aiops.chat.currentJob')}</span>
                  <Badge variant="secondary" className="h-4 font-mono text-[10px]">
                    {currentJobName}
                  </Badge>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    )
  }

  // ── Rule/LLM mode layout (unchanged) ──────────────────────────────────────

  return (
    <>
      <div className="bg-background fixed inset-y-0 right-0 z-50 flex w-full min-w-0 flex-col border-l shadow-2xl sm:w-[500px]">
        {/* Header */}
        <div className="from-primary/5 to-primary/10 flex flex-none items-center justify-between border-b bg-gradient-to-r p-4">
          <div className="flex items-center gap-2">
            <Sparkles className="text-primary h-5 w-5" />
            <div>
              <h3 className="font-semibold">{t('aiops.chat.assistantName')}</h3>
              <p className="text-muted-foreground mt-0.5 text-xs">
                {chatMode === 'llm' ? t('aiops.chat.llmBased') : t('aiops.chat.ruleBased')}
              </p>
            </div>
          </div>
          <div className="flex items-center gap-1">
            <Button
              variant={chatMode === 'rule' ? 'default' : 'outline'}
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => setChatMode('rule')}
            >
              {t('aiops.chat.mode.rule')}
            </Button>
            <Button
              variant={chatMode === 'llm' ? 'default' : 'outline'}
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => setChatMode('llm')}
            >
              {t('aiops.chat.mode.llm')}
            </Button>
            <Button
              variant="outline"
              size="sm"
              className="h-7 px-2 text-xs"
              onClick={() => setChatMode('agent')}
            >
              {t('aiops.chat.mode.agent', { defaultValue: 'Agent' })}
            </Button>
            <Dialog open={showHelp} onOpenChange={setShowHelp}>
              <DialogTrigger asChild>
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-8 w-8"
                  aria-label={t('aiops.chat.helpTitle')}
                >
                  <HelpCircle className="h-4 w-4" />
                </Button>
              </DialogTrigger>
              <DialogContent
                showCloseButton={false}
                className="max-h-[80vh] max-w-2xl overflow-hidden p-0"
              >
                <div className="bg-background sticky top-0 z-10 border-b px-6 py-4 pr-12">
                  <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                      <Sparkles className="text-primary h-5 w-5" />
                      {t('aiops.chat.helpTitle')}
                    </DialogTitle>
                    <DialogDescription>{t('aiops.chat.helpDesc')}</DialogDescription>
                  </DialogHeader>
                  <DialogClose asChild>
                    <Button
                      variant="ghost"
                      size="icon"
                      className="absolute top-4 right-4 h-8 w-8"
                      aria-label={t('common.close')}
                    >
                      <X className="h-4 w-4" />
                    </Button>
                  </DialogClose>
                </div>
                <div className="overflow-y-auto px-6 py-4">
                  <HelpContent />
                </div>
              </DialogContent>
            </Dialog>
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8"
              onClick={onClose}
              aria-label={t('common.close')}
            >
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Messages */}
        <ScrollArea className="chat-drawer-scroll-area min-h-0 min-w-0 flex-1 p-4">
          <div className="min-w-0 space-y-4">
            {messages.map((message) => (
              <div
                key={message.id}
                className={cn(
                  'flex min-w-0',
                  message.role === 'user' ? 'justify-end' : 'justify-start'
                )}
              >
                {message.role === 'user' ? (
                  <div className="bg-primary text-primary-foreground max-w-[85%] min-w-0 overflow-hidden rounded-lg px-4 py-2">
                    <p className="text-sm [overflow-wrap:anywhere] break-words whitespace-pre-wrap">
                      {message.content}
                    </p>
                  </div>
                ) : (
                  <div className="max-w-[95%] min-w-0 space-y-2">
                    <div className="bg-muted min-w-0 overflow-hidden rounded-lg px-4 py-3">
                      {(() => {
                        const { showAdminHint, cleanedContent } = getAssistantContentForDisplay(
                          message.content,
                          message.data,
                          adminHintText
                        )
                        return (
                          <>
                            {showAdminHint && (
                              <div className="bg-background/70 text-muted-foreground mb-2 rounded-md border px-3 py-1.5 text-xs">
                                {adminHintText}
                              </div>
                            )}
                            <div className="markdown-content w-full min-w-0 text-sm">
                              <ReactMarkdown
                                remarkPlugins={[remarkGfm]}
                                components={markdownComponents}
                              >
                                {cleanedContent}
                              </ReactMarkdown>
                            </div>
                          </>
                        )
                      })()}
                    </div>
                    {message.type === 'diagnosis' && isDiagnosisData(message.data) && (
                      <div className="min-w-0">
                        <DiagnosisCard diagnosis={message.data} />
                      </div>
                    )}
                  </div>
                )}
              </div>
            ))}
            {chatMutation.isPending && (
              <div className="flex min-w-0 justify-start">
                <div className="bg-muted flex min-w-0 items-center gap-2 overflow-hidden rounded-lg px-4 py-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-muted-foreground text-sm">
                    {t('aiops.chat.thinking')}
                  </span>
                </div>
              </div>
            )}
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Smart Prompts */}
        <div className="bg-muted/30 max-h-[30vh] flex-none overflow-y-auto border-t">
          <div className="space-y-2 p-3">
            {smartPrompts.map((category) => (
              <Collapsible
                key={category.id}
                open={expandedCategories.includes(category.id)}
                onOpenChange={() => toggleCategory(category.id)}
              >
                <CollapsibleTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    className="h-7 w-full justify-between text-xs font-medium"
                  >
                    {category.category}
                    <ChevronDown
                      className={cn(
                        'h-3 w-3 transition-transform',
                        expandedCategories.includes(category.id) && 'rotate-180'
                      )}
                    />
                  </Button>
                </CollapsibleTrigger>
                <CollapsibleContent className="pt-2">
                  <div className="grid gap-1.5">
                    {category.prompts.map((prompt, idx) => (
                      <Button
                        key={idx}
                        variant="outline"
                        size="sm"
                        onClick={() => handleSend(prompt.text)}
                        disabled={chatMutation.isPending}
                        className="h-auto justify-start px-2 py-1.5 text-left text-xs whitespace-normal"
                        title={prompt.hint}
                      >
                        <span className="mr-1.5">{prompt.icon}</span>
                        <span className="flex-1">{prompt.text}</span>
                        {prompt.hint && (
                          <span className="text-muted-foreground ml-1 hidden text-[10px] sm:inline">
                            {prompt.hint.split(/[，,]/)[0]}
                          </span>
                        )}
                      </Button>
                    ))}
                  </div>
                </CollapsibleContent>
              </Collapsible>
            ))}
          </div>
        </div>

        {/* Input */}
        <div className="bg-background flex-none border-t p-4">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  handleSend()
                }
              }}
              placeholder={t('aiops.chat.inputPlaceholder')}
              disabled={chatMutation.isPending}
              className="flex-1"
            />
            <Button
              onClick={() => handleSend()}
              disabled={chatMutation.isPending || !input.trim()}
              size="icon"
              aria-label={t('common.send')}
            >
              <Send className="h-4 w-4" />
            </Button>
          </div>
          {currentJobName && (
            <div className="text-muted-foreground mt-2 flex items-center gap-1 text-xs">
              <span>{t('aiops.chat.currentJob')}</span>
              <Badge variant="secondary" className="font-mono text-xs">
                {currentJobName}
              </Badge>
            </div>
          )}
        </div>
      </div>
    </>
  )
}

// ── Help Content Component ──────────────────────────────────────────────────

function HelpContent() {
  const { t } = useTranslation()

  return (
    <div className="space-y-6 text-sm">
      <section>
        <h4 className="mb-2 text-base font-semibold">{t('aiops.chat.help.usageTitle')}</h4>
        <div className="text-muted-foreground space-y-3">
          <div>
            <p className="text-foreground mb-1 font-medium">{t('aiops.chat.help.usage1Title')}</p>
            <p>{t('aiops.chat.help.usage1Desc')}</p>
          </div>
          <div>
            <p className="text-foreground mb-1 font-medium">{t('aiops.chat.help.usage2Title')}</p>
            <p>{t('aiops.chat.help.usage2Desc')}</p>
          </div>
          <div>
            <p className="text-foreground mb-1 font-medium">{t('aiops.chat.help.usage3Title')}</p>
            <p>
              {t('aiops.chat.help.usage3Desc')}:{' '}
              <code className="bg-muted rounded px-1 py-0.5">job:jpt-xxx-xxx</code> /{' '}
              <code className="bg-muted rounded px-1 py-0.5">analyze job jpt-xxx-xxx</code>
            </p>
          </div>
        </div>
      </section>

      <section>
        <h4 className="mb-2 text-base font-semibold">{t('aiops.chat.help.statsTitle')}</h4>
        <div className="space-y-2">
          <StatItem
            rank="1"
            type={t('aiops.chat.help.stat1Type')}
            percentage="24.1%"
            description={t('aiops.chat.help.stat1Desc')}
            isUserIssue
          />
          <StatItem
            rank="2"
            type={t('aiops.chat.help.stat2Type')}
            percentage="23.6%"
            description={t('aiops.chat.help.stat2Desc')}
          />
          <StatItem
            rank="3"
            type={t('aiops.chat.help.stat3Type')}
            percentage="18.4%"
            description={t('aiops.chat.help.stat3Desc')}
          />
          <StatItem
            rank="4"
            type={t('aiops.chat.help.stat4Type')}
            percentage="9.4%"
            description={t('aiops.chat.help.stat4Desc')}
            isUserIssue
          />
          <StatItem
            rank="5"
            type={t('aiops.chat.help.stat5Type')}
            percentage="4.2%"
            description={t('aiops.chat.help.stat5Desc')}
            isUserIssue
          />
        </div>
      </section>

      <section>
        <h4 className="mb-2 text-base font-semibold">{t('aiops.chat.help.selfCheckTitle')}</h4>
        <div className="space-y-2 rounded-lg border border-yellow-500/20 bg-yellow-500/10 p-3">
          <p className="font-medium text-yellow-700 dark:text-yellow-300">
            {t('aiops.chat.help.selfCheckHighlight')}
          </p>
          <div className="text-muted-foreground space-y-1 text-xs">
            <p>{t('aiops.chat.help.selfCheck1')}</p>
            <p>{t('aiops.chat.help.selfCheck2')}</p>
            <p>{t('aiops.chat.help.selfCheck3')}</p>
            <p>{t('aiops.chat.help.selfCheck4')}</p>
          </div>
        </div>
      </section>

      <section>
        <h4 className="mb-2 text-base font-semibold">{t('aiops.chat.help.supportedInputTitle')}</h4>
        <div className="bg-muted space-y-1.5 rounded-lg p-3 font-mono text-xs">
          <p>• {t('aiops.chat.help.input1')}</p>
          <p>• {t('aiops.chat.help.input2')}</p>
          <p>• {t('aiops.chat.help.input3')}</p>
          <p>• {t('aiops.chat.help.input4')}</p>
          <p>• {t('aiops.chat.help.input5')}</p>
        </div>
      </section>

      <section>
        <h4 className="mb-2 text-base font-semibold">{t('aiops.chat.help.limitationsTitle')}</h4>
        <div className="text-muted-foreground space-y-1 text-xs">
          <p>{t('aiops.chat.help.limit1')}</p>
          <p>{t('aiops.chat.help.limit2')}</p>
          <p>{t('aiops.chat.help.limit3')}</p>
          <p>{t('aiops.chat.help.limit4')}</p>
        </div>
      </section>
    </div>
  )
}

// ── Stat Item Component ─────────────────────────────────────────────────────

function StatItem({
  rank,
  type,
  percentage,
  description,
  isUserIssue = false,
}: {
  rank: string
  type: string
  percentage: string
  description: string
  isUserIssue?: boolean
}) {
  const { t } = useTranslation()

  return (
    <div className="bg-muted/50 flex items-start gap-2 rounded p-2">
      <Badge variant="outline" className="flex-shrink-0 font-mono text-xs">
        #{rank}
      </Badge>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-sm font-medium">{type}</span>
          <Badge variant="secondary" className="text-xs">
            {percentage}
          </Badge>
          {isUserIssue && (
            <Badge variant="destructive" className="h-4 text-[10px]">
              {t('aiops.chat.selfCheckShort')}
            </Badge>
          )}
        </div>
        <p className="text-muted-foreground mt-0.5 text-xs">{description}</p>
      </div>
    </div>
  )
}

// ── Diagnosis Card Component ────────────────────────────────────────────────

function DiagnosisCard({ diagnosis }: { diagnosis: IDiagnosis }) {
  const { t } = useTranslation()

  const severityColors = {
    critical: 'destructive',
    error: 'destructive',
    warning: 'secondary',
    info: 'secondary',
  } as const

  const severityIcons = {
    critical: AlertCircle,
    error: AlertCircle,
    warning: AlertCircle,
    info: CheckCircle,
  }

  const Icon = severityIcons[diagnosis.severity as keyof typeof severityIcons] || AlertCircle
  const severity = severityColors[diagnosis.severity as keyof typeof severityColors] ?? 'default'

  const isUserCodeIssue = ['ContainerError', 'CommandNotFound'].includes(diagnosis.category)

  return (
    <Card className="space-y-3 p-4">
      <div className="flex items-start gap-2">
        <Icon className="mt-0.5 h-5 w-5 flex-shrink-0" />
        <div className="flex-1 space-y-1">
          <div className="flex flex-wrap items-center gap-2">
            <h4 className="text-sm font-semibold">
              {t(`aiops.reason.${diagnosis.category}`, { defaultValue: diagnosis.category })}
            </h4>
            <Badge variant={severity}>{diagnosis.severity}</Badge>
            <Badge variant="outline" className="text-xs">
              {diagnosis.confidence}
            </Badge>
            {isUserCodeIssue && (
              <Badge variant="destructive" className="text-xs">
                {t('aiops.chat.selfCheckCode')}
              </Badge>
            )}
          </div>
          <p className="text-muted-foreground text-sm">{diagnosis.diagnosis}</p>
        </div>
      </div>
      {diagnosis.solution && (
        <div className="bg-muted/50 rounded p-3">
          <p className="mb-1 text-xs font-medium">{t('aiops.chat.solution')}</p>
          <p className="text-xs whitespace-pre-wrap">{diagnosis.solution}</p>
        </div>
      )}
      {isUserCodeIssue && (
        <div className="rounded border border-yellow-500/20 bg-yellow-500/10 p-2">
          <p className="flex items-center gap-1 text-xs text-yellow-700 dark:text-yellow-300">
            <AlertCircle className="h-3 w-3" />
            <strong>{t('aiops.chat.tipLabel')}</strong>
            {t('aiops.chat.tipText')}
          </p>
        </div>
      )}
      {diagnosis.evidence && diagnosis.evidence.exitCode && (
        <div className="flex items-center gap-2 text-xs">
          <span className="text-muted-foreground">{t('aiops.chat.exitCode')}</span>
          <Badge variant="outline" className="font-mono">
            {diagnosis.evidence.exitCode}
          </Badge>
          {diagnosis.evidence.exitReason && (
            <span className="text-muted-foreground">({diagnosis.evidence.exitReason})</span>
          )}
        </div>
      )}
      {diagnosis.evidence && diagnosis.evidence.events && diagnosis.evidence.events.length > 0 && (
        <div className="space-y-1">
          <p className="text-xs font-medium">{t('aiops.chat.relatedEvents')}</p>
          <div className="space-y-1">
            {diagnosis.evidence.events.slice(0, 3).map((event: string, i: number) => (
              <div key={i} className="text-muted-foreground bg-muted/30 rounded p-2 text-xs">
                {event}
              </div>
            ))}
          </div>
        </div>
      )}
    </Card>
  )
}
