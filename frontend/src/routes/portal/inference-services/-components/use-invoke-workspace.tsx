import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import {
  ChatCompletionResp,
  KthenaConversation,
  KthenaService,
  apiDeleteKthenaConversation,
  apiListKthenaConversations,
  apiStreamKthenaTurn,
  apiUpdateKthenaConversation,
} from '@/services/api/inference'

import { atomUserContext, atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import {
  ChatMessage,
  buildCurl,
  createChatID,
  draftChatSessionKey,
  emptyChatMessages,
} from './detail-parts'

export function useInvokeWorkspace(service: KthenaService) {
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
  const controllers = useRef<Record<string, AbortController>>({})
  const [streamedBySession, setStreamedBySession] = useState<Record<string, string>>({})
  useEffect(() => {
    const active = controllers.current
    return () => {
      Object.values(active).forEach((controller) => controller.abort())
    }
  }, [])
  const messagesViewportRef = useRef<HTMLDivElement>(null)
  const promptInputRef = useRef<HTMLTextAreaElement>(null)
  const { data: sessions = [], isLoading: isSessionsLoading } = useQuery({
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
  }, [
    activeSession?.sessionId,
    displayedMessages.length,
    isActiveSessionPending,
    streamedBySession,
    activeSessionKey,
  ])
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
    if (pendingBySession[draftChatSessionKey]) return
    delete retryTurn.current[draftChatSessionKey]
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
    setSessionAction({ kind: 'clear', id: activeSession.sessionId })
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
    mutationFn: async ({
      sessionID,
      content,
      clientTurnID,
      sessionKey,
    }: {
      sessionID: string
      content: string
      clientTurnID: string
      sessionKey: string
    }) => {
      const controller = new AbortController()
      controllers.current[sessionKey] = controller
      return apiStreamKthenaTurn(
        service.name,
        { sessionId: sessionID, content, temperature: 0.2, clientTurnId: clientTurnID },
        controller.signal,
        (text) =>
          setStreamedBySession((current) => ({
            ...current,
            [sessionKey]: (current[sessionKey] || '') + text,
          }))
      )
    },
    onMutate: ({ sessionKey, content }) => {
      setPendingBySession((current) => ({ ...current, [sessionKey]: true }))
      setStreamedBySession((current) => ({ ...current, [sessionKey]: '' }))
      setPendingMessageBySession((current) => ({
        ...current,
        [sessionKey]: { role: 'user', content },
      }))
      return { sessionKey }
    },
    onSuccess: (response, variables) => {
      delete retryTurn.current[variables.sessionKey]
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
      if (!(error instanceof Error && error.name === 'AbortError')) showErrorToast(error)
    },
    onSettled: (_data, _error, _variables, context) => {
      const sessionKey = context?.sessionKey
      if (!sessionKey) return
      delete controllers.current[sessionKey]
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
  const retryTurn = useRef<
    Record<string, { sessionID: string; content: string; clientTurnID: string }>
  >({})
  const [sessionAction, setSessionAction] = useState<{
    kind: 'clear' | 'delete'
    id: string
  } | null>(null)
  const submitMessage = () => {
    const content = prompt.trim()
    if (!content || isActiveSessionPending || isSessionsLoading) return
    setPrompt('')
    setResponseBySession((current) => {
      const next = { ...current }
      delete next[activeSessionKey]
      return next
    })
    const previous = retryTurn.current[activeSessionKey]
    const turn =
      previous &&
      previous.content === content &&
      (!activeSession || previous.sessionID === activeSession.sessionId)
        ? previous
        : {
            sessionID: activeSession?.sessionId ?? createChatID(),
            content,
            clientTurnID: createChatID(),
          }
    retryTurn.current[activeSessionKey] = turn
    sendMessage({ ...turn, sessionKey: activeSessionKey })
  }
  return {
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
  }
}
