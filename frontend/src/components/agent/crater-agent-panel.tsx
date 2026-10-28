import { useChat } from '@ai-sdk/react'
import { useLocation, useNavigate } from '@tanstack/react-router'
import { DefaultChatTransport, type UIMessage } from 'ai'
import { useAtom, useAtomValue, useSetAtom } from 'jotai'
import {
  BotIcon,
  CopyIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PencilIcon,
  SendIcon,
  SquareIcon,
  XIcon,
} from 'lucide-react'
import {
  type FormEvent,
  type KeyboardEvent,
  type PointerEvent,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'
import type { Components } from 'react-markdown'
import { toast } from 'sonner'

import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
  SheetTrigger,
} from '@/components/ui/sheet'
import { Textarea } from '@/components/ui/textarea'

import type {
  AgentActionContext,
  AgentFormPatch,
  AgentFormRuntime,
  AgentPageContext,
  AgentResourceContext,
} from '@/components/agent/agent-context'
import {
  AgentToolCard,
  AgentToolGroup,
  type AgentToolPart,
} from '@/components/agent/agent-tool-card'
import { ConversationSidebar } from '@/components/agent/conversation-sidebar'
import { isAgentNavigationDestination } from '@/components/agent/form-capabilities'
import { MarkdownRenderer } from '@/components/form/markdown-renderer'
import { UserAvatar } from '@/components/user/user-avatar'

import {
  type AgentConversation,
  type AgentIdentity,
  agentIdentityHeader,
  apiCancelAgentConversationRun,
  apiDeleteAgentConversation,
  apiGetAgentConversationMessages,
  apiGetAgentConversations,
  apiRenameAgentConversation,
  apiSaveAgentToolOutput,
} from '@/services/api/agent'

import {
  ACCESS_TOKEN_KEY,
  atomAgentActionContext,
  atomAgentEntryRequest,
  atomAgentFormRuntime,
  atomAgentPageResourceContext,
  atomAgentPanelOpen,
  atomAgentUserFloatingButtonEnabled,
  atomUserInfo,
} from '@/utils/store'

import { cn } from '@/lib/utils'

const chatEndpoint = '/api/v1/agent/chat'
const streamThrottleMilliseconds = 40
const panelWidthStorageKey = 'crater_agent_panel_width'
const defaultPanelWidth = 760
const minimumPanelWidth = 560
const maximumPanelWidth = 1100
const bottomStickThreshold = 96
const activeConversationCache = new Map<string, string>()
const agentMarkdownComponents: Components = {
  table: ({ children, className, ...props }) => (
    <div className="border-foreground/20 w-full max-w-full min-w-0 overflow-x-auto rounded-md border">
      <table {...props} className={cn('w-full min-w-[42rem] border-collapse', className)}>
        {children}
      </table>
    </div>
  ),
}

type PendingConversationTitle = {
  conversationID: string
  firstMessage: string
}

type AgentSendMode = 'append' | 'replace'

type AgentRequestContext = {
  conversationID: string
  userMessageID: string
  userText: string
  finalizing: boolean
  cancelPromise?: Promise<void>
}

type SynchronizationExpectation = {
  userMessageID: string
  assistantResponseExpected: boolean
  initialDelay?: number
}

type SynchronizeConversation = (
  conversationID: string,
  expectation?: SynchronizationExpectation
) => Promise<void>

const synchronizationRetryDelays = [0, 100, 250, 500, 1_000, 1_500]
const cancellationRetryDelays = [0, 100, 250, 500]

function createConversationID() {
  return crypto.randomUUID()
}

function clampPanelWidth(width: number) {
  if (typeof window === 'undefined') return defaultPanelWidth
  const viewportMaximum = Math.max(320, Math.min(maximumPanelWidth, window.innerWidth - 16))
  const viewportMinimum = Math.min(minimumPanelWidth, viewportMaximum)
  return Math.min(viewportMaximum, Math.max(viewportMinimum, width))
}

function initialPanelWidth() {
  if (typeof window === 'undefined') return defaultPanelWidth
  const storedWidth = Number(localStorage.getItem(panelWidthStorageKey))
  return clampPanelWidth(
    Number.isFinite(storedWidth) && storedWidth > 0 ? storedWidth : defaultPanelWidth
  )
}

function messageText(message: UIMessage) {
  return message.parts
    .filter((part) => part.type === 'text')
    .map((part) => part.text)
    .join('')
}

function AgentAvatar() {
  return (
    <Avatar className="mt-1 size-8 border">
      <AvatarFallback className="bg-primary/10 text-primary">
        <BotIcon className="size-4" />
      </AvatarFallback>
    </Avatar>
  )
}

type ResizeHandleProps = {
  panelWidth: number
  setPanelWidth: (width: number) => void
}

function ResizeHandle({ panelWidth, setPanelWidth }: ResizeHandleProps) {
  const { t } = useTranslation()
  const startRef = useRef<{ pointerX: number; panelWidth: number } | null>(null)
  const currentWidthRef = useRef(panelWidth)
  currentWidthRef.current = panelWidth

  useEffect(
    () => () => {
      document.body.style.cursor = ''
      document.body.style.userSelect = ''
    },
    []
  )

  const startResize = (event: PointerEvent<HTMLDivElement>) => {
    startRef.current = { pointerX: event.clientX, panelWidth }
    event.currentTarget.setPointerCapture(event.pointerId)
    document.body.style.cursor = 'ew-resize'
    document.body.style.userSelect = 'none'
  }

  const resize = (event: PointerEvent<HTMLDivElement>) => {
    const start = startRef.current
    if (!start) return
    const nextWidth = clampPanelWidth(start.panelWidth + start.pointerX - event.clientX)
    currentWidthRef.current = nextWidth
    setPanelWidth(nextWidth)
  }

  const finishResize = (event: PointerEvent<HTMLDivElement>) => {
    if (!startRef.current) return
    startRef.current = null
    if (event.currentTarget.hasPointerCapture(event.pointerId)) {
      event.currentTarget.releasePointerCapture(event.pointerId)
    }
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    localStorage.setItem(panelWidthStorageKey, String(currentWidthRef.current))
  }

  const resizeWithKeyboard = (event: KeyboardEvent<HTMLDivElement>) => {
    let nextWidth = panelWidth
    if (event.key === 'ArrowLeft') nextWidth += 24
    else if (event.key === 'ArrowRight') nextWidth -= 24
    else if (event.key === 'Home') nextWidth = minimumPanelWidth
    else if (event.key === 'End') nextWidth = maximumPanelWidth
    else return

    event.preventDefault()
    nextWidth = clampPanelWidth(nextWidth)
    setPanelWidth(nextWidth)
    localStorage.setItem(panelWidthStorageKey, String(nextWidth))
  }

  return (
    <div
      role="separator"
      tabIndex={0}
      aria-label={t('craterAgent.resize')}
      aria-orientation="vertical"
      aria-valuemin={minimumPanelWidth}
      aria-valuemax={maximumPanelWidth}
      aria-valuenow={Math.round(panelWidth)}
      className="group absolute inset-y-0 -left-1.5 z-50 w-3 cursor-ew-resize touch-none outline-none"
      onKeyDown={resizeWithKeyboard}
      onPointerCancel={finishResize}
      onPointerDown={startResize}
      onPointerMove={resize}
      onPointerUp={finishResize}
    >
      <span className="bg-border group-hover:bg-primary group-focus-visible:bg-primary absolute inset-y-0 left-1/2 w-px transition-colors" />
    </div>
  )
}

type ChatMessageProps = {
  identity: AgentIdentity
  message: UIMessage
  user?: { name: string; avatar?: string }
  busy: boolean
  editing: boolean
  editingText: string
  onEditingTextChange: (text: string) => void
  onCopy: (text: string) => void
  onEdit: (messageID: string, text: string) => void
  onCancelEdit: () => void
  onResend: (event: FormEvent) => void
  runtime?: AgentFormRuntime
  onNavigate: (messageID: string, part: AgentToolPart, path: string) => Promise<void>
  onNavigateJob: (messageID: string, part: AgentToolPart, jobName: string) => Promise<void>
  onApply: (
    messageID: string,
    part: AgentToolPart,
    patches: readonly AgentFormPatch[]
  ) => Promise<string[]>
  onReject: (messageID: string, part: AgentToolPart) => Promise<void>
}

type AgentToolPartGroup = {
  type: string
  parts: AgentToolPart[]
}

const confirmationToolTypes = new Set([
  'tool-navigate_to_page',
  'tool-navigate_to_job_detail',
  'tool-propose_form_patch',
])

function isConfirmationToolPart(part: AgentToolPart) {
  return confirmationToolTypes.has(part.type)
}

function isPendingConfirmationToolPart(part: AgentToolPart) {
  return (
    isConfirmationToolPart(part) &&
    part.state !== 'output-available' &&
    part.state !== 'output-error'
  )
}

function groupAgentToolParts(parts: readonly AgentToolPart[]): AgentToolPartGroup[] {
  const groups: AgentToolPartGroup[] = []
  for (const part of parts) {
    const previous = groups.at(-1)
    if (previous && previous.type === part.type && !isConfirmationToolPart(part)) {
      previous.parts.push(part)
      continue
    }
    groups.push({ type: part.type, parts: [part] })
  }
  return groups
}

function confirmationLeadText(parts: readonly AgentToolPart[]) {
  for (const part of parts) {
    if (typeof part.input !== 'object' || part.input === null || Array.isArray(part.input)) continue
    const input = part.input as Record<string, unknown>
    const value =
      part.type === 'tool-navigate_to_page' || part.type === 'tool-navigate_to_job_detail'
        ? input.reason
        : input.summary
    if (typeof value === 'string' && value.trim()) return value.trim()
  }
  return ''
}

function ChatMessage({
  identity,
  message,
  user,
  busy,
  editing,
  editingText,
  onEditingTextChange,
  onCopy,
  onEdit,
  onCancelEdit,
  onResend,
  runtime,
  onNavigate,
  onNavigateJob,
  onApply,
  onReject,
}: ChatMessageProps) {
  const { t } = useTranslation()
  const text = messageText(message)
  const toolParts = message.parts.filter((part): part is typeof part & AgentToolPart =>
    part.type.startsWith('tool-')
  )
  const automaticToolPartGroups = groupAgentToolParts(
    toolParts.filter((part) => !isConfirmationToolPart(part))
  )
  const confirmationToolParts = toolParts.filter(isConfirmationToolPart)
  const fromUser = message.role === 'user'
  const displayText = fromUser ? text : text || confirmationLeadText(confirmationToolParts)
  if (!displayText && toolParts.length === 0) return null

  if (editing) {
    return (
      <form className="flex flex-row-reverse items-start gap-2.5" onSubmit={onResend}>
        <UserAvatar user={user} className="mt-1" />
        <div className="border-primary/40 bg-background min-w-0 flex-1 space-y-2 rounded-2xl border p-3 shadow-sm">
          <Textarea
            autoFocus
            rows={2}
            value={editingText}
            className="max-h-40 min-h-20 resize-none overflow-y-auto"
            aria-label={t('craterAgent.editMessage')}
            onChange={(event) => onEditingTextChange(event.target.value)}
            onKeyDown={(event) => {
              if (event.key === 'Escape') onCancelEdit()
              if (event.key === 'Enter' && !event.shiftKey) {
                event.preventDefault()
                event.currentTarget.form?.requestSubmit()
              }
            }}
          />
          <div className="flex justify-end gap-2">
            <Button type="button" size="sm" variant="ghost" onClick={onCancelEdit}>
              <XIcon />
              {t('craterAgent.cancelEdit')}
            </Button>
            <Button type="submit" size="sm" disabled={!editingText.trim()}>
              <SendIcon />
              {t('craterAgent.resend')}
            </Button>
          </div>
        </div>
      </form>
    )
  }

  return (
    <div
      className={cn(
        'group flex max-w-full min-w-0 items-start gap-2.5',
        fromUser ? 'flex-row-reverse' : 'flex-row'
      )}
    >
      {fromUser ? <UserAvatar user={user} className="mt-1" /> : <AgentAvatar />}
      <div className={cn('relative min-w-0', fromUser ? 'max-w-[calc(100%_-_7.25rem)]' : 'flex-1')}>
        <div
          className={cn(
            'max-w-full min-w-0 overflow-hidden rounded-2xl px-4 py-3 text-sm leading-6',
            fromUser ? 'bg-primary text-primary-foreground' : 'bg-muted'
          )}
        >
          {!fromUser && automaticToolPartGroups.length > 0 && (
            <div className="space-y-1.5">
              {automaticToolPartGroups.map((group) => {
                const firstPart = group.parts[0]
                if (!firstPart) return null
                if (group.parts.length === 1) {
                  return (
                    <AgentToolCard
                      key={firstPart.toolCallId}
                      identity={identity}
                      part={firstPart}
                      runtime={runtime}
                      busy={busy}
                      onNavigate={(path) => onNavigate(message.id, firstPart, path)}
                      onNavigateJob={(jobName) => onNavigateJob(message.id, firstPart, jobName)}
                      onApply={(patches) => onApply(message.id, firstPart, patches)}
                      onReject={() => onReject(message.id, firstPart)}
                    />
                  )
                }
                return (
                  <AgentToolGroup key={firstPart.toolCallId} parts={group.parts}>
                    {group.parts.map((part, index) => (
                      <AgentToolCard
                        key={part.toolCallId}
                        identity={identity}
                        part={part}
                        sequence={index + 1}
                        runtime={runtime}
                        busy={busy}
                        onNavigate={(path) => onNavigate(message.id, part, path)}
                        onNavigateJob={(jobName) => onNavigateJob(message.id, part, jobName)}
                        onApply={(patches) => onApply(message.id, part, patches)}
                        onReject={() => onReject(message.id, part)}
                      />
                    ))}
                  </AgentToolGroup>
                )
              })}
            </div>
          )}
          {fromUser ? (
            <p className="wrap-anywhere whitespace-pre-wrap">{displayText}</p>
          ) : displayText ? (
            <div
              className={cn(
                'max-w-full min-w-0 wrap-anywhere [&_a]:break-all [&_pre]:max-w-full',
                automaticToolPartGroups.length > 0 && 'mt-3'
              )}
            >
              <MarkdownRenderer components={agentMarkdownComponents}>
                {displayText}
              </MarkdownRenderer>
            </div>
          ) : null}
          {!fromUser && confirmationToolParts.length > 0 && (
            <div className={cn('space-y-2', displayText && 'mt-3')}>
              {confirmationToolParts.map((part) => (
                <AgentToolCard
                  key={part.toolCallId}
                  identity={identity}
                  part={part}
                  runtime={runtime}
                  busy={busy}
                  onNavigate={(path) => onNavigate(message.id, part, path)}
                  onNavigateJob={(jobName) => onNavigateJob(message.id, part, jobName)}
                  onApply={(patches) => onApply(message.id, part, patches)}
                  onReject={() => onReject(message.id, part)}
                />
              ))}
            </div>
          )}
        </div>
        <div
          className={cn(
            'bg-background text-foreground absolute top-1/2 z-10 flex -translate-y-1/2 rounded-lg border opacity-0 shadow-sm transition-opacity group-focus-within:opacity-100 group-hover:opacity-100',
            fromUser ? 'right-full mr-1' : 'left-full ml-1'
          )}
        >
          {fromUser && (
            <Button
              type="button"
              variant="ghost"
              size="icon"
              className="size-8"
              disabled={busy}
              aria-label={t('craterAgent.editMessage')}
              onClick={() => onEdit(message.id, text)}
            >
              <PencilIcon className="size-4" />
            </Button>
          )}
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="size-8"
            aria-label={t('craterAgent.copy')}
            onClick={() => onCopy(displayText)}
          >
            <CopyIcon className="size-4" />
          </Button>
        </div>
      </div>
    </div>
  )
}

type CraterAgentPanelProps = {
  identity: AgentIdentity
}

export function CraterAgentPanel({ identity }: CraterAgentPanelProps) {
  const { t } = useTranslation()
  const user = useAtomValue(atomUserInfo)
  const userID = user?.id
  const [input, setInput] = useState('')
  const [open, setOpen] = useAtom(atomAgentPanelOpen)
  const actionContext = useAtomValue(atomAgentActionContext)
  const entryRequest = useAtomValue(atomAgentEntryRequest)
  const clearEntryRequest = useSetAtom(atomAgentEntryRequest)
  const formRuntime = useAtomValue(atomAgentFormRuntime)
  const resourceContext = useAtomValue(atomAgentPageResourceContext)
  const navigate = useNavigate()
  const location = useLocation()
  const floatingButtonEnabled = useAtomValue(atomAgentUserFloatingButtonEnabled)
  const identityCacheKey = `${userID ?? 0}:${identity}`
  const [cancelled, setCancelled] = useState(false)
  const [editingMessageID, setEditingMessageID] = useState<string>()
  const [editingText, setEditingText] = useState('')
  const [panelWidth, setPanelWidth] = useState(initialPanelWidth)
  const [historyOpen, setHistoryOpen] = useState(true)
  const [conversations, setConversations] = useState<AgentConversation[]>([])
  const [loadedIdentityCacheKey, setLoadedIdentityCacheKey] = useState<string>()
  const [activeConversationID, setActiveConversationID] = useState<string>(createConversationID)
  const [requestLocked, setRequestLocked] = useState(false)
  const [synchronizing, setSynchronizing] = useState(false)
  const viewportRef = useRef<HTMLDivElement | null>(null)
  const stickToBottomRef = useRef(true)
  const pendingTitleRef = useRef<PendingConversationTitle | undefined>(undefined)
  const sendModeRef = useRef<AgentSendMode>('append')
  const activeRequestRef = useRef<AgentRequestContext | undefined>(undefined)
  const interactionLockedRef = useRef(false)
  const navigationSequenceRef = useRef(0)
  const synchronizationSequenceRef = useRef(0)
  const processedEntryRequestIDRef = useRef<string | undefined>(undefined)
  const synchronizeConversationRef = useRef<SynchronizeConversation>(async () => {})
  const activeConversationIDRef = useRef(activeConversationID)
  activeConversationIDRef.current = activeConversationID
  const pageContextRef = useRef<AgentPageContext | undefined>(undefined)
  pageContextRef.current = buildPageContext(
    actionContext,
    formRuntime,
    resourceContext,
    location.pathname
  )
  const transport = useMemo(
    () =>
      new DefaultChatTransport({
        api: chatEndpoint,
        headers: () => ({
          Authorization: `Bearer ${localStorage.getItem(ACCESS_TOKEN_KEY) ?? ''}`,
          [agentIdentityHeader]: identity,
        }),
        fetch: agentFetch,
        prepareSendMessagesRequest: ({ messages }) => {
          const message = messages[messages.length - 1]
          const parentMessageID = messages.length > 1 ? messages[messages.length - 2].id : null
          const mode = sendModeRef.current
          const conversationID = activeConversationIDRef.current
          activeRequestRef.current = {
            conversationID,
            userMessageID: message.id,
            userText: messageText(message),
            finalizing: false,
          }
          sendModeRef.current = 'append'
          return {
            body: {
              conversationId: conversationID,
              parentMessageId: parentMessageID,
              mode,
              message,
              pageContext: pageContextRef.current,
            },
          }
        },
      }),
    [identity]
  )
  const generatePendingConversationTitle = (conversationID: string) => {
    const pendingTitle = pendingTitleRef.current
    if (!pendingTitle || pendingTitle.conversationID !== conversationID) return
    pendingTitleRef.current = undefined

    void requestConversationTitle(identity, pendingTitle.conversationID, pendingTitle.firstMessage)
      .then((title) => {
        if (!title) return
        setConversations((current) =>
          current.map((conversation) =>
            conversation.id === pendingTitle.conversationID && !conversation.customTitle
              ? { ...conversation, title, updatedAt: Date.now() }
              : conversation
          )
        )
      })
      .catch(() => {
        // Title generation is optional and must never interrupt the conversation.
      })
  }
  const { messages, setMessages, sendMessage, addToolOutput, status, stop, error, clearError } =
    useChat({
      transport,
      throttle: streamThrottleMilliseconds,
      onFinish: ({ message, isAbort, isError }) => {
        const request = activeRequestRef.current
        if (!request || request.finalizing) return
        request.finalizing = true
        if (isAbort || isError) {
          if (pendingTitleRef.current?.conversationID === request.conversationID) {
            pendingTitleRef.current = undefined
          }
        }
        const assistantResponseExpected = messageText(message).trim().length > 0
        const cancellation = isAbort
          ? (request.cancelPromise ?? cancelActiveRun(request.conversationID))
          : Promise.resolve()
        void cancellation
          .catch((cancelError: Error) => toast.error(cancelError.message))
          .then(() =>
            synchronizeConversationRef.current(request.conversationID, {
              userMessageID: request.userMessageID,
              assistantResponseExpected: !isAbort && !isError && assistantResponseExpected,
              initialDelay: isAbort || isError ? 100 : undefined,
            })
          )
          .then(() => {
            if (!isAbort && !isError) generatePendingConversationTitle(request.conversationID)
          })
          .catch((syncError: Error) => toast.error(syncError.message))
          .finally(() => {
            if (activeRequestRef.current === request) activeRequestRef.current = undefined
            interactionLockedRef.current = false
            setRequestLocked(false)
          })
      },
      onError: (chatError) => {
        const request = activeRequestRef.current
        if (!request || request.finalizing) return
        request.finalizing = true
        if (pendingTitleRef.current?.conversationID === request.conversationID) {
          pendingTitleRef.current = undefined
        }
        toast.error(
          chatError instanceof AgentRequestError && chatError.status === 409
            ? t('craterAgent.conversationRefreshed')
            : chatError.message
        )
        const conflict = chatError instanceof AgentRequestError && chatError.status === 409
        void (request.cancelPromise ?? Promise.resolve())
          .catch((cancelError: Error) => toast.error(cancelError.message))
          .then(() => synchronizeConversationRef.current(request.conversationID))
          .then(() => {
            clearError()
            if (conflict) {
              setInput((current) => (current.trim() ? current : request.userText))
            }
          })
          .catch((syncError: Error) => toast.error(syncError.message))
          .finally(() => {
            if (activeRequestRef.current === request) activeRequestRef.current = undefined
            interactionLockedRef.current = false
            setRequestLocked(false)
          })
      },
    })
  const cancelActiveRun = async (conversationID: string) => {
    let lastError: Error | undefined
    for (const delay of cancellationRetryDelays) {
      if (delay > 0) await new Promise((resolve) => window.setTimeout(resolve, delay))
      try {
        const response = await apiCancelAgentConversationRun(identity, conversationID)
        if (response.data) return
        lastError = undefined
      } catch (cancelError) {
        lastError = cancelError instanceof Error ? cancelError : new Error(String(cancelError))
      }
    }
    if (lastError) throw lastError
  }
  const synchronizeConversation: SynchronizeConversation = async (conversationID, expectation) => {
    const synchronizationSequence = synchronizationSequenceRef.current + 1
    synchronizationSequenceRef.current = synchronizationSequence
    setSynchronizing(true)
    let latestMessages: UIMessage[] = []
    try {
      if (expectation?.initialDelay) {
        await new Promise((resolve) => window.setTimeout(resolve, expectation.initialDelay))
      }
      for (const delay of synchronizationRetryDelays) {
        if (delay > 0) await new Promise((resolve) => window.setTimeout(resolve, delay))
        latestMessages = (await apiGetAgentConversationMessages(identity, conversationID)).data
        const userMessageIndex = expectation
          ? latestMessages.findIndex((candidate) => candidate.id === expectation.userMessageID)
          : -1
        const responseSaved =
          !expectation ||
          (userMessageIndex >= 0 &&
            (!expectation.assistantResponseExpected ||
              latestMessages
                .slice(userMessageIndex + 1)
                .some((candidate) => candidate.role === 'assistant')))
        if (responseSaved) {
          if (activeConversationIDRef.current === conversationID) setMessages(latestMessages)
          return
        }
      }
      if (activeConversationIDRef.current === conversationID) setMessages(latestMessages)
      throw new Error(t('craterAgent.syncFailed'))
    } finally {
      if (synchronizationSequenceRef.current === synchronizationSequence) {
        setSynchronizing(false)
      }
    }
  }
  synchronizeConversationRef.current = synchronizeConversation
  const generating = status === 'submitted' || status === 'streaming'
  const busy = generating || synchronizing || requestLocked
  const awaitingConfirmation = messages.some((message) =>
    message.parts.some(
      (part) =>
        part.type.startsWith('tool-') && isPendingConfirmationToolPart(part as AgentToolPart)
    )
  )

  useEffect(() => {
    const resizePanel = () => setPanelWidth((width) => clampPanelWidth(width))
    window.addEventListener('resize', resizePanel)
    return () => window.removeEventListener('resize', resizePanel)
  }, [])

  useEffect(() => {
    if (userID === undefined) return
    setLoadedIdentityCacheKey(undefined)
    let disposed = false
    const navigationSequence = navigationSequenceRef.current + 1
    navigationSequenceRef.current = navigationSequence
    void apiGetAgentConversations(identity)
      .then(async (response) => {
        if (disposed || navigationSequenceRef.current !== navigationSequence) return
        const loadedConversations = response.data
        const cachedConversationID =
          activeConversationCache.get(identityCacheKey) ?? createConversationID()
        const cachedConversation = loadedConversations.find(
          (conversation) => conversation.id === cachedConversationID
        )
        const loadedMessages = cachedConversation
          ? (await apiGetAgentConversationMessages(identity, cachedConversation.id)).data
          : []
        if (disposed || navigationSequenceRef.current !== navigationSequence) return
        activeConversationCache.set(identityCacheKey, cachedConversationID)
        setConversations(loadedConversations)
        setActiveConversationID(cachedConversationID)
        setMessages(loadedMessages)
        setInput('')
        setCancelled(false)
        setEditingMessageID(undefined)
        setEditingText('')
        setLoadedIdentityCacheKey(identityCacheKey)
      })
      .catch((loadError: Error) => {
        if (disposed || navigationSequenceRef.current !== navigationSequence) return
        setLoadedIdentityCacheKey(identityCacheKey)
        toast.error(loadError.message)
      })
    return () => {
      disposed = true
    }
  }, [identity, identityCacheKey, setMessages, userID])

  useEffect(() => {
    if (!open) return
    const viewport = viewportRef.current
    if (!viewport) return

    stickToBottomRef.current = true
    const trackPosition = () => {
      const distanceFromBottom = viewport.scrollHeight - viewport.scrollTop - viewport.clientHeight
      stickToBottomRef.current = distanceFromBottom < bottomStickThreshold
    }
    viewport.addEventListener('scroll', trackPosition, { passive: true })
    const frame = requestAnimationFrame(() => {
      viewport.scrollTop = viewport.scrollHeight
    })
    return () => {
      cancelAnimationFrame(frame)
      viewport.removeEventListener('scroll', trackPosition)
    }
  }, [open])

  useEffect(() => {
    if (!open || !busy) return
    stickToBottomRef.current = true
    let frame: number
    const followStreamingOutput = () => {
      const viewport = viewportRef.current
      if (!viewport) return
      viewport.scrollTop = viewport.scrollHeight
      frame = requestAnimationFrame(followStreamingOutput)
    }
    frame = requestAnimationFrame(followStreamingOutput)
    return () => cancelAnimationFrame(frame)
  }, [busy, open])

  useEffect(() => {
    const viewport = viewportRef.current
    if (!open || busy || !viewport || !stickToBottomRef.current) return
    const frame = requestAnimationFrame(() => {
      viewport.scrollTop = viewport.scrollHeight
    })
    return () => cancelAnimationFrame(frame)
  }, [busy, messages, open])

  const updateConversations = (updater: (current: AgentConversation[]) => AgentConversation[]) => {
    setConversations(updater)
  }

  const resetConversationView = useCallback(
    (conversationID: string, nextMessages: UIMessage[]) => {
      if (userID !== undefined) activeConversationCache.set(identityCacheKey, conversationID)
      activeConversationIDRef.current = conversationID
      synchronizationSequenceRef.current += 1
      sendModeRef.current = 'append'
      activeRequestRef.current = undefined
      interactionLockedRef.current = false
      setRequestLocked(false)
      pendingTitleRef.current = undefined
      setSynchronizing(false)
      setActiveConversationID(conversationID)
      setMessages(nextMessages)
      setInput('')
      setCancelled(false)
      setEditingMessageID(undefined)
      setEditingText('')
      clearError()
      stickToBottomRef.current = true
    },
    [clearError, identityCacheKey, setMessages, userID]
  )

  const startNewConversationWithMessage = useCallback(
    (initialMessage: string) => {
      const text = initialMessage.trim()
      if (!text || busy || interactionLockedRef.current) return false

      const conversationID = createConversationID()
      const now = Date.now()
      navigationSequenceRef.current += 1
      resetConversationView(conversationID, [])
      setConversations((current) => [
        {
          id: conversationID,
          title: t('craterAgent.newConversation'),
          createdAt: now,
          updatedAt: now,
          customTitle: false,
        },
        ...current,
      ])
      interactionLockedRef.current = true
      setRequestLocked(true)
      pendingTitleRef.current = { conversationID, firstMessage: text }
      stickToBottomRef.current = true
      sendModeRef.current = 'append'
      void sendMessage({ text })
      return true
    },
    [busy, resetConversationView, sendMessage, t]
  )

  useEffect(() => {
    if (
      !open ||
      !entryRequest ||
      loadedIdentityCacheKey !== identityCacheKey ||
      processedEntryRequestIDRef.current === entryRequest.id
    ) {
      return
    }
    if (!startNewConversationWithMessage(entryRequest.initialMessage)) return

    processedEntryRequestIDRef.current = entryRequest.id
    clearEntryRequest(undefined)
  }, [
    clearEntryRequest,
    entryRequest,
    identityCacheKey,
    loadedIdentityCacheKey,
    open,
    startNewConversationWithMessage,
  ])

  const createNewConversation = () => {
    if (busy) return
    navigationSequenceRef.current += 1
    resetConversationView(createConversationID(), [])
  }

  const selectConversation = (conversation: AgentConversation) => {
    if (busy || conversation.id === activeConversationID) return
    const navigationSequence = navigationSequenceRef.current + 1
    navigationSequenceRef.current = navigationSequence
    void apiGetAgentConversationMessages(identity, conversation.id)
      .then((response) => {
        if (navigationSequenceRef.current !== navigationSequence) return
        resetConversationView(conversation.id, response.data)
      })
      .catch((loadError: Error) => toast.error(loadError.message))
  }

  const ensureActiveConversation = () => {
    updateConversations((current) => {
      if (current.some((conversation) => conversation.id === activeConversationID)) return current
      const now = Date.now()
      return [
        {
          id: activeConversationID,
          title: t('craterAgent.newConversation'),
          createdAt: now,
          updatedAt: now,
          customTitle: false,
        },
        ...current,
      ]
    })
  }

  const submit = (event: FormEvent) => {
    event.preventDefault()
    const text = input.trim()
    if (!text || busy || awaitingConfirmation || interactionLockedRef.current) return
    interactionLockedRef.current = true
    setRequestLocked(true)
    const isFirstMessage = !messages.some((message) => message.role === 'user')
    ensureActiveConversation()
    if (isFirstMessage) {
      pendingTitleRef.current = { conversationID: activeConversationID, firstMessage: text }
    }
    stickToBottomRef.current = true
    clearError()
    setCancelled(false)
    setInput('')
    sendModeRef.current = 'append'
    void sendMessage({ text })
  }

  const resendEditedMessage = (event: FormEvent) => {
    event.preventDefault()
    const text = editingText.trim()
    const messageIndex = messages.findIndex((message) => message.id === editingMessageID)
    if (!text || busy || awaitingConfirmation || interactionLockedRef.current || messageIndex < 0)
      return
    interactionLockedRef.current = true
    setRequestLocked(true)

    const firstUserMessageIndex = messages.findIndex((message) => message.role === 'user')
    if (messageIndex === firstUserMessageIndex) {
      updateConversations((current) =>
        current.map((conversation) =>
          conversation.id === activeConversationID && !conversation.customTitle
            ? {
                ...conversation,
                title: t('craterAgent.newConversation'),
                updatedAt: Date.now(),
              }
            : conversation
        )
      )
      pendingTitleRef.current = { conversationID: activeConversationID, firstMessage: text }
    }
    stickToBottomRef.current = true
    clearError()
    setCancelled(false)
    sendModeRef.current = 'replace'
    setMessages(messages.slice(0, messageIndex))
    setEditingMessageID(undefined)
    setEditingText('')
    void sendMessage({ text })
  }

  const renameConversation = (conversationID: string, title: string) => {
    void apiRenameAgentConversation(identity, conversationID, title)
      .then(() =>
        updateConversations((current) =>
          current.map((conversation) =>
            conversation.id === conversationID
              ? { ...conversation, title, customTitle: true, updatedAt: Date.now() }
              : conversation
          )
        )
      )
      .catch((renameError: Error) => toast.error(renameError.message))
  }

  const deleteConversation = (conversationID: string) => {
    void apiDeleteAgentConversation(identity, conversationID)
      .then(() => {
        updateConversations((current) =>
          current.filter((conversation) => conversation.id !== conversationID)
        )
        if (conversationID === activeConversationID) {
          navigationSequenceRef.current += 1
          resetConversationView(createConversationID(), [])
        }
      })
      .catch((deleteError: Error) => toast.error(deleteError.message))
  }

  const copyMessage = (text: string) => {
    void navigator.clipboard
      .writeText(text)
      .then(() =>
        toast.success(t('craterAgent.copySuccess'), {
          icon: <CopyIcon className="size-4" />,
          position: 'bottom-right',
        })
      )
      .catch(() => toast.error(t('craterAgent.copyFailed'), { position: 'bottom-right' }))
  }

  const saveToolOutput = async (
    messageID: string,
    part: AgentToolPart,
    output: Record<string, unknown>
  ) => {
    await apiSaveAgentToolOutput(identity, activeConversationID, {
      messageId: messageID,
      toolCallId: part.toolCallId,
      state: 'output-available',
      output,
    })
    await addToolOutput({
      tool: part.type.slice('tool-'.length),
      toolCallId: part.toolCallId,
      output,
    })
  }

  const navigateFromTool = async (messageID: string, part: AgentToolPart, path: string) => {
    if (!isAgentNavigationDestination(identity, path)) {
      throw new Error(t('craterAgent.navigationNotAllowed'))
    }
    await navigate({ to: path })
    await saveToolOutput(messageID, part, { status: 'accepted', path })
    setOpen(false)
  }

  const navigateToJobDetailFromTool = async (
    messageID: string,
    part: AgentToolPart,
    jobName: string
  ) => {
    if (!jobName) throw new Error(t('craterAgent.navigationNotAllowed'))
    if (identity === 'admin') {
      await navigate({ to: '/admin/jobs/$name', params: { name: jobName } })
    } else {
      await navigate({ to: '/portal/jobs/detail/$name', params: { name: jobName } })
    }
    await saveToolOutput(messageID, part, { status: 'accepted', jobName })
    setOpen(false)
  }

  const applyFormPatch = async (
    messageID: string,
    part: AgentToolPart,
    patches: readonly AgentFormPatch[]
  ) => {
    if (!formRuntime) throw new Error(t('craterAgent.formChanged'))
    const changedPaths = await formRuntime.apply(patches)
    if (changedPaths.length === 0) throw new Error(t('craterAgent.formChanged'))
    await saveToolOutput(messageID, part, { status: 'accepted', changedPaths })
    toast.success(t('craterAgent.formApplied'))
    return changedPaths
  }

  const rejectTool = async (messageID: string, part: AgentToolPart) => {
    await saveToolOutput(messageID, part, { status: 'rejected' })
  }

  const handleOpenChange = (nextOpen: boolean) => {
    if (nextOpen) stickToBottomRef.current = true
    else clearEntryRequest(undefined)
    setOpen(nextOpen)
  }

  return (
    <Sheet open={open} onOpenChange={handleOpenChange}>
      {floatingButtonEnabled && (
        <SheetTrigger asChild>
          <Button
            aria-label={t('craterAgent.open')}
            className="fixed right-6 bottom-6 z-40 size-14 rounded-full shadow-xl"
            size="icon"
          >
            <BotIcon className="size-6" />
          </Button>
        </SheetTrigger>
      )}
      <SheetContent
        className="w-auto max-w-[calc(100vw-1rem)] gap-0 p-0 sm:max-w-none"
        style={{ width: panelWidth }}
      >
        <ResizeHandle panelWidth={panelWidth} setPanelWidth={setPanelWidth} />
        <SheetHeader className="flex h-[5.75rem] shrink-0 flex-row items-center justify-between gap-4 border-b px-5 py-3 pr-14 text-left">
          <div className="min-w-0 space-y-1">
            <SheetTitle className="flex items-center gap-2 text-lg">
              <BotIcon className="text-primary size-5" />
              Crater-Agent
            </SheetTitle>
            <SheetDescription>{t('craterAgent.description')}</SheetDescription>
          </div>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            className="shrink-0"
            aria-label={
              historyOpen ? t('craterAgent.collapseHistory') : t('craterAgent.expandHistory')
            }
            onClick={() => setHistoryOpen((current) => !current)}
          >
            {historyOpen ? <PanelLeftCloseIcon /> : <PanelLeftOpenIcon />}
            <span className="hidden sm:inline">{t('craterAgent.history')}</span>
          </Button>
        </SheetHeader>
        <div className="flex min-h-0 flex-1 overflow-hidden">
          <div
            className={cn(
              'h-full shrink-0 overflow-hidden transition-[width] duration-200',
              historyOpen ? 'w-52' : 'w-0'
            )}
          >
            <ConversationSidebar
              conversations={conversations}
              activeConversationID={activeConversationID}
              busy={busy}
              onCreate={createNewConversation}
              onSelect={selectConversation}
              onRename={renameConversation}
              onDelete={deleteConversation}
            />
          </div>
          <section className="flex min-w-0 flex-1 flex-col">
            <ScrollArea
              viewportRef={viewportRef}
              className="min-h-0 flex-1 px-4 py-4 [&_[data-slot=scroll-area-viewport]]:!overflow-x-hidden [&_[data-slot=scroll-area-viewport]>div]:!block [&_[data-slot=scroll-area-viewport]>div]:!w-full [&_[data-slot=scroll-area-viewport]>div]:!min-w-0"
            >
              <div className="w-full max-w-full min-w-0 space-y-3 overflow-x-hidden pr-2">
                {messages.length === 0 && (
                  <div className="flex items-start gap-2.5">
                    <AgentAvatar />
                    <div className="bg-muted rounded-2xl p-4 text-sm leading-6">
                      {t('craterAgent.welcome')}
                    </div>
                  </div>
                )}
                {messages.map((message) => (
                  <ChatMessage
                    key={message.id}
                    identity={identity}
                    message={message}
                    user={user}
                    busy={busy}
                    editing={message.id === editingMessageID}
                    editingText={editingText}
                    onEditingTextChange={setEditingText}
                    onCopy={copyMessage}
                    onEdit={(messageID, text) => {
                      setEditingMessageID(messageID)
                      setEditingText(text)
                    }}
                    onCancelEdit={() => {
                      setEditingMessageID(undefined)
                      setEditingText('')
                    }}
                    onResend={resendEditedMessage}
                    runtime={formRuntime}
                    onNavigate={navigateFromTool}
                    onNavigateJob={navigateToJobDetailFromTool}
                    onApply={applyFormPatch}
                    onReject={rejectTool}
                  />
                ))}
                {status === 'submitted' && (
                  <div className="flex items-center gap-2.5">
                    <AgentAvatar />
                    <p className="text-muted-foreground text-sm">{t('craterAgent.thinking')}</p>
                  </div>
                )}
                {cancelled && (
                  <p className="text-muted-foreground text-sm">{t('craterAgent.cancelled')}</p>
                )}
                {error && <p className="text-destructive text-sm">{error.message}</p>}
              </div>
            </ScrollArea>
            <form className="border-t p-4" onSubmit={submit}>
              <div className="flex items-end gap-2">
                <Textarea
                  value={input}
                  rows={1}
                  className="max-h-36 min-h-11 resize-none overflow-y-auto"
                  placeholder={
                    awaitingConfirmation
                      ? t('craterAgent.confirmationRequiredPlaceholder')
                      : t('craterAgent.placeholder')
                  }
                  onChange={(event) => setInput(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' && !event.shiftKey) {
                      event.preventDefault()
                      if (!busy && !awaitingConfirmation) event.currentTarget.form?.requestSubmit()
                    }
                  }}
                />
                <Button
                  type={generating ? 'button' : 'submit'}
                  size="icon"
                  className="size-11 shrink-0"
                  disabled={
                    generating
                      ? cancelled
                      : busy || awaitingConfirmation || input.trim().length === 0
                  }
                  aria-label={generating ? t('craterAgent.stop') : t('craterAgent.send')}
                  onClick={
                    generating
                      ? () => {
                          const request = activeRequestRef.current
                          if (request && !request.cancelPromise) {
                            request.cancelPromise = cancelActiveRun(request.conversationID)
                          }
                          void stop()
                          setCancelled(true)
                        }
                      : undefined
                  }
                >
                  {generating ? (
                    <SquareIcon className="size-4 fill-current" />
                  ) : (
                    <SendIcon className="size-4" />
                  )}
                </Button>
              </div>
              <p
                className={cn(
                  'mt-2 text-xs',
                  awaitingConfirmation
                    ? 'text-blue-700 dark:text-blue-300'
                    : 'text-muted-foreground'
                )}
              >
                {awaitingConfirmation
                  ? t('craterAgent.confirmationRequired')
                  : t('craterAgent.disclaimer')}
              </p>
            </form>
          </section>
        </div>
      </SheetContent>
    </Sheet>
  )
}

function buildPageContext(
  actionContext: AgentActionContext | undefined,
  runtime: AgentFormRuntime | undefined,
  resourceContext: AgentResourceContext | undefined,
  pathname: string
): AgentPageContext | undefined {
  const currentRuntime =
    runtime && normalizedRoute(runtime.capability.route) === normalizedRoute(pathname)
      ? runtime
      : undefined
  const currentAction =
    actionContext && normalizedRoute(actionContext.route) === normalizedRoute(pathname)
      ? actionContext
      : undefined
  const currentPageResource =
    resourceContext && normalizedRoute(resourceContext.route) === normalizedRoute(pathname)
      ? resourceContext
      : undefined
  const currentActionResource =
    currentAction?.resource &&
    normalizedRoute(currentAction.resource.route) === normalizedRoute(pathname)
      ? currentAction.resource
      : undefined
  const currentResource = currentPageResource ?? currentActionResource
  if (!currentAction && !currentRuntime && !currentResource) return undefined

  const effectiveAction: AgentActionContext = currentAction ?? {
    action: currentResource ? 'diagnose-job' : 'assist-form',
    title: currentResource ? 'Job diagnosis' : currentRuntime!.capability.title,
    prompt: currentResource
      ? 'Diagnose the current Crater job using available read-only evidence.'
      : 'Assist with the currently open Crater form.',
    route: currentResource ? currentResource.route : currentRuntime!.capability.route,
    capabilityId: currentResource ? undefined : currentRuntime!.capability.id,
  }
  const compatibleRuntime =
    currentRuntime &&
    (!effectiveAction.capabilityId || currentRuntime.capability.id === effectiveAction.capabilityId)
      ? currentRuntime
      : undefined
  return {
    route: effectiveAction.route,
    action: effectiveAction.action,
    title: effectiveAction.title,
    prompt: effectiveAction.prompt,
    form: compatibleRuntime
      ? {
          capabilityId: compatibleRuntime.capability.id,
          schemaVersion: compatibleRuntime.capability.schemaVersion,
          title: compatibleRuntime.capability.title,
          fields: compatibleRuntime.fields,
          currentValues: compatibleRuntime.snapshot(),
        }
      : undefined,
    resource:
      effectiveAction.action === 'diagnose-job' && currentResource
        ? {
            kind: currentResource.kind,
            jobName: currentResource.jobName,
            status: currentResource.status,
            jobType: currentResource.jobType,
          }
        : undefined,
  }
}

function normalizedRoute(route: string) {
  return route.length > 1 ? route.replace(/\/+$/, '') : route
}

async function agentFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const response = await fetch(input, init)
  if (response.ok) return response

  const payload = (await response
    .clone()
    .json()
    .catch(() => null)) as { msg?: string; error?: string } | null
  throw new AgentRequestError(
    payload?.msg || payload?.error || `AI service returned status ${response.status}`,
    response.status
  )
}

class AgentRequestError extends Error {
  constructor(
    message: string,
    readonly status: number
  ) {
    super(message)
    this.name = 'AgentRequestError'
  }
}

async function requestConversationTitle(
  identity: AgentIdentity,
  conversationID: string,
  firstMessage: string
) {
  const maximumAttempts = 2
  let lastError: unknown
  for (let attempt = 0; attempt < maximumAttempts; attempt += 1) {
    try {
      const response = await agentFetch('/api/v1/agent/title', {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${localStorage.getItem(ACCESS_TOKEN_KEY) ?? ''}`,
          'Content-Type': 'application/json',
          [agentIdentityHeader]: identity,
        },
        body: JSON.stringify({ conversationId: conversationID, firstMessage }),
      })
      const payload = (await response.json()) as { data?: { title?: string } }
      return payload.data?.title?.trim() ?? ''
    } catch (error) {
      lastError = error
      if (attempt + 1 < maximumAttempts) {
        await new Promise((resolve) => window.setTimeout(resolve, 500))
      }
    }
  }
  throw lastError
}
