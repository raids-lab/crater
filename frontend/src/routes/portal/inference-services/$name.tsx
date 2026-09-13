import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  BotIcon,
  BracesIcon,
  CalendarIcon,
  CheckCircle2Icon,
  CopyIcon,
  CpuIcon,
  EraserIcon,
  GaugeIcon,
  MessageSquareIcon,
  NetworkIcon,
  PanelLeftCloseIcon,
  PanelLeftOpenIcon,
  PlayIcon,
  PlusIcon,
  ServerIcon,
  TerminalIcon,
  Trash2Icon,
  UserRoundIcon,
} from 'lucide-react'
import { type ReactNode, useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Textarea } from '@/components/ui/textarea'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import KthenaStatusBadge from '@/components/badge/kthena-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { CopyButton } from '@/components/button/copy-button'
import { TimeDistance } from '@/components/custom/time-distance'
import CardTitle from '@/components/label/card-title'
import TooltipCopy from '@/components/label/tooltop-copy'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import NotFound from '@/components/placeholder/not-found'

import {
  ChatCompletionReq,
  ChatCompletionResp,
  KthenaConversation,
  KthenaDiagnostic,
  KthenaResource,
  KthenaRuntimePod,
  KthenaService,
  apiCreateKthenaConversation,
  apiCreateKthenaConversationTurn,
  apiDeleteKthenaConversation,
  apiDeleteKthenaService,
  apiGetKthenaService,
  apiListKthenaConversations,
  apiUpdateKthenaConversation,
} from '@/services/api/inference'

import { atomUserContext, atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import { REFETCH_INTERVAL } from '@/lib/constants'

import ResourceUsage from './-components/resource-usage'

export const Route = createFileRoute('/portal/inference-services/$name')({
  component: KthenaServiceDetailPage,
  errorComponent: () => <NotFound />,
  loader: ({ params }) => ({ crumb: params.name }),
})

const getResourcePhaseVariant = (
  phase: string
): 'default' | 'secondary' | 'outline' | 'destructive' => {
  if (phase === 'Ready' || phase === 'Active') return 'default'
  if (phase === 'Failed') return 'destructive'
  if (phase === 'Pending' || phase === 'Progressing') return 'secondary'
  return 'outline'
}

type ChatMessage = ChatCompletionReq['messages'][number]

type ChatSession = {
  id: string
  title: string
  messages: ChatMessage[]
  createdAt: string
  updatedAt: string
}

const emptyChatMessages: ChatMessage[] = []
const draftChatSessionKey = '__new__'
const getChatSessionStorageKey = (service: Pick<KthenaService, 'name' | 'namespace'>) =>
  `kthena-chat-sessions:${service.namespace}:${service.name}`
const getLegacyChatSessionStorageKey = (serviceName: string) =>
  `kthena-chat-sessions:${serviceName}`
const getLegacyChatStorageKey = (serviceName: string) => `kthena-chat-history:${serviceName}`
const getChatMigrationStorageKey = (
  userID: number | undefined,
  accountScope: string | undefined,
  service: Pick<KthenaService, 'name' | 'namespace'>
) =>
  `kthena-chat-server-migration:v1:${userID ?? 'anonymous'}:${accountScope ?? 'default'}:${service.namespace}:${service.name}`

function KthenaServiceDetailPage() {
  const { t } = useTranslation()
  const navigate = Route.useNavigate()
  const { name } = Route.useParams()
  const queryClient = useQueryClient()
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { data, isLoading } = useQuery({
    queryKey: ['kthena/inference-services', name],
    queryFn: () => apiGetKthenaService(name).then((res) => res.data),
    refetchInterval: REFETCH_INTERVAL,
  })

  const service = data
  const { mutate: deleteService, isPending: isDeleting } = useMutation({
    mutationFn: apiDeleteKthenaService,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      toast.success(t('kthena.delete.success'))
      navigate({ to: '/portal/inference-services' })
    },
    onError: showErrorToast,
  })

  if (isLoading || !service) {
    return (
      <div className="flex flex-col gap-4">
        <PageTitle title={t('kthena.detail.title')} description={t('kthena.list.syncing')}>
          <Button variant="outline" asChild>
            <Link to="/portal/inference-services">
              <ArrowLeftIcon className="size-4" />
              {t('kthena.actions.back')}
            </Link>
          </Button>
        </PageTitle>
        <Card>
          <CardContent className="text-muted-foreground flex h-40 items-center justify-center">
            {t('kthena.loading')}
          </CardContent>
        </Card>
      </div>
    )
  }

  const workerResources = getWorkerResources(service)
  const primaryPod = service.runtimePods?.find((pod) => pod.ready) ?? service.runtimePods?.[0]

  return (
    <div className="flex flex-col gap-6">
      <PageTitle title={service.name} description={t('kthena.detail.description')}>
        <Button variant="outline" asChild>
          <Link to="/portal/inference-services/new" search={{ clone: service.name }}>
            <CopyIcon className="size-4" />
            {t('kthena.actions.clone')}
          </Link>
        </Button>
        <Button variant="outline" asChild>
          <Link to="/portal/inference-services">
            <ArrowLeftIcon className="size-4" />
            {t('kthena.actions.backToList')}
          </Link>
        </Button>
        <Button variant="destructive" disabled={isDeleting} onClick={() => setDeleteOpen(true)}>
          <Trash2Icon className="size-4" />
          {t('kthena.actions.delete')}
        </Button>
      </PageTitle>

      <div className="text-muted-foreground grid grid-cols-1 gap-3 text-sm sm:grid-cols-2 md:grid-cols-4">
        <KthenaDetailMeta icon={ActivityIcon} label={t('kthena.table.status')}>
          <KthenaStatusBadge service={service} />
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={ServerIcon} label={t('kthena.table.backend')}>
          <span className="text-foreground font-medium">{service.backendType || '-'}</span>
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CheckCircle2Icon} label={t('kthena.detail.servedModel')}>
          <span className="text-foreground truncate font-mono text-sm" title={service.servedModel}>
            {service.servedModel || '-'}
          </span>
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CpuIcon} label={t('kthena.detail.runtimeResources')}>
          {Object.keys(workerResources).length ? (
            <ResourceBadges resources={workerResources} />
          ) : (
            <span>-</span>
          )}
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={UserRoundIcon} label={t('kthena.table.owner')}>
          {service.userInfo?.username ? (
            <UserLabel info={service.userInfo} />
          ) : (
            <span className="truncate">{service.owner || '-'}</span>
          )}
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CalendarIcon} label={t('kthena.table.createdAt')}>
          <TimeDistance date={service.createdAt} />
        </KthenaDetailMeta>
      </div>

      <Tabs defaultValue="invoke" className="gap-0">
        <TabsList className="tabs-list-underline">
          <TabsTrigger className="tabs-trigger-underline" value="invoke">
            <MessageSquareIcon className="size-4" />
            {t('kthena.detail.tabs.invoke')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="overview">
            <NetworkIcon className="size-4" />
            {t('kthena.detail.tabs.overview')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="resources">
            <BracesIcon className="size-4" />
            {t('kthena.detail.tabs.resources')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="usage">
            <GaugeIcon className="size-4" />
            {t('kthena.detail.tabs.usage')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="diagnostics">
            <AlertTriangleIcon className="size-4" />
            {t('kthena.detail.tabs.diagnostics')}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-0">
          <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <div className="bg-card overflow-hidden rounded-xl border">
              <OverviewSection icon={TerminalIcon} title={t('kthena.detail.invokeInfo')}>
                <OverviewField label={t('kthena.detail.apiBase')}>
                  <CopyValue value={displayAPIBaseURL(service)} />
                </OverviewField>
              </OverviewSection>
              <OverviewSection
                icon={NetworkIcon}
                title={t('kthena.detail.overviewTitle')}
                className="border-t"
              >
                <div className="grid gap-2 sm:grid-cols-2">
                  <OverviewField label={t('kthena.detail.routeModelName')}>
                    <CopyValue value={service.access?.modelName || service.name} />
                  </OverviewField>
                  <OverviewField label={t('kthena.detail.servedModel')}>
                    <CopyValue value={service.servedModel || '-'} />
                  </OverviewField>
                  <OverviewField label={t('kthena.detail.routeResource')}>
                    <CopyValue
                      value={service.access?.routeName || t('kthena.detail.waitingModelRoute')}
                    />
                  </OverviewField>
                  <OverviewField label="Router">
                    <CopyValue value={service.access?.routerService || '-'} />
                  </OverviewField>
                </div>
              </OverviewSection>
            </div>
            <RuntimePanel service={service} primaryPod={primaryPod} resources={workerResources} />
          </div>
        </TabsContent>

        <TabsContent value="resources" className="mt-0">
          <Card>
            <CardHeader>
              <CardTitle icon={BracesIcon}>{t('kthena.detail.tabs.resources')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-2">
              {service.resources?.length ? (
                service.resources.map((resource) => (
                  <ResourceCard key={`${resource.kind}/${resource.name}`} resource={resource} />
                ))
              ) : (
                <div className="text-muted-foreground text-sm">
                  {t('kthena.detail.noResources')}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="usage" className="mt-0">
          <ResourceUsage runtimePods={service.runtimePods} />
        </TabsContent>

        <TabsContent value="diagnostics" className="mt-0">
          <Card>
            <CardHeader>
              <CardTitle icon={AlertTriangleIcon}>{t('kthena.detail.diagnosticsTitle')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
              {service.diagnostics?.length ? (
                <>
                  <DiagnosticSummary diagnostics={service.diagnostics} />
                  <div className="before:bg-border relative grid gap-3 pl-5 before:absolute before:top-1 before:bottom-1 before:left-2 before:w-px">
                    {service.diagnostics.map((diagnostic, index) => (
                      <DiagnosticCard
                        key={`${diagnostic.resource}-${diagnostic.reason}-${index}`}
                        diagnostic={diagnostic}
                      />
                    ))}
                  </div>
                </>
              ) : (
                <div className="text-muted-foreground text-sm">
                  {t('kthena.detail.noDiagnostics')}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="invoke" className="mt-0">
          <InvokeWorkspace key={`${service.namespace}/${service.name}`} service={service} />
        </TabsContent>
      </Tabs>
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.delete.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.delete.description', { name: service.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>
              {t('kthena.actions.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={isDeleting}
              onClick={() => deleteService(service.name)}
            >
              {t('kthena.actions.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}

function KthenaDetailMeta({
  icon: Icon,
  label,
  children,
}: {
  icon: typeof ActivityIcon
  label: string
  children: ReactNode
}) {
  return (
    <div className="flex min-w-0 items-center">
      <Icon className="text-muted-foreground mr-1.5 size-4 shrink-0" />
      <span className="text-muted-foreground mr-1.5 truncate text-sm">{label}:</span>
      <span className="min-w-0 truncate">{children}</span>
    </div>
  )
}

function InvokeWorkspace({ service }: { service: KthenaService }) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const user = useAtomValue(atomUserInfo)
  const accountContext = useAtomValue(atomUserContext)
  const conversationQueryKey = useMemo(
    () =>
      [
        'kthena',
        'inference-services',
        service.namespace,
        service.name,
        'conversations',
        user?.id,
        accountContext?.space,
      ] as const,
    [accountContext?.space, service.name, service.namespace, user?.id]
  )
  const [legacySessions] = useState<ChatSession[]>(() => readStoredChatSessions(service))
  const legacyMigrationAttemptedRef = useRef(false)
  const [activeSessionID, setActiveSessionID] = useState<string | null | undefined>(undefined)
  const [prompt, setPrompt] = useState('')
  const [isSessionRailCollapsed, setIsSessionRailCollapsed] = useState(false)
  const [responseBySession, setResponseBySession] = useState<
    Record<string, ChatCompletionResp | undefined>
  >({})
  const [pendingBySession, setPendingBySession] = useState<Record<string, boolean>>({})
  const [pendingMessageBySession, setPendingMessageBySession] = useState<
    Record<string, ChatMessage | undefined>
  >({})
  const messagesViewportRef = useRef<HTMLDivElement>(null)
  const promptInputRef = useRef<HTMLTextAreaElement>(null)
  const {
    data: sessions = [],
    isLoading: isSessionsLoading,
    isSuccess: areSessionsLoaded,
  } = useQuery({
    queryKey: [
      'kthena',
      'inference-services',
      service.namespace,
      service.name,
      'conversations',
      user?.id,
      accountContext?.space,
    ],
    queryFn: () =>
      apiListKthenaConversations(service.name, {
        includeMessages: true,
        limit: 100,
        messageLimit: 500,
      }).then((res) => res.data),
    enabled: Boolean(user?.id),
  })

  useEffect(() => {
    if (activeSessionID === undefined) {
      setActiveSessionID(sessions[0]?.sessionId ?? null)
      return
    }
    if (activeSessionID && !sessions.some((session) => session.sessionId === activeSessionID)) {
      setActiveSessionID(sessions[0]?.sessionId ?? null)
    }
  }, [activeSessionID, sessions])

  useEffect(() => {
    if (!user?.id || !areSessionsLoaded || legacySessions.length === 0) return
    if (legacyMigrationAttemptedRef.current) return
    const migrationKey = getChatMigrationStorageKey(user.id, accountContext?.space, service)
    try {
      if (window.localStorage.getItem(migrationKey)) return
    } catch {
      return
    }
    legacyMigrationAttemptedRef.current = true
    void Promise.all(
      legacySessions.slice(0, 100).map((session) =>
        apiCreateKthenaConversation(service.name, {
          sessionId: session.id,
          title: session.title,
          messages: session.messages,
        })
      )
    )
      .then(async () => {
        try {
          window.localStorage.setItem(migrationKey, new Date().toISOString())
        } catch {
          // The server import is still complete when this browser cannot retain the marker.
        }
        await queryClient.invalidateQueries({ queryKey: conversationQueryKey })
      })
      .catch(() => {
        // Preserve source data and leave a retry possible after the next page load.
        legacyMigrationAttemptedRef.current = false
      })
  }, [
    accountContext?.space,
    areSessionsLoaded,
    conversationQueryKey,
    legacySessions,
    queryClient,
    service,
    user?.id,
  ])

  const activeSession = activeSessionID
    ? sessions.find((session) => session.sessionId === activeSessionID)
    : undefined
  const messages = activeSession?.messages ?? emptyChatMessages
  const activeSessionKey = activeSession?.sessionId ?? draftChatSessionKey
  const pendingMessage = pendingMessageBySession[activeSessionKey]
  const displayedMessages = useMemo(
    () => (pendingMessage ? [...messages, pendingMessage] : messages),
    [messages, pendingMessage]
  )
  const isActiveSessionPending = Boolean(pendingBySession[activeSessionKey])
  const rawResponse = responseBySession[activeSessionKey]
  const pendingMessages = useMemo(
    () =>
      prompt.trim()
        ? [...displayedMessages, { role: 'user', content: prompt.trim() } satisfies ChatMessage]
        : displayedMessages,
    [displayedMessages, prompt]
  )
  const curl = useMemo(
    () => buildCurl(service, pendingMessages, t('kthena.detail.defaultPrompt')),
    [pendingMessages, service, t]
  )

  useEffect(() => {
    const viewport = messagesViewportRef.current
    if (!viewport) return
    viewport.scrollTo({ top: viewport.scrollHeight, behavior: 'smooth' })
  }, [activeSession?.sessionId, displayedMessages.length, isActiveSessionPending])

  const focusPromptInput = () => {
    window.requestAnimationFrame(() => promptInputRef.current?.focus())
  }

  const upsertConversation = (conversation: KthenaConversation) => {
    queryClient.setQueryData<KthenaConversation[]>(conversationQueryKey, (current) => {
      const next = [
        conversation,
        ...(current ?? []).filter((item) => item.sessionId !== conversation.sessionId),
      ]
      return next.sort(
        (left, right) => new Date(right.updatedAt).getTime() - new Date(left.updatedAt).getTime()
      )
    })
  }

  const startNewSession = () => {
    setActiveSessionID(null)
    setPrompt('')
    focusPromptInput()
  }

  const selectSession = (sessionID: string) => {
    setActiveSessionID(sessionID)
    setPrompt('')
    focusPromptInput()
  }

  const { mutate: clearSession, isPending: isClearingSession } = useMutation({
    mutationFn: (sessionID: string) =>
      apiUpdateKthenaConversation(service.name, sessionID, { title: '', messages: [] }).then(
        (res) => res.data
      ),
    onSuccess: (conversation) => {
      upsertConversation(conversation)
      setResponseBySession((current) => {
        const next = { ...current }
        delete next[conversation.sessionId]
        return next
      })
      focusPromptInput()
    },
    onError: showErrorToast,
  })

  const clearActiveSession = () => {
    if (!activeSession) {
      setPrompt('')
      return
    }
    clearSession(activeSession.sessionId)
  }

  const [deletingSessionID, setDeletingSessionID] = useState<string | null>(null)
  const { mutate: removeSession } = useMutation({
    mutationFn: (sessionID: string) => apiDeleteKthenaConversation(service.name, sessionID),
    onMutate: (sessionID) => setDeletingSessionID(sessionID),
    onSuccess: (_response, sessionID) => {
      queryClient.setQueryData<KthenaConversation[]>(conversationQueryKey, (current) =>
        (current ?? []).filter((session) => session.sessionId !== sessionID)
      )
      if (activeSessionID === sessionID) {
        setActiveSessionID(undefined)
        setPrompt('')
      }
      setResponseBySession((current) => {
        const next = { ...current }
        delete next[sessionID]
        return next
      })
    },
    onError: showErrorToast,
    onSettled: () => setDeletingSessionID(null),
  })

  const { mutate: sendMessage } = useMutation({
    mutationFn: ({
      sessionID,
      content,
      clientTurnID,
    }: {
      sessionID?: string
      content: string
      clientTurnID: string
    }) =>
      apiCreateKthenaConversationTurn(service.name, {
        sessionId: sessionID,
        content,
        temperature: 0.2,
        clientTurnId: clientTurnID,
      }).then((res) => res.data),
    onMutate: ({ sessionID, content }) => {
      const sessionKey = sessionID ?? draftChatSessionKey
      setPendingBySession((current) => ({ ...current, [sessionKey]: true }))
      setPendingMessageBySession((current) => ({
        ...current,
        [sessionKey]: { role: 'user', content },
      }))
      return { sessionKey }
    },
    onSuccess: (response) => {
      upsertConversation(response.conversation)
      setActiveSessionID(response.conversation.sessionId)
      setResponseBySession((current) => ({
        ...current,
        [response.conversation.sessionId]: response.completion ?? undefined,
      }))
    },
    onError: (error, variables, context) => {
      if (context?.sessionKey === activeSessionKey) {
        setPrompt(variables.content)
      }
      showErrorToast(error)
    },
    onSettled: (_data, _error, _variables, context) => {
      const sessionKey = context?.sessionKey
      if (!sessionKey) return
      setPendingBySession((current) => {
        const next = { ...current }
        delete next[sessionKey]
        return next
      })
      setPendingMessageBySession((current) => {
        const next = { ...current }
        delete next[sessionKey]
        return next
      })
    },
  })

  const submitMessage = () => {
    const content = prompt.trim()
    if (!content || isActiveSessionPending || isSessionsLoading) return
    setPrompt('')
    setResponseBySession((current) => {
      const next = { ...current }
      delete next[activeSessionKey]
      return next
    })
    sendMessage({
      sessionID: activeSession?.sessionId,
      content,
      clientTurnID: createChatID(),
    })
  }

  return (
    <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_360px]">
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
                          onClick={() => removeSession(session.sessionId)}
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
                    onClick={() => removeSession(session.sessionId)}
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
                      message={{ role: 'assistant', content: t('kthena.actions.requesting') }}
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

function OverviewSection({
  icon: Icon,
  title,
  children,
  className,
}: {
  icon: typeof ActivityIcon
  title: string
  children: ReactNode
  className?: string
}) {
  return (
    <section className={className}>
      <div className="flex items-center gap-2 px-4 py-3">
        <Icon className="text-primary size-4" />
        <h3 className="text-sm font-semibold">{title}</h3>
      </div>
      <div className="px-4 pb-4">{children}</div>
    </section>
  )
}

function OverviewField({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="bg-muted/20 min-w-0 rounded-lg border px-3 py-2.5">
      <div className="text-muted-foreground text-xs font-medium">{label}</div>
      <div className="mt-1 min-w-0 text-sm">{children}</div>
    </div>
  )
}

function RuntimePanel({
  service,
  primaryPod,
  resources,
}: {
  service: KthenaService
  primaryPod?: KthenaRuntimePod
  resources: Record<string, string>
}) {
  const { t } = useTranslation()
  const pods = service.runtimePods ?? []
  return (
    <section className="bg-card overflow-hidden rounded-xl border">
      <div className="flex items-center justify-between gap-3 border-b px-4 py-3">
        <div className="flex min-w-0 items-center gap-2">
          <ActivityIcon className="text-primary size-4 shrink-0" />
          <h3 className="truncate text-sm font-semibold">{t('kthena.table.status')}</h3>
        </div>
        <KthenaStatusBadge service={service} />
      </div>
      <div className="grid gap-3 p-3">
        <OverviewField label={t('kthena.detail.runtimePod')}>
          <CopyValue value={primaryPod?.name || '-'} />
        </OverviewField>
        <div className="grid gap-2 sm:grid-cols-2 xl:grid-cols-1 2xl:grid-cols-2">
          <OverviewField label={t('kthena.detail.runtimeNode')}>
            <CopyValue value={primaryPod?.nodeName || '-'} />
          </OverviewField>
          <OverviewField label={t('kthena.detail.runtimeResources')}>
            {Object.keys(resources).length ? (
              <ResourceBadges resources={resources} />
            ) : (
              <span className="text-muted-foreground">-</span>
            )}
          </OverviewField>
        </div>
      </div>
      {pods.length ? (
        <div className="grid gap-1.5 border-t p-3">
          <div className="text-muted-foreground flex items-center justify-between gap-2 text-xs font-medium">
            <span>{t('kthena.detail.runtimePods')}</span>
            <Badge variant="secondary">{pods.length}</Badge>
          </div>
          {pods.map((pod) => (
            <div
              key={`${pod.namespace}/${pod.name}`}
              className="bg-muted/20 flex min-w-0 items-start justify-between gap-3 rounded-lg border px-3 py-2"
            >
              <div className="min-w-0 flex-1">
                <TooltipCopy
                  name={pod.name}
                  copyMessage={t('kthena.copy.generic', { label: 'Pod' })}
                  className="max-w-full min-w-0 truncate font-mono text-xs"
                />
                <div className="text-muted-foreground mt-1 flex flex-wrap gap-x-2 gap-y-0.5 text-xs">
                  <span>{pod.nodeName || '-'}</span>
                  {pod.podIP && <span>{pod.podIP}</span>}
                  <span>
                    {pod.readyContainers}/{pod.totalContainers}
                  </span>
                  {pod.restarts > 0 && (
                    <span>{t('kthena.detail.restarts', { count: pod.restarts })}</span>
                  )}
                </div>
              </div>
              <Badge className="shrink-0" variant={pod.ready ? 'default' : 'secondary'}>
                {pod.phase || '-'}
              </Badge>
            </div>
          ))}
        </div>
      ) : null}
    </section>
  )
}

function CopyValue({ value }: { value?: string }) {
  const { t } = useTranslation()
  const display = value || '-'
  return (
    <TooltipCopy
      name={display}
      copyMessage={t('kthena.copy.generic', { label: display })}
      showIcon={display !== '-'}
      className="max-w-full text-left font-mono break-all"
    />
  )
}

function InvocationValue({ label, value }: { label: string; value?: string }) {
  const { t } = useTranslation()
  const display = value || '-'
  return (
    <div className="grid min-w-0 gap-1.5">
      <div className="text-muted-foreground text-xs font-medium">{label}</div>
      <div className="bg-muted/35 flex min-w-0 items-center gap-1 rounded-lg border px-2.5 py-1.5">
        <span className="min-w-0 flex-1 truncate font-mono text-xs" title={display}>
          {display}
        </span>
        {display !== '-' && (
          <CopyButton
            content={display}
            copyMessage={t('kthena.copy.generic', { label })}
            className="shrink-0"
          />
        )}
      </div>
    </div>
  )
}

function ChatBubble({ message, muted = false }: { message: ChatMessage; muted?: boolean }) {
  const isUser = message.role === 'user'
  const content = isUser ? message.content : stripThinkBlocks(message.content)
  return (
    <div
      className={[
        'flex items-end gap-2',
        isUser ? 'flex-row-reverse justify-start' : 'justify-start',
      ].join(' ')}
    >
      <div
        className={[
          'flex size-7 shrink-0 items-center justify-center rounded-full border',
          isUser ? 'bg-primary text-primary-foreground' : 'bg-background text-muted-foreground',
        ].join(' ')}
      >
        {isUser ? <UserRoundIcon className="size-3.5" /> : <BotIcon className="size-3.5" />}
      </div>
      <div
        className={[
          'max-w-[min(82%,48rem)] rounded-2xl px-4 py-3 text-sm leading-6 whitespace-pre-wrap shadow-sm',
          isUser
            ? 'bg-primary text-primary-foreground rounded-br-md'
            : 'bg-background rounded-bl-md border',
          muted ? 'text-muted-foreground' : '',
        ].join(' ')}
      >
        {content}
      </div>
    </div>
  )
}

function ResourceCard({ resource }: { resource: KthenaResource }) {
  const { t } = useTranslation()
  return (
    <div className="rounded-md border p-3">
      <div className="mb-2 flex items-center justify-between gap-2">
        <div className="font-medium">{resource.kind}</div>
        <Badge variant={getResourcePhaseVariant(resource.phase)}>
          {resource.phase || 'Pending'}
        </Badge>
      </div>
      <TooltipCopy
        name={`${resource.namespace}/${resource.name}`}
        copyMessage={t('kthena.copy.resource')}
        className="max-w-full truncate font-mono text-xs"
      />
      {resource.conditions?.length > 0 && (
        <div className="text-muted-foreground mt-2 line-clamp-2 text-xs">
          {resource.conditions
            .map((condition) => `${String(condition.type ?? '')}:${String(condition.status ?? '')}`)
            .join(' · ')}
        </div>
      )}
    </div>
  )
}

function DiagnosticCard({ diagnostic }: { diagnostic: KthenaDiagnostic }) {
  const { t } = useTranslation()
  const isError = diagnostic.level === 'error'
  return (
    <div
      className={[
        'relative rounded-md border p-3',
        isError ? 'border-destructive/40 bg-destructive/5' : '',
      ].join(' ')}
    >
      <div
        className={[
          'border-background absolute top-4 -left-[17px] size-2.5 rounded-full border-2',
          isError ? 'bg-destructive' : 'bg-amber-500',
        ].join(' ')}
      />
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge variant={isError ? 'destructive' : 'secondary'}>
          {diagnostic.reason || diagnostic.level}
        </Badge>
        {diagnostic.pod && <span className="text-muted-foreground text-xs">{diagnostic.pod}</span>}
        {diagnostic.container && (
          <span className="text-muted-foreground text-xs">/{diagnostic.container}</span>
        )}
      </div>
      <div className="text-sm leading-6 break-words">{diagnostic.message}</div>
      {diagnostic.details && (
        <details className="mt-3">
          <summary className="text-muted-foreground cursor-pointer text-xs">
            {t('kthena.detail.logDetails')}
          </summary>
          <pre className="bg-muted mt-2 max-h-72 overflow-auto rounded-md p-3 text-xs leading-5 break-words whitespace-pre-wrap">
            {diagnostic.details}
          </pre>
        </details>
      )}
      {(diagnostic.resource || diagnostic.timestamp) && (
        <div className="text-muted-foreground mt-2 flex flex-wrap gap-3 text-xs">
          {diagnostic.resource && <span>{diagnostic.resource}</span>}
          {diagnostic.timestamp && <span>{new Date(diagnostic.timestamp).toLocaleString()}</span>}
        </div>
      )}
    </div>
  )
}

function DiagnosticSummary({ diagnostics }: { diagnostics: KthenaDiagnostic[] }) {
  const { t } = useTranslation()
  const firstError =
    diagnostics.find((diagnostic) => diagnostic.level === 'error') ?? diagnostics[0]
  return (
    <div className="rounded-md border p-3">
      <div className="mb-2 flex flex-wrap items-center gap-2">
        <Badge variant={firstError.level === 'error' ? 'destructive' : 'secondary'}>
          {firstError.reason || firstError.level}
        </Badge>
        <span className="text-muted-foreground text-xs">{t('kthena.detail.primaryIssue')}</span>
      </div>
      <div className="text-sm leading-6 break-words">{firstError.message}</div>
    </div>
  )
}

function getWorkerResources(service: KthenaService) {
  const resources: Record<string, string> = {}
  if (service.workerCPU) resources.cpu = service.workerCPU
  if (service.workerMemory) resources.memory = service.workerMemory
  if (service.workerGPU && service.workerGPU !== '0') {
    resources[service.workerGPUModel || 'gpu'] = service.workerGPU
  }
  return resources
}

function buildCurl(service: KthenaService, messages: ChatMessage[], placeholder: string) {
  const model = service.access?.modelName || service.name
  const baseURL = displayAPIBaseURL(service)
  const requestBody = JSON.stringify({
    model,
    messages: messages.length ? messages : [{ role: 'user', content: placeholder }],
    temperature: 0.2,
  })
  return `curl -X POST '${baseURL}/chat/completions' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <your-crater-token>' \\
  -d '${requestBody}'`
}

function displayAPIBaseURL(service: KthenaService) {
  return `${window.location.origin}/api${service.access?.proxyBaseURL || `/v1/kthena/inference-services/${service.name}/openai/v1`}`
}

function stripThinkBlocks(content: string) {
  return content
    .replace(/<think>[\s\S]*?<\/think>/gi, '')
    .replace(/^\s+/, '')
    .trimEnd()
}

function createChatID() {
  return typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function'
    ? crypto.randomUUID()
    : `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function createChatSession(): ChatSession {
  const now = new Date().toISOString()
  return {
    id: createChatID(),
    title: '',
    messages: [],
    createdAt: now,
    updatedAt: now,
  }
}

function chatSessionTitle(messages: ChatMessage[]) {
  const firstUserMessage = messages.find((message) => message.role === 'user')?.content
  if (!firstUserMessage) return ''
  const normalized = firstUserMessage.replace(/\s+/g, ' ').trim()
  return normalized.length > 28 ? `${normalized.slice(0, 28)}…` : normalized
}

function readStoredChatSessions(service: Pick<KthenaService, 'name' | 'namespace'>): ChatSession[] {
  for (const storageKey of [
    getChatSessionStorageKey(service),
    getLegacyChatSessionStorageKey(service.name),
  ]) {
    try {
      const raw = window.localStorage.getItem(storageKey)
      if (!raw) continue
      const parsed = JSON.parse(raw) as { sessions?: unknown }
      const sessions = Array.isArray(parsed.sessions) ? parsed.sessions.filter(isChatSession) : []
      if (sessions.length > 0) {
        return sessions
      }
    } catch {
      // Try the next compatible storage format.
    }
  }

  const legacyMessages = readStoredMessages(service.name)
  const migratedSession = createChatSession()
  if (legacyMessages.length > 0) {
    migratedSession.messages = legacyMessages
    migratedSession.title = chatSessionTitle(legacyMessages)
    return [migratedSession]
  }
  return []
}

function readStoredMessages(serviceName: string): ChatMessage[] {
  try {
    const raw = window.localStorage.getItem(getLegacyChatStorageKey(serviceName))
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed.filter(isChatMessage)
  } catch {
    return []
  }
}

function isChatSession(value: unknown): value is ChatSession {
  if (!value || typeof value !== 'object') return false
  const session = value as Partial<ChatSession>
  return (
    typeof session.id === 'string' &&
    typeof session.title === 'string' &&
    typeof session.createdAt === 'string' &&
    typeof session.updatedAt === 'string' &&
    Array.isArray(session.messages) &&
    session.messages.every(isChatMessage)
  )
}

function isChatMessage(value: unknown): value is ChatMessage {
  if (!value || typeof value !== 'object') return false
  const message = value as Partial<ChatMessage>
  return (
    (message.role === 'system' || message.role === 'user' || message.role === 'assistant') &&
    typeof message.content === 'string'
  )
}
