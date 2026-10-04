import {
  EraserIcon,
  MessageSquareIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PlayIcon,
  PlusIcon,
  TerminalIcon,
  Trash2Icon,
} from 'lucide-react'

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
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import { CopyButton } from '@/components/button/copy-button'
import CardTitle from '@/components/label/card-title'

import { KthenaService } from '@/services/api/inference'

import { ChatBubble, InvocationValue, displayAPIBaseURL, draftChatSessionKey } from './detail-parts'
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
    <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
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
      <Card className="overflow-hidden shadow-sm">
        <div className="flex min-h-[34rem] xl:h-[calc(100dvh-24rem)] xl:min-h-[38rem]">
          <aside
            aria-label={t('kthena.chat.sessions')}
            className={[
              'bg-muted/35 hidden shrink-0 flex-col border-r transition-[width] duration-200 ease-out motion-reduce:transition-none md:flex',
              isSessionRailCollapsed ? 'w-14' : 'w-56',
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
            {!isSessionRailCollapsed && (
              <>
                <div className="text-muted-foreground border-t px-3 py-2.5 font-mono text-xs">
                  <div className="truncate">{service.access?.modelName || service.name}</div>
                </div>
              </>
            )}
          </aside>

          <section className="flex min-w-0 flex-1 flex-col">
            <header className="bg-background flex min-h-16 items-center justify-between gap-3 border-b px-4 py-3 sm:px-5">
              <div className="min-w-0">
                <div className="text-muted-foreground text-xs">{t('kthena.detail.onlineTest')}</div>
                <div className="truncate text-sm font-semibold">
                  {activeSession?.title || t('kthena.chat.untitledSession')}
                </div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <Badge
                  variant="outline"
                  className="hidden max-w-48 truncate font-mono text-xs sm:inline-flex"
                >
                  {service.access?.modelName || service.name}
                </Badge>
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

            <div className="bg-muted/20 flex gap-1.5 overflow-x-auto border-b p-2 md:hidden">
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
              className="bg-muted/15 min-h-0 flex-1 overflow-y-auto px-4 py-6 sm:px-8"
              role="log"
              aria-live="polite"
            >
              {displayedMessages.length ? (
                <div className="mx-auto flex max-w-4xl flex-col gap-6">
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
                  <div className="bg-background flex size-11 items-center justify-center rounded-2xl border shadow-sm">
                    <MessageSquareIcon className="text-primary size-5" />
                  </div>
                  <div className="text-foreground text-base font-medium">
                    {t('kthena.chat.emptyTitle')}
                  </div>
                  <p className="text-sm leading-6">{t('kthena.detail.responsePlaceholder')}</p>
                </div>
              )}
            </div>

            <div className="bg-background border-t p-3 sm:p-4">
              <div className="bg-background mx-auto max-w-4xl rounded-2xl border p-3 shadow-sm">
                <Textarea
                  ref={promptInputRef}
                  value={prompt}
                  onChange={(event) => setPrompt(event.target.value)}
                  onKeyDown={(event) => {
                    if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') {
                      event.preventDefault()
                      submitMessage()
                    }
                  }}
                  className="min-h-20 resize-none border-0 bg-transparent p-0 shadow-none focus-visible:ring-0"
                  placeholder={t('kthena.detail.promptPlaceholder')}
                />
                <div className="mt-3 flex flex-wrap items-center justify-between gap-2 border-t pt-3">
                  <span className="text-muted-foreground hidden min-w-0 truncate font-mono text-xs sm:block">
                    {service.access?.modelName || service.name}
                  </span>
                  <div className="ml-auto flex flex-wrap gap-2">
                    {isActiveSessionPending && (
                      <Button
                        variant="outline"
                        onClick={() => controllers.current[activeSessionKey]?.abort()}
                      >
                        {t('kthena.actions.cancel')}
                      </Button>
                    )}
                    <Button
                      variant="outline"
                      disabled={
                        isActiveSessionPending ||
                        isClearingSession ||
                        displayedMessages.length === 0
                      }
                      onClick={clearActiveSession}
                    >
                      <EraserIcon className="size-4" />
                      {t('kthena.actions.clearChat')}
                    </Button>
                    <Button
                      disabled={isActiveSessionPending || isSessionsLoading || !prompt.trim()}
                      onClick={submitMessage}
                    >
                      <PlayIcon className="size-4" />
                      {isActiveSessionPending
                        ? t('kthena.actions.requesting')
                        : t('kthena.actions.send')}
                    </Button>
                  </div>
                </div>
              </div>
              {rawResponse && (
                <details className="text-muted-foreground bg-muted/20 mx-auto mt-3 max-w-4xl rounded-lg border p-3 text-xs">
                  <summary className="cursor-pointer">{t('kthena.detail.rawResponse')}</summary>
                  <pre className="bg-background mt-3 max-h-64 overflow-auto rounded-md border p-3">
                    {JSON.stringify(rawResponse, null, 2)}
                  </pre>
                </details>
              )}
            </div>
          </section>
        </div>
      </Card>

      <aside className="grid gap-4 xl:sticky xl:top-4">
        <Card className="overflow-hidden shadow-sm">
          <CardHeader className="bg-muted/20 border-b py-4">
            <CardTitle icon={TerminalIcon}>{t('kthena.detail.invokeInfo')}</CardTitle>
          </CardHeader>
          <CardContent className="grid gap-5 p-4">
            <div className="grid gap-3">
              <InvocationValue label="Crater API" value={displayAPIBaseURL(service)} />
              <InvocationValue
                label={t('kthena.detail.routeModelName')}
                value={service.access?.modelName || service.name}
              />
            </div>
            <div className="min-w-0 border-t pt-4">
              <div className="mb-2 flex items-center justify-between gap-3">
                <div className="min-w-0">
                  <div className="font-mono text-xs font-medium">curl</div>
                  <div className="text-muted-foreground mt-0.5 text-xs">
                    {t('kthena.detail.curlContext')}
                  </div>
                </div>
                <CopyButton
                  content={curl}
                  copyMessage={t('kthena.copy.generic', { label: 'curl' })}
                  className="bg-muted hover:bg-muted-foreground/10 shrink-0 rounded-md border"
                />
              </div>
              <pre className="bg-muted/60 text-foreground max-h-52 overflow-auto rounded-lg border p-3 font-mono text-xs leading-5 whitespace-pre">
                {curl}
              </pre>
            </div>
            <p className="text-muted-foreground bg-muted/30 rounded-lg border px-3 py-2.5 text-xs leading-5">
              {t('kthena.detail.invokeHint')}
            </p>
          </CardContent>
        </Card>
      </aside>
    </div>
  )
}
