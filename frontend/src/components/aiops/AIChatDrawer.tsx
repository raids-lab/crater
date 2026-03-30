'use client'

import { useMutation } from '@tanstack/react-query'
import {
  AlertCircle,
  CheckCircle,
  ChevronDown,
  HelpCircle,
  Loader2,
  Send,
  Sparkles,
  X,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
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
  apiAdminChatMessage,
  apiAdminChatMessageLLM,
  apiChatMessage,
  apiChatMessageLLM,
} from '@/services/api/aiops'
import type { IChatRequest, IChatResponse, IDiagnosis } from '@/services/api/aiops'
import { connectAgentChat } from '@/services/api/agent'
import type { AgentSSEEvent } from '@/services/api/agent'

import { ConfirmActionCard } from './ConfirmActionCard'
import { ThinkingIndicator } from './ThinkingIndicator'
import { ToolCallCard } from './ToolCallCard'

import { cn } from '@/lib/utils'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  type?: 'text' | 'diagnosis' | 'suggestion'
  data?: IChatResponse['data']
  timestamp: Date
}

// ── Agent-mode message types ──────────────────────────────────────────────────

type AgentMessageKind =
  | 'user'
  | 'thinking'
  | 'tool_call'
  | 'message'
  | 'confirmation_required'
  | 'error'

interface AgentChatItem {
  id: string
  kind: AgentMessageKind
  /** For 'user' and 'message' kinds */
  text?: string
  /** For 'thinking' — may be partial/streaming */
  thinkingContent?: string
  /** For 'tool_call' */
  toolName?: string
  toolArgs?: Record<string, any>
  toolStatus?: 'executing' | 'done' | 'error'
  toolResult?: string
  /** For 'confirmation_required' */
  confirmId?: string
  confirmAction?: string
  confirmDescription?: string
  timestamp: Date
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
  const [agentItems, setAgentItems] = useState<AgentChatItem[]>([])
  const [agentStreaming, setAgentStreaming] = useState(false)
  const agentAbortRef = useRef<AbortController | null>(null)
  /** Current session ID — null means the backend will create a new one */
  const agentSessionIdRef = useRef<string | null>(null)

  /** Extract page context from the current URL */
  const getPageContext = () => {
    const pathname = window.location.pathname
    const jobMatch = pathname.match(/\/jobs\/detail\/([^/?#]+)/)
    return {
      url: pathname,
      jobName: jobMatch?.[1] ?? currentJobName,
    }
  }

  /** Cancel any running SSE stream */
  const cancelAgentStream = () => {
    agentAbortRef.current?.abort()
    agentAbortRef.current = null
    setAgentStreaming(false)
  }

  // Cancel stream when the drawer closes
  useEffect(() => {
    if (!isOpen) cancelAgentStream()
  }, [isOpen])

  // Cancel stream when switching away from agent mode
  useEffect(() => {
    if (chatMode !== 'agent') cancelAgentStream()
  }, [chatMode])

  const handleAgentSend = (messageText?: string) => {
    const textToSend = messageText ?? input.trim()
    if (!textToSend || agentStreaming) return

    setInput('')

    // Append user bubble
    const userItem: AgentChatItem = {
      id: `user-${Date.now()}`,
      kind: 'user',
      text: textToSend,
      timestamp: new Date(),
    }
    setAgentItems((prev) => [...prev, userItem])

    setAgentStreaming(true)

    // Placeholder for the live thinking item (updated in-place via id)
    const thinkingId = `thinking-${Date.now()}`

    const ctrl = connectAgentChat(
      agentSessionIdRef.current,
      textToSend,
      getPageContext(),
      // onEvent
      (event: AgentSSEEvent) => {
        switch (event.event) {
          case 'thinking': {
            const content: string =
              typeof event.data === 'string' ? event.data : event.data?.content ?? ''
            setAgentItems((prev) => {
              const existing = prev.find((i) => i.id === thinkingId)
              if (existing) {
                return prev.map((i) =>
                  i.id === thinkingId ? { ...i, thinkingContent: (i.thinkingContent ?? '') + content } : i
                )
              }
              const item: AgentChatItem = {
                id: thinkingId,
                kind: 'thinking',
                thinkingContent: content,
                timestamp: new Date(),
              }
              return [...prev, item]
            })
            break
          }

          case 'tool_call': {
            const toolName: string = event.data?.name ?? event.data?.toolName ?? 'unknown'
            const toolArgs: Record<string, any> = event.data?.args ?? event.data?.arguments ?? {}
            const toolCallId = event.data?.id ?? `tool-${Date.now()}`
            const item: AgentChatItem = {
              id: `toolcall-${toolCallId}`,
              kind: 'tool_call',
              toolName,
              toolArgs,
              toolStatus: 'executing',
              timestamp: new Date(),
            }
            setAgentItems((prev) => [...prev, item])
            break
          }

          case 'tool_result': {
            const toolCallId = event.data?.id ?? event.data?.toolCallId
            const result: string =
              typeof event.data?.result === 'string'
                ? event.data.result
                : JSON.stringify(event.data?.result ?? event.data ?? '')
            const isError: boolean = event.data?.isError ?? false
            setAgentItems((prev) =>
              prev.map((i) => {
                if (i.id === `toolcall-${toolCallId}`) {
                  return {
                    ...i,
                    toolStatus: isError ? 'error' : 'done',
                    toolResult: result,
                  }
                }
                return i
              })
            )
            break
          }

          case 'message': {
            const text: string =
              typeof event.data === 'string' ? event.data : event.data?.content ?? ''
            // Capture session id if provided
            if (event.data?.sessionId) {
              agentSessionIdRef.current = event.data.sessionId
            }
            const item: AgentChatItem = {
              id: `msg-${Date.now()}`,
              kind: 'message',
              text,
              timestamp: new Date(),
            }
            setAgentItems((prev) => [...prev, item])
            break
          }

          case 'confirmation_required': {
            const item: AgentChatItem = {
              id: `confirm-${Date.now()}`,
              kind: 'confirmation_required',
              confirmId: event.data?.confirmId ?? event.data?.id ?? '',
              confirmAction: event.data?.action ?? '',
              confirmDescription: event.data?.description ?? '',
              timestamp: new Date(),
            }
            setAgentItems((prev) => [...prev, item])
            break
          }

          default:
            break
        }
      },
      // onError
      (err: Error) => {
        const errItem: AgentChatItem = {
          id: `err-${Date.now()}`,
          kind: 'error',
          text: err.message,
          timestamp: new Date(),
        }
        setAgentItems((prev) => [...prev, errItem])
        setAgentStreaming(false)
      },
      // onDone
      () => {
        setAgentStreaming(false)
      },
    )

    agentAbortRef.current = ctrl
  }

  // Auto scroll to bottom when messages or agent items change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages, agentItems])

  // Chat mutation
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
        const backend = (error as { data?: { msg?: string; msgKey?: string } }).data
        if (backend?.msgKey) {
          message = t(backend.msgKey, { defaultValue: backend.msg || message })
        } else if (backend?.msg) {
          message = backend.msg
        }
      }
      const errorMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'assistant',
        content: t('aiops.chat.errorMessage', {
          message,
        }),
        type: 'text',
        timestamp: new Date(),
      }
      setMessages((prev) => [...prev, errorMessage])
    },
  })

  const handleSend = (messageText?: string) => {
    const textToSend = messageText || input.trim()
    if (!textToSend || chatMutation.isPending) return

    // Extract jobName if user provides it in format
    let jobName = currentJobName
    const jobNameMatch = textToSend.match(/(?:作业|job)[：:]\s*([a-zA-Z0-9-]+)/i)
    if (jobNameMatch) {
      jobName = jobNameMatch[1]
    }

    // Add user message
    const userMessage: ChatMessage = {
      id: Date.now().toString(),
      role: 'user',
      content: textToSend,
      timestamp: new Date(),
    }
    setMessages((prev) => [...prev, userMessage])

    // Send to API
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

  if (!isOpen) return null

  return (
    <>
      <div className="bg-background fixed inset-y-0 right-0 z-50 flex w-full flex-col border-l shadow-2xl sm:w-[500px]">
        {/* Header */}
        <div className="from-primary/5 to-primary/10 flex flex-none items-center justify-between border-b bg-gradient-to-r p-4">
          <div className="flex items-center gap-2">
            <Sparkles className="text-primary h-5 w-5" />
            <div>
              <h3 className="font-semibold">{t('aiops.chat.assistantName')}</h3>
              <p className="text-muted-foreground mt-0.5 text-xs">
                {chatMode === 'llm'
                  ? t('aiops.chat.llmBased')
                  : chatMode === 'agent'
                    ? t('aiops.agent.modeBased', { defaultValue: 'Agent 自主执行模式' })
                    : t('aiops.chat.ruleBased')}
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
              variant={chatMode === 'agent' ? 'default' : 'outline'}
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
        <ScrollArea className="min-h-0 flex-1 p-4">
          <div className="space-y-4">
            {chatMode === 'agent' ? (
              <>
                {agentItems.length === 0 && (
                  <div className="flex justify-start">
                    <div className="bg-muted max-w-[95%] rounded-lg px-4 py-3">
                      <p className="text-sm">
                        {t('aiops.agent.initialMessage', {
                          defaultValue:
                            '你好！我是 Crater Agent，可以自主执行操作来帮助你管理作业。请告诉我你需要什么帮助。',
                        })}
                      </p>
                    </div>
                  </div>
                )}
                {agentItems.map((item) => {
                  if (item.kind === 'user') {
                    return (
                      <div key={item.id} className="flex justify-end">
                        <div className="bg-primary text-primary-foreground max-w-[85%] rounded-lg px-4 py-2">
                          <p className="text-sm whitespace-pre-wrap">{item.text}</p>
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'thinking') {
                    return (
                      <div key={item.id} className="flex justify-start">
                        <div className="bg-muted max-w-[95%] rounded-lg px-4 py-3">
                          <ThinkingIndicator content={item.thinkingContent} />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'tool_call') {
                    return (
                      <div key={item.id} className="flex justify-start">
                        <div className="w-full max-w-[95%]">
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
                      <div key={item.id} className="flex justify-start">
                        <div className="bg-muted max-w-[95%] rounded-lg px-4 py-3">
                          <div className="markdown-content text-sm">
                            <ReactMarkdown
                              remarkPlugins={[remarkGfm]}
                              components={{
                                p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                                strong: ({ children }) => (
                                  <strong className="text-foreground font-semibold">
                                    {children}
                                  </strong>
                                ),
                                ul: ({ children }) => (
                                  <ul className="my-2 list-inside list-disc space-y-1">
                                    {children}
                                  </ul>
                                ),
                                ol: ({ children }) => (
                                  <ol className="my-2 list-inside list-decimal space-y-1">
                                    {children}
                                  </ol>
                                ),
                                li: ({ children }) => <li className="text-sm">{children}</li>,
                                code: ({ children, className }) => {
                                  const isInline = !className
                                  return isInline ? (
                                    <code className="bg-background rounded px-1 py-0.5 font-mono text-xs">
                                      {children}
                                    </code>
                                  ) : (
                                    <code className="bg-background block overflow-x-auto rounded p-2 font-mono text-xs whitespace-pre">
                                      {children}
                                    </code>
                                  )
                                },
                                h1: ({ children }) => (
                                  <h1 className="mb-2 text-lg font-bold">{children}</h1>
                                ),
                                h2: ({ children }) => (
                                  <h2 className="mb-2 text-base font-bold">{children}</h2>
                                ),
                                h3: ({ children }) => (
                                  <h3 className="mb-1 text-sm font-semibold">{children}</h3>
                                ),
                                h4: ({ children }) => (
                                  <h4 className="mb-1 text-sm font-semibold">{children}</h4>
                                ),
                                blockquote: ({ children }) => (
                                  <blockquote className="border-primary my-2 border-l-2 pl-3 italic">
                                    {children}
                                  </blockquote>
                                ),
                              }}
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
                      <div key={item.id} className="flex justify-start">
                        <div className="w-full max-w-[95%]">
                          <ConfirmActionCard
                            confirmId={item.confirmId ?? ''}
                            action={item.confirmAction ?? ''}
                            description={item.confirmDescription ?? ''}
                            onConfirmed={() => {}}
                            onRejected={() => {}}
                          />
                        </div>
                      </div>
                    )
                  }

                  if (item.kind === 'error') {
                    return (
                      <div key={item.id} className="flex justify-start">
                        <div className="bg-destructive/10 border-destructive/30 max-w-[95%] rounded-lg border px-4 py-3">
                          <p className="text-destructive text-sm">
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
                {agentStreaming && agentItems[agentItems.length - 1]?.kind !== 'thinking' && (
                  <div className="flex justify-start">
                    <div className="bg-muted rounded-lg px-4 py-3">
                      <ThinkingIndicator />
                    </div>
                  </div>
                )}
              </>
            ) : (
              <>
                {messages.map((message) => (
                  <div
                    key={message.id}
                    className={cn('flex', message.role === 'user' ? 'justify-end' : 'justify-start')}
                  >
                    {message.role === 'user' ? (
                      <div className="bg-primary text-primary-foreground max-w-[85%] rounded-lg px-4 py-2">
                        <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                      </div>
                    ) : (
                      <div className="max-w-[95%] space-y-2">
                        <div className="bg-muted rounded-lg px-4 py-3">
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
                                <div className="markdown-content text-sm">
                                  <ReactMarkdown
                                    remarkPlugins={[remarkGfm]}
                                    components={{
                                      p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                                      strong: ({ children }) => (
                                        <strong className="text-foreground font-semibold">
                                          {children}
                                        </strong>
                                      ),
                                      ul: ({ children }) => (
                                        <ul className="my-2 list-inside list-disc space-y-1">
                                          {children}
                                        </ul>
                                      ),
                                      ol: ({ children }) => (
                                        <ol className="my-2 list-inside list-decimal space-y-1">
                                          {children}
                                        </ol>
                                      ),
                                      li: ({ children }) => <li className="text-sm">{children}</li>,
                                      code: ({ children, className }) => {
                                        const isInline = !className
                                        return isInline ? (
                                          <code className="bg-background rounded px-1 py-0.5 font-mono text-xs">
                                            {children}
                                          </code>
                                        ) : (
                                          <code className="bg-background block overflow-x-auto rounded p-2 font-mono text-xs whitespace-pre">
                                            {children}
                                          </code>
                                        )
                                      },
                                      h1: ({ children }) => (
                                        <h1 className="mb-2 text-lg font-bold">{children}</h1>
                                      ),
                                      h2: ({ children }) => (
                                        <h2 className="mb-2 text-base font-bold">{children}</h2>
                                      ),
                                      h3: ({ children }) => (
                                        <h3 className="mb-1 text-sm font-semibold">{children}</h3>
                                      ),
                                      h4: ({ children }) => (
                                        <h4 className="mb-1 text-sm font-semibold">{children}</h4>
                                      ),
                                      blockquote: ({ children }) => (
                                        <blockquote className="border-primary my-2 border-l-2 pl-3 italic">
                                          {children}
                                        </blockquote>
                                      ),
                                    }}
                                  >
                                    {cleanedContent}
                                  </ReactMarkdown>
                                </div>
                              </>
                            )
                          })()}
                        </div>
                        {message.type === 'diagnosis' && isDiagnosisData(message.data) && (
                          <DiagnosisCard diagnosis={message.data} />
                        )}
                      </div>
                    )}
                  </div>
                ))}
                {chatMutation.isPending && (
                  <div className="flex justify-start">
                    <div className="bg-muted flex items-center gap-2 rounded-lg px-4 py-2">
                      <Loader2 className="h-4 w-4 animate-spin" />
                      <span className="text-muted-foreground text-sm">{t('aiops.chat.thinking')}</span>
                    </div>
                  </div>
                )}
              </>
            )}
            {/* Invisible element for auto-scroll */}
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Smart Prompts - only shown for rule/llm modes */}
        {chatMode !== 'agent' && (
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
        )}

        {/* Input */}
        <div className="bg-background flex-none border-t p-4">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault()
                  chatMode === 'agent' ? handleAgentSend() : handleSend()
                }
              }}
              placeholder={
                chatMode === 'agent'
                  ? t('aiops.agent.inputPlaceholder', {
                      defaultValue: '告诉 Agent 你想做什么...',
                    })
                  : t('aiops.chat.inputPlaceholder')
              }
              disabled={chatMode === 'agent' ? agentStreaming : chatMutation.isPending}
              className="flex-1"
            />
            <Button
              onClick={() => (chatMode === 'agent' ? handleAgentSend() : handleSend())}
              disabled={
                chatMode === 'agent'
                  ? agentStreaming || !input.trim()
                  : chatMutation.isPending || !input.trim()
              }
              size="icon"
              aria-label={t('common.send')}
            >
              {chatMode === 'agent' && agentStreaming ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
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

// Help Content Component
function HelpContent() {
  const { t } = useTranslation()

  return (
    <div className="space-y-6 text-sm">
      {/* Usage Guide */}
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

      {/* Failure Statistics */}
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

      {/* Self-Check Guide */}
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

      {/* Supported Formats */}
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

      {/* Limitations */}
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

// Stat Item Component
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

// Diagnosis Card Component
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

  // Check if it's a user code issue
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
