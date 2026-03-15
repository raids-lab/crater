/**
 * Enhanced AI Chat Drawer with intelligent prompts and help dialog
 */
'use client'

import { useState, useRef, useEffect } from 'react'
import { Send, X, Loader2, AlertCircle, CheckCircle, HelpCircle, Sparkles, ChevronDown } from 'lucide-react'
import { useMutation } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Card } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible'
import { cn } from '@/lib/utils'

import {
  apiAdminChatMessage,
  apiAdminChatMessageLLM,
  apiChatMessage,
  apiChatMessageLLM,
} from '@/services/api/aiops'
import type { IChatRequest, IChatResponse } from '@/services/api/aiops'

interface ChatMessage {
  id: string
  role: 'user' | 'assistant'
  content: string
  type?: 'text' | 'diagnosis' | 'suggestion'
  data?: any
  timestamp: Date
}

interface AIChatDrawerProps {
  isOpen: boolean
  onClose: () => void
  currentJobName?: string
}

const ADMIN_CHAT_HINT = '管理员账号可前往 Admin 页面使用 Chat 诊断（/admin/aiops）。'

function getAssistantContentForDisplay(content: string) {
  const showAdminHint = content.includes('仅管理员可查看所有作业')
  const cleanedContent = content.replace(ADMIN_CHAT_HINT, '').trim()
  return { showAdminHint, cleanedContent }
}

export function AIChatDrawer({ isOpen, onClose, currentJobName }: AIChatDrawerProps) {
  const { t } = useTranslation()

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
        { text: t('aiops.chat.prompt.evicted'), icon: '🔧', hint: t('aiops.chat.prompt.hint.evicted') },
        { text: t('aiops.chat.prompt.mountFailed'), icon: '💾', hint: t('aiops.chat.prompt.hint.mountFailed') },
        { text: t('aiops.chat.prompt.exit127'), icon: '⚙️', hint: t('aiops.chat.prompt.hint.exit127') },
        { text: t('aiops.chat.prompt.oom'), icon: '💥', hint: t('aiops.chat.prompt.hint.oom') },
      ],
    },
    {
      id: 'job-diagnosis',
      category: t('aiops.chat.prompt.category.jobDiagnosis'),
      prompts: [
        { text: t('aiops.chat.prompt.diagnoseJob'), icon: '🔍', hint: t('aiops.chat.prompt.hint.jobName') },
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
  const [chatMode, setChatMode] = useState<'rule' | 'llm'>('rule')
  const isAdminRoute = typeof window !== 'undefined' && window.location.pathname.startsWith('/admin')
  const [expandedCategories, setExpandedCategories] = useState<string[]>(['top-issues'])
  const messagesEndRef = useRef<HTMLDivElement>(null)

  // Auto scroll to bottom when messages change
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [messages])

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
    onError: (error: any) => {
      const errorMessage: ChatMessage = {
        id: Date.now().toString(),
        role: 'assistant',
        content: t('aiops.chat.errorMessage', { message: error.message || t('aiops.common.unknownError') }),
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

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  const toggleCategory = (categoryId: string) => {
    setExpandedCategories((prev) =>
      prev.includes(categoryId) ? prev.filter((c) => c !== categoryId) : [...prev, categoryId]
    )
  }

  if (!isOpen) return null

  return (
    <>
      <div className="fixed inset-y-0 right-0 w-full sm:w-[500px] bg-background border-l shadow-2xl flex flex-col z-50">
        {/* Header */}
        <div className="flex-none p-4 border-b bg-gradient-to-r from-primary/5 to-primary/10 flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Sparkles className="h-5 w-5 text-primary" />
            <div>
              <h3 className="font-semibold">{t('aiops.chat.assistantName')}</h3>
              <p className="text-xs text-muted-foreground mt-0.5">
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
            <Dialog open={showHelp} onOpenChange={setShowHelp}>
              <DialogTrigger asChild>
                <Button variant="ghost" size="icon" className="h-8 w-8">
                  <HelpCircle className="h-4 w-4" />
                </Button>
              </DialogTrigger>
              <DialogContent
                showCloseButton={false}
                className="max-w-2xl max-h-[80vh] p-0 overflow-hidden"
              >
                <div className="sticky top-0 z-10 border-b bg-background px-6 py-4 pr-12">
                  <DialogHeader>
                    <DialogTitle className="flex items-center gap-2">
                      <Sparkles className="h-5 w-5 text-primary" />
                      {t('aiops.chat.helpTitle')}
                    </DialogTitle>
                    <DialogDescription>{t('aiops.chat.helpDesc')}</DialogDescription>
                  </DialogHeader>
                  <DialogClose asChild>
                    <Button variant="ghost" size="icon" className="absolute right-4 top-4 h-8 w-8">
                      <X className="h-4 w-4" />
                    </Button>
                  </DialogClose>
                </div>
                <div className="overflow-y-auto px-6 py-4">
                  <HelpContent />
                </div>
              </DialogContent>
            </Dialog>
            <Button variant="ghost" size="icon" className="h-8 w-8" onClick={onClose}>
              <X className="h-4 w-4" />
            </Button>
          </div>
        </div>

        {/* Messages */}
        <ScrollArea className="flex-1 min-h-0 p-4">
          <div className="space-y-4">
            {messages.map((message) => (
              <div
                key={message.id}
                className={cn('flex', message.role === 'user' ? 'justify-end' : 'justify-start')}
              >
                {message.role === 'user' ? (
                  <div className="bg-primary text-primary-foreground rounded-lg px-4 py-2 max-w-[85%]">
                    <p className="text-sm whitespace-pre-wrap">{message.content}</p>
                  </div>
                ) : (
                  <div className="max-w-[95%] space-y-2">
                    <div className="bg-muted rounded-lg px-4 py-3">
                      {(() => {
                        const { showAdminHint, cleanedContent } = getAssistantContentForDisplay(
                          message.content
                        )
                        return (
                          <>
                            {showAdminHint && (
                              <div className="mb-2 rounded-md border bg-background/70 px-3 py-1.5 text-xs text-muted-foreground">
                                {ADMIN_CHAT_HINT}
                              </div>
                            )}
                            <div className="text-sm markdown-content">
                              <ReactMarkdown
                                remarkPlugins={[remarkGfm]}
                                components={{
                                  p: ({ children }) => <p className="mb-2 last:mb-0">{children}</p>,
                                  strong: ({ children }) => <strong className="font-semibold text-foreground">{children}</strong>,
                                  ul: ({ children }) => <ul className="list-disc list-inside space-y-1 my-2">{children}</ul>,
                                  ol: ({ children }) => <ol className="list-decimal list-inside space-y-1 my-2">{children}</ol>,
                                  li: ({ children }) => <li className="text-sm">{children}</li>,
                                  code: ({ children, className }) => {
                                    const isInline = !className
                                    return isInline ? (
                                      <code className="bg-background px-1 py-0.5 rounded text-xs font-mono">
                                        {children}
                                      </code>
                                    ) : (
                                      <code className="block bg-background p-2 rounded text-xs font-mono overflow-x-auto whitespace-pre">
                                        {children}
                                      </code>
                                    )
                                  },
                                  h1: ({ children }) => <h1 className="text-lg font-bold mb-2">{children}</h1>,
                                  h2: ({ children }) => <h2 className="text-base font-bold mb-2">{children}</h2>,
                                  h3: ({ children }) => <h3 className="text-sm font-semibold mb-1">{children}</h3>,
                                  h4: ({ children }) => <h4 className="text-sm font-semibold mb-1">{children}</h4>,
                                  blockquote: ({ children }) => (
                                    <blockquote className="border-l-2 border-primary pl-3 my-2 italic">{children}</blockquote>
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
                    {message.type === 'diagnosis' && message.data && (
                      <DiagnosisCard diagnosis={message.data} />
                    )}
                  </div>
                )}
              </div>
            ))}
            {chatMutation.isPending && (
              <div className="flex justify-start">
                <div className="bg-muted rounded-lg px-4 py-2 flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span className="text-sm text-muted-foreground">{t('aiops.chat.thinking')}</span>
                </div>
              </div>
            )}
            {/* Invisible element for auto-scroll */}
            <div ref={messagesEndRef} />
          </div>
        </ScrollArea>

        {/* Smart Prompts - Collapsible */}
        <div className="flex-none border-t bg-muted/30 max-h-[30vh] overflow-y-auto">
          <div className="p-3 space-y-2">
            {smartPrompts.map((category) => (
              <Collapsible
                key={category.id}
                open={expandedCategories.includes(category.id)}
                onOpenChange={() => toggleCategory(category.id)}
              >
                <CollapsibleTrigger asChild>
                  <Button variant="ghost" size="sm" className="w-full justify-between h-7 text-xs font-medium">
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
                        className="justify-start h-auto py-1.5 px-2 text-xs text-left whitespace-normal"
                        title={prompt.hint}
                      >
                        <span className="mr-1.5">{prompt.icon}</span>
                        <span className="flex-1">{prompt.text}</span>
                        {prompt.hint && (
                          <span className="text-[10px] text-muted-foreground ml-1 hidden sm:inline">
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
        <div className="flex-none p-4 border-t bg-background">
          <div className="flex gap-2">
            <Input
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyPress}
              placeholder={t('aiops.chat.inputPlaceholder')}
              disabled={chatMutation.isPending}
              className="flex-1"
            />
            <Button onClick={() => handleSend()} disabled={chatMutation.isPending || !input.trim()} size="icon">
              <Send className="h-4 w-4" />
            </Button>
          </div>
          {currentJobName && (
            <div className="mt-2 text-xs text-muted-foreground flex items-center gap-1">
              <span>{t('aiops.chat.currentJob')}</span>
              <Badge variant="secondary" className="text-xs font-mono">
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
        <h4 className="font-semibold mb-2 text-base">{t('aiops.chat.help.usageTitle')}</h4>
        <div className="space-y-3 text-muted-foreground">
          <div>
            <p className="font-medium text-foreground mb-1">{t('aiops.chat.help.usage1Title')}</p>
            <p>{t('aiops.chat.help.usage1Desc')}</p>
          </div>
          <div>
            <p className="font-medium text-foreground mb-1">{t('aiops.chat.help.usage2Title')}</p>
            <p>{t('aiops.chat.help.usage2Desc')}</p>
          </div>
          <div>
            <p className="font-medium text-foreground mb-1">{t('aiops.chat.help.usage3Title')}</p>
            <p>
              {t('aiops.chat.help.usage3Desc')}: <code className="bg-muted px-1 py-0.5 rounded">job:jpt-xxx-xxx</code> /{' '}
              <code className="bg-muted px-1 py-0.5 rounded">analyze job jpt-xxx-xxx</code>
            </p>
          </div>
        </div>
      </section>

      {/* Failure Statistics */}
      <section>
        <h4 className="font-semibold mb-2 text-base">{t('aiops.chat.help.statsTitle')}</h4>
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
        <h4 className="font-semibold mb-2 text-base">{t('aiops.chat.help.selfCheckTitle')}</h4>
        <div className="bg-yellow-500/10 border border-yellow-500/20 rounded-lg p-3 space-y-2">
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
        <h4 className="font-semibold mb-2 text-base">{t('aiops.chat.help.supportedInputTitle')}</h4>
        <div className="space-y-1.5 font-mono text-xs bg-muted p-3 rounded-lg">
          <p>• {t('aiops.chat.help.input1')}</p>
          <p>• {t('aiops.chat.help.input2')}</p>
          <p>• {t('aiops.chat.help.input3')}</p>
          <p>• {t('aiops.chat.help.input4')}</p>
          <p>• {t('aiops.chat.help.input5')}</p>
        </div>
      </section>

      {/* Limitations */}
      <section>
        <h4 className="font-semibold mb-2 text-base">{t('aiops.chat.help.limitationsTitle')}</h4>
        <div className="text-muted-foreground text-xs space-y-1">
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
    <div className="flex items-start gap-2 p-2 bg-muted/50 rounded">
      <Badge variant="outline" className="flex-shrink-0 font-mono text-xs">
        #{rank}
      </Badge>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2">
          <span className="font-medium text-sm">{type}</span>
          <Badge variant="secondary" className="text-xs">
            {percentage}
          </Badge>
          {isUserIssue && (
            <Badge variant="destructive" className="text-[10px] h-4">
              {t('aiops.chat.selfCheckShort')}
            </Badge>
          )}
        </div>
        <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
      </div>
    </div>
  )
}

// Diagnosis Card Component
function DiagnosisCard({ diagnosis }: { diagnosis: any }) {
  const { t } = useTranslation()

  const severityColors = {
    critical: 'destructive',
    error: 'destructive',
    warning: 'default',
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
    <Card className="p-4 space-y-3">
      <div className="flex items-start gap-2">
        <Icon className="h-5 w-5 mt-0.5 flex-shrink-0" />
        <div className="flex-1 space-y-1">
          <div className="flex items-center gap-2 flex-wrap">
            <h4 className="font-semibold text-sm">{diagnosis.category}</h4>
            <Badge variant={severity}>
              {diagnosis.severity}
            </Badge>
            <Badge variant="outline" className="text-xs">
              {diagnosis.confidence}
            </Badge>
            {isUserCodeIssue && (
              <Badge variant="destructive" className="text-xs">
                {t('aiops.chat.selfCheckCode')}
              </Badge>
            )}
          </div>
          <p className="text-sm text-muted-foreground">{diagnosis.diagnosis}</p>
        </div>
      </div>
      {diagnosis.solution && (
        <div className="bg-muted/50 rounded p-3">
          <p className="text-xs font-medium mb-1">{t('aiops.chat.solution')}</p>
          <p className="text-xs whitespace-pre-wrap">{diagnosis.solution}</p>
        </div>
      )}
      {isUserCodeIssue && (
        <div className="bg-yellow-500/10 border border-yellow-500/20 rounded p-2">
          <p className="text-xs text-yellow-700 dark:text-yellow-300 flex items-center gap-1">
            <AlertCircle className="h-3 w-3" />
            <strong>{t('aiops.chat.tipLabel')}</strong>{t('aiops.chat.tipText')}
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
              <div key={i} className="text-xs text-muted-foreground bg-muted/30 rounded p-2">
                {event}
              </div>
            ))}
          </div>
        </div>
      )}
    </Card>
  )
}
