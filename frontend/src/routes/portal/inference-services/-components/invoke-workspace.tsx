import {
  CircleAlertIcon,
  CodeIcon,
  EraserIcon,
  MessageSquareIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PlusIcon,
  SendHorizontalIcon,
  TerminalIcon,
  Trash2Icon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

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
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import { CopyButton } from '@/components/button/copy-button'
import CardTitle from '@/components/label/card-title'

import { KthenaService } from '@/services/api/inference'

import { ChatBubble, buildCurl, displayAPIBaseURL, draftChatSessionKey } from './detail-parts'
import { useInvokeWorkspace } from './use-invoke-workspace'

export function InvokeWorkspace({ service }: { service: KthenaService }) {
  const {
    t,
    prompt,
    setPrompt,
    isSessionRailCollapsed,
    setIsSessionRailCollapsed,
    pendingBySession,
    controllers,
    streamedBySession,
    messagesViewportRef,
    promptInputRef,
    sessions,
    isSessionsLoading,
    activeSession,
    activeSessionKey,
    displayedMessages,
    isActiveSessionPending,
    canSend,
    turnError,
    rawResponse,
    curl,
    startNewSession,
    selectSession,
    clearSession,
    isClearingSession,
    clearActiveSession,
    deletingSessionID,
    removeSession,
    sessionAction,
    setSessionAction,
    submitMessage,
  } = useInvokeWorkspace(service)
  return (
    <Tabs defaultValue="online" className="min-w-0 gap-3">
      <TabsList aria-label={t('kthena.detail.tabs.invoke')}>
        <TabsTrigger value="online">
          <MessageSquareIcon />
          {t('kthena.detail.onlineTest')}
        </TabsTrigger>
        <TabsTrigger value="api">
          <TerminalIcon />
          {t('kthena.detail.apiCall')}
        </TabsTrigger>
      </TabsList>
      <AlertDialog
        open={!!sessionAction}
        onOpenChange={(open) => {
          if (!open) setSessionAction(null)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.chat.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.chat.confirmDescription', {
                name:
                  sessions.find((item) => item.sessionId === sessionAction?.id)?.title ||
                  sessionAction?.id,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('kthena.actions.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (sessionAction?.kind === 'clear') clearSession(sessionAction.id)
                if (sessionAction?.kind === 'delete') removeSession(sessionAction.id)
                setSessionAction(null)
              }}
            >
              {t('kthena.actions.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <TabsContent forceMount value="online" className="min-w-0 data-[state=inactive]:hidden">
        <Card className="h-[clamp(28rem,calc(100dvh-18rem),48rem)] flex-row gap-0 overflow-hidden p-0">
          <aside
            aria-label={t('kthena.chat.sessions')}
            className={[
              'bg-muted/35 hidden shrink-0 flex-col border-r transition-[width] duration-200 ease-out motion-reduce:transition-none md:flex',
              isSessionRailCollapsed ? 'w-14' : 'w-48 lg:w-56',
            ].join(' ')}
          >
            <div
              className={[
                'border-b',
                isSessionRailCollapsed ? 'flex flex-col gap-2 p-2' : 'flex items-center gap-2 p-3',
              ].join(' ')}
            >
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="outline"
                    size={isSessionRailCollapsed ? 'icon' : 'default'}
                    className={[
                      'bg-background shadow-xs',
                      isSessionRailCollapsed ? 'w-full' : 'min-w-0 flex-1 justify-start',
                    ].join(' ')}
                    aria-label={t('kthena.chat.newSession')}
                    onClick={startNewSession}
                  >
                    <PlusIcon className="size-4" />
                    {!isSessionRailCollapsed && t('kthena.chat.newSession')}
                  </Button>
                </TooltipTrigger>
                {isSessionRailCollapsed && (
                  <TooltipContent side="right" sideOffset={6}>
                    {t('kthena.chat.newSession')}
                  </TooltipContent>
                )}
              </Tooltip>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="icon"
                    className={isSessionRailCollapsed ? 'w-full' : 'shrink-0'}
                    aria-controls="kthena-chat-session-list"
                    aria-expanded={!isSessionRailCollapsed}
                    aria-label={
                      isSessionRailCollapsed
                        ? t('kthena.chat.expandSessions')
                        : t('kthena.chat.collapseSessions')
                    }
                    onClick={() => setIsSessionRailCollapsed((current) => !current)}
                  >
                    {isSessionRailCollapsed ? (
                      <PanelLeftOpenIcon className="size-4" />
                    ) : (
                      <PanelLeftCloseIcon className="size-4" />
                    )}
                  </Button>
                </TooltipTrigger>
                <TooltipContent side="right" sideOffset={6}>
                  {isSessionRailCollapsed
                    ? t('kthena.chat.expandSessions')
                    : t('kthena.chat.collapseSessions')}
                </TooltipContent>
              </Tooltip>
            </div>
            <div
              id="kthena-chat-session-list"
              className="min-h-0 flex-1 space-y-1 overflow-y-auto p-2"
            >
              {isSessionRailCollapsed
                ? sessions.map((session, index) => {
                    const selected = session.sessionId === activeSession?.sessionId
                    const sessionTitle = session.title || t('kthena.chat.untitledSession')
                    const compactTitle = sessionTitle.trim().slice(0, 2)
                    return (
                      <Tooltip key={session.sessionId}>
                        <TooltipTrigger asChild>
                          <button
                            type="button"
                            aria-current={selected ? 'page' : undefined}
                            aria-label={sessionTitle}
                            className={[
                              'relative flex size-9 items-center justify-center rounded-md border text-[11px] leading-none font-semibold transition-colors',
                              selected
                                ? 'border-primary/30 bg-primary/10 text-primary'
                                : 'bg-background/70 text-muted-foreground hover:bg-background hover:text-foreground',
                            ].join(' ')}
                            onClick={() => selectSession(session.sessionId)}
                          >
                            <span aria-hidden="true">
                              {compactTitle || <MessageSquareIcon className="size-3.5" />}
                            </span>
                            <span className="bg-background absolute -right-1 -bottom-1 rounded-full border px-1 text-[8px] leading-3">
                              {index + 1}
                            </span>
                          </button>
                        </TooltipTrigger>
                        <TooltipContent
                          side="right"
                          sideOffset={6}
                          className="max-w-64 break-words"
                        >
                          {sessionTitle}
                        </TooltipContent>
                      </Tooltip>
                    )
                  })
                : sessions.map((session) => {
                    const selected = session.sessionId === activeSession?.sessionId
                    return (
                      <div
                        key={session.sessionId}
                        className={[
                          'group flex min-w-0 items-center gap-1 rounded-lg p-1',
                          selected ? 'bg-background border shadow-xs' : 'hover:bg-muted/70',
                        ].join(' ')}
                      >
                        <button
                          type="button"
                          aria-current={selected ? 'page' : undefined}
                          className="flex min-w-0 flex-1 items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm"
                          title={session.title || t('kthena.chat.untitledSession')}
                          onClick={() => selectSession(session.sessionId)}
                        >
                          <MessageSquareIcon className="text-muted-foreground size-3.5 shrink-0" />
                          <span className="truncate">
                            {session.title || t('kthena.chat.untitledSession')}
                          </span>
                        </button>
                        <Button
                          variant="ghost"
                          size="icon"
                          className="text-muted-foreground hover:text-destructive size-7 shrink-0 opacity-0 group-hover:opacity-100 focus-visible:opacity-100"
                          title={t('kthena.chat.deleteSession')}
                          aria-label={t('kthena.chat.deleteSession')}
                          disabled={
                            Boolean(pendingBySession[session.sessionId]) ||
                            deletingSessionID === session.sessionId
                          }
                          onClick={() =>
                            setSessionAction({ kind: 'delete', id: session.sessionId })
                          }
                        >
                          <Trash2Icon className="size-3.5" />
                        </Button>
                      </div>
                    )
                  })}
            </div>
          </aside>

          <section className="flex min-w-0 flex-1 flex-col">
            <header className="flex min-h-14 items-center justify-between gap-2 border-b px-3 py-2 sm:px-5">
              <div className="min-w-0 truncate text-sm font-semibold">
                {activeSession?.title || t('kthena.chat.untitledSession')}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <Badge
                  variant="outline"
                  className="hidden max-w-48 font-mono text-xs lg:inline-flex"
                >
                  <span className="truncate">{service.access?.modelName || service.name}</span>
                </Badge>
                {rawResponse && (
                  <Dialog>
                    <DialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={t('kthena.detail.rawResponse')}
                        title={t('kthena.detail.rawResponse')}
                      >
                        <CodeIcon className="size-4" />
                        <span className="hidden sm:inline">{t('kthena.detail.rawResponse')}</span>
                      </Button>
                    </DialogTrigger>
                    <DialogContent aria-describedby={undefined} className="min-w-0 sm:max-w-3xl">
                      <DialogHeader>
                        <DialogTitle>{t('kthena.detail.rawResponse')}</DialogTitle>
                      </DialogHeader>
                      <pre className="bg-muted max-h-[65dvh] overflow-auto rounded-md p-4 text-xs">
                        {JSON.stringify(rawResponse, null, 2)}
                      </pre>
                    </DialogContent>
                  </Dialog>
                )}
                <Button
                  variant="ghost"
                  size="icon"
                  className="size-8"
                  title={t('kthena.actions.clearChat')}
                  aria-label={t('kthena.actions.clearChat')}
                  disabled={
                    isActiveSessionPending || isClearingSession || !displayedMessages.length
                  }
                  onClick={clearActiveSession}
                >
                  <EraserIcon className="size-4" />
                </Button>
                <Button
                  variant="outline"
                  size="icon"
                  className="md:hidden"
                  title={t('kthena.chat.newSession')}
                  aria-label={t('kthena.chat.newSession')}
                  onClick={startNewSession}
                >
                  <PlusIcon className="size-4" />
                </Button>
              </div>
            </header>

            <div className="bg-muted/20 flex shrink-0 gap-1.5 overflow-x-auto border-b p-2 empty:hidden md:hidden">
              {sessions.map((session) => (
                <div
                  key={session.sessionId}
                  className="bg-background flex max-w-52 shrink-0 items-center rounded-md border p-0.5"
                >
                  <Button
                    variant={session.sessionId === activeSession?.sessionId ? 'secondary' : 'ghost'}
                    size="sm"
                    className="min-w-0 flex-1 justify-start"
                    onClick={() => selectSession(session.sessionId)}
                  >
                    <span className="truncate">
                      {session.title || t('kthena.chat.untitledSession')}
                    </span>
                  </Button>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="text-muted-foreground hover:text-destructive size-7 shrink-0"
                    title={t('kthena.chat.deleteSession')}
                    aria-label={t('kthena.chat.deleteSession')}
                    disabled={
                      Boolean(pendingBySession[session.sessionId]) ||
                      deletingSessionID === session.sessionId
                    }
                    onClick={() => setSessionAction({ kind: 'delete', id: session.sessionId })}
                  >
                    <Trash2Icon className="size-3.5" />
                  </Button>
                </div>
              ))}
            </div>

            <div
              ref={messagesViewportRef}
              className="bg-muted/20 min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-8"
              role="log"
              aria-live="polite"
            >
              {displayedMessages.length ? (
                <div className="mx-auto flex max-w-3xl flex-col gap-5">
                  {displayedMessages.map((message, index) => (
                    <ChatBubble
                      key={`${activeSession?.sessionId ?? draftChatSessionKey}-${message.role}-${index}`}
                      message={message}
                    />
                  ))}
                  {isActiveSessionPending && (
                    <ChatBubble
                      message={{
                        role: 'assistant',
                        content:
                          streamedBySession[activeSessionKey] || t('kthena.actions.requesting'),
                      }}
                      muted
                    />
                  )}
                </div>
              ) : (
                <div className="text-muted-foreground mx-auto flex h-full max-w-md flex-col items-center justify-center gap-3 text-center">
                  <div className="bg-primary/10 text-primary flex size-10 items-center justify-center rounded-lg">
                    <MessageSquareIcon className="size-5" />
                  </div>
                  <div className="text-foreground text-base font-medium">
                    {t('kthena.chat.emptyTitle')}
                  </div>
                  <p className="text-sm leading-6">{t('kthena.detail.responsePlaceholder')}</p>
                </div>
              )}
            </div>

            <div className="border-t p-3 sm:p-4">
              <div className="mx-auto max-w-3xl">
                {!canSend && (
                  <p role="status" className="text-muted-foreground mb-2 text-sm">
                    {t('kthena.chat.notReady')}
                  </p>
                )}
                {turnError && (
                  <p role="alert" className="text-destructive mb-2 flex items-start gap-2 text-sm">
                    <CircleAlertIcon className="mt-0.5 size-4 shrink-0" />
                    {turnError}
                  </p>
                )}
                <div className="bg-background focus-within:border-ring rounded-lg border p-2.5">
                  <Textarea
                    ref={promptInputRef}
                    value={prompt}
                    disabled={!canSend}
                    aria-label={t('kthena.detail.promptPlaceholder')}
                    onChange={(event) => setPrompt(event.target.value)}
                    onKeyDown={(event) => {
                      if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                        event.preventDefault()
                        submitMessage()
                      }
                    }}
                    className="min-h-14 resize-none border-0 bg-transparent px-1 py-0.5 shadow-none focus-visible:ring-0"
                    placeholder={t('kthena.detail.promptPlaceholder')}
                  />
                  <div className="mt-1 flex items-center justify-between gap-2">
                    <kbd className="text-muted-foreground hidden px-1 text-xs sm:block">
                      Ctrl / ⌘ + Enter
                    </kbd>
                    <div className="ml-auto flex flex-wrap gap-2">
                      {isActiveSessionPending && (
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => controllers.current[activeSessionKey]?.abort()}
                        >
                          {t('kthena.actions.cancel')}
                        </Button>
                      )}
                      <Button
                        size="sm"
                        disabled={
                          !canSend || isActiveSessionPending || isSessionsLoading || !prompt.trim()
                        }
                        onClick={submitMessage}
                      >
                        <SendHorizontalIcon className="size-4" />
                        {isActiveSessionPending
                          ? t('kthena.actions.requesting')
                          : t('kthena.actions.send')}
                      </Button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </section>
        </Card>
      </TabsContent>
      <TabsContent value="api" className="min-w-0">
        <InvokeAPI
          service={service}
          contextCurl={displayedMessages.length || prompt.trim() ? curl : undefined}
        />
      </TabsContent>
    </Tabs>
  )
}

function InvokeAPI({ service, contextCurl }: { service: KthenaService; contextCurl?: string }) {
  const { t } = useTranslation()
  const [useContext, setUseContext] = useState(false)
  const curl =
    (useContext && contextCurl) || buildCurl(service, [], t('kthena.detail.defaultPrompt'))
  return (
    <Card className="min-w-0 gap-4">
      <CardHeader>
        <CardTitle icon={TerminalIcon}>{t('kthena.detail.invokeInfo')}</CardTitle>
      </CardHeader>
      <CardContent className="grid min-w-0 gap-5 px-4 sm:px-6">
        <dl className="grid gap-3 md:grid-cols-[minmax(0,2fr)_minmax(0,1fr)]">
          {[
            ['Base URL', displayAPIBaseURL(service)],
            [t('kthena.detail.apiModel'), service.access?.modelName || service.name],
          ].map(([label, value]) => (
            <div key={label} className="min-w-0 space-y-1.5">
              <dt className="text-muted-foreground text-sm font-medium">{label}</dt>
              <dd className="bg-muted/40 flex min-h-10 items-start gap-2 rounded-md border px-3 py-2">
                <code className="min-w-0 flex-1 self-center text-xs leading-5 [overflow-wrap:anywhere]">
                  {value}
                </code>
                <CopyButton
                  content={value}
                  aria-label={t('kthena.detail.copyField', { field: label })}
                  copyMessage={t('kthena.copy.generic', { label })}
                  className="shrink-0"
                />
              </dd>
            </div>
          ))}
        </dl>
        <div className="flex flex-wrap items-baseline gap-x-3 gap-y-1 text-sm">
          <span className="font-medium">{t('kthena.detail.apiAuthentication')}</span>
          <p className="text-muted-foreground">{t('kthena.detail.invokeHint')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2 text-sm">
          <span className="font-medium">{t('kthena.detail.apiEndpoints')}</span>
          {['/chat/completions', '/completions', '/embeddings'].map((path) => (
            <Badge key={path} variant="secondary" className="font-mono">
              POST {path}
            </Badge>
          ))}
          <span className="text-muted-foreground">{t('kthena.detail.apiEndpointsHelp')}</span>
        </div>
        <div className="min-w-0 space-y-2 border-t pt-4">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="flex flex-wrap items-center gap-x-3 gap-y-1">
              <h3 className="text-sm font-medium">{t('kthena.detail.apiExample')}</h3>
              <code className="text-muted-foreground text-xs">POST /chat/completions</code>
            </div>
            <div className="flex items-center gap-2">
              {contextCurl && (
                <Button
                  variant={useContext ? 'secondary' : 'ghost'}
                  size="sm"
                  aria-pressed={useContext}
                  onClick={() => setUseContext((previous) => !previous)}
                >
                  {t('kthena.detail.curlContext')}
                </Button>
              )}
              <CopyButton
                variant="outline"
                content={curl}
                aria-label={t('kthena.detail.copyField', { field: 'curl' })}
                copyMessage={t('kthena.copy.generic', { label: 'curl' })}
              />
            </div>
          </div>
          <pre className="bg-muted/50 max-h-72 overflow-y-auto rounded-lg border p-4 font-mono text-xs leading-6 [overflow-wrap:anywhere] whitespace-pre-wrap">
            <code>{curl}</code>
          </pre>
        </div>
      </CardContent>
    </Card>
  )
}
