import { useQuery } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import type { UIMessage } from 'ai'
import { format } from 'date-fns'
import { BotIcon, ChevronLeftIcon, ChevronRightIcon, EyeIcon } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useDebounceValue } from 'usehooks-ts'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

import { MarkdownRenderer } from '@/components/form/markdown-renderer'
import PageTitle from '@/components/layout/page-title'

import {
  type AdminAgentConversation,
  type AgentIdentity,
  apiAdminGetAgentConversationMessages,
  apiAdminGetAgentConversations,
} from '@/services/api/agent'

import { cn } from '@/lib/utils'

export const Route = createFileRoute('/admin/agent/')({
  component: AgentOperationsPage,
})

function getMessageText(message: UIMessage) {
  return message.parts
    .filter((part) => part.type === 'text')
    .map((part) => part.text)
    .join('')
}

function ConversationMessages({ conversation }: { conversation: AdminAgentConversation }) {
  const { t } = useTranslation()
  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'agent', 'conversations', conversation.id, 'messages'],
    queryFn: () => apiAdminGetAgentConversationMessages(conversation.id).then((res) => res.data),
  })

  if (isLoading) {
    return (
      <p className="text-muted-foreground py-8 text-center text-sm">
        {t('agentAudit.loadingMessages')}
      </p>
    )
  }

  return (
    <ScrollArea className="h-[min(68vh,720px)] pr-4">
      <div className="space-y-4">
        {(data ?? []).map((message) => {
          const text = getMessageText(message)
          const fromUser = message.role === 'user'
          return (
            <div
              key={message.id}
              className={cn('flex', fromUser ? 'justify-end' : 'justify-start')}
            >
              <div
                className={cn(
                  'max-w-[85%] rounded-xl px-4 py-3 text-sm leading-6',
                  fromUser ? 'bg-primary text-primary-foreground' : 'bg-muted'
                )}
              >
                {text ? (
                  fromUser ? (
                    <p className="whitespace-pre-wrap">{text}</p>
                  ) : (
                    <MarkdownRenderer>{text}</MarkdownRenderer>
                  )
                ) : (
                  <p className="text-muted-foreground">{t('agentAudit.structuredMessage')}</p>
                )}
              </div>
            </div>
          )
        })}
        {!data?.length && (
          <p className="text-muted-foreground py-8 text-center text-sm">
            {t('agentAudit.noMessages')}
          </p>
        )}
      </div>
    </ScrollArea>
  )
}

function AgentOperationsPage() {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const [debouncedSearch] = useDebounceValue(search.trim(), 300)
  const [identity, setIdentity] = useState<'all' | AgentIdentity>('all')
  const [page, setPage] = useState(1)
  const [selectedConversation, setSelectedConversation] = useState<AdminAgentConversation>()
  const pageSize = 20

  const { data, isLoading } = useQuery({
    queryKey: ['admin', 'agent', 'conversations', page, pageSize, debouncedSearch, identity],
    queryFn: () =>
      apiAdminGetAgentConversations({
        page,
        pageSize,
        search: debouncedSearch || undefined,
        identityScope: identity === 'all' ? undefined : identity,
      }).then((res) => res.data),
  })
  const conversations = data?.items ?? []
  const totalPages = Math.max(1, Math.ceil((data?.total ?? 0) / pageSize))

  return (
    <div className="space-y-4">
      <PageTitle title={t('agentAudit.title')} description={t('agentAudit.description')} />

      <Card className="gap-0 py-0">
        <CardContent className="space-y-4 p-4">
          <div className="flex flex-col gap-3 sm:flex-row">
            <Input
              value={search}
              className="max-w-md"
              placeholder={t('agentAudit.searchPlaceholder')}
              onChange={(event) => {
                setSearch(event.target.value)
                setPage(1)
              }}
            />
            <Select
              value={identity}
              onValueChange={(value) => {
                setIdentity(value as typeof identity)
                setPage(1)
              }}
            >
              <SelectTrigger className="w-full sm:w-40" aria-label={t('agentAudit.identityFilter')}>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">{t('agentAudit.identityAll')}</SelectItem>
                <SelectItem value="user">{t('agentAudit.identityUser')}</SelectItem>
                <SelectItem value="admin">{t('agentAudit.identityAdmin')}</SelectItem>
              </SelectContent>
            </Select>
          </div>
          <div className="overflow-hidden rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t('agentAudit.user')}</TableHead>
                  <TableHead>{t('agentAudit.identity')}</TableHead>
                  <TableHead>{t('agentAudit.conversationTitle')}</TableHead>
                  <TableHead>{t('agentAudit.messageCount')}</TableHead>
                  <TableHead>{t('agentAudit.updatedAt')}</TableHead>
                  <TableHead className="w-20 text-right">{t('agentAudit.actions')}</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {conversations.map((conversation) => (
                  <TableRow key={conversation.id}>
                    <TableCell>
                      <div className="font-medium">
                        {conversation.nickname || conversation.username}
                      </div>
                      <div className="text-muted-foreground text-xs">@{conversation.username}</div>
                    </TableCell>
                    <TableCell>
                      <Badge
                        variant={conversation.identityScope === 'admin' ? 'default' : 'secondary'}
                      >
                        {conversation.identityScope === 'admin'
                          ? t('agentAudit.identityAdmin')
                          : t('agentAudit.identityUser')}
                      </Badge>
                    </TableCell>
                    <TableCell
                      className="max-w-80 truncate"
                      title={conversation.title || t('craterAgent.newConversation')}
                    >
                      {conversation.title || t('craterAgent.newConversation')}
                    </TableCell>
                    <TableCell>{conversation.messageCount}</TableCell>
                    <TableCell>{format(conversation.updatedAt, 'yyyy-MM-dd HH:mm:ss')}</TableCell>
                    <TableCell className="text-right">
                      <Button
                        size="icon"
                        variant="ghost"
                        aria-label={t('agentAudit.viewConversation')}
                        onClick={() => setSelectedConversation(conversation)}
                      >
                        <EyeIcon />
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
                {!isLoading && conversations.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                      {t('agentAudit.empty')}
                    </TableCell>
                  </TableRow>
                )}
                {isLoading && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-muted-foreground h-24 text-center">
                      {t('agentAudit.loading')}
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          </div>
          <div className="flex items-center justify-end gap-3">
            <span className="text-muted-foreground text-sm">
              {t('agentAudit.pagination', { page, totalPages, total: data?.total ?? 0 })}
            </span>
            <Button
              size="icon"
              variant="outline"
              aria-label={t('agentAudit.previousPage')}
              disabled={page <= 1 || isLoading}
              onClick={() => setPage((current) => Math.max(1, current - 1))}
            >
              <ChevronLeftIcon />
            </Button>
            <Button
              size="icon"
              variant="outline"
              aria-label={t('agentAudit.nextPage')}
              disabled={page >= totalPages || isLoading}
              onClick={() => setPage((current) => Math.min(totalPages, current + 1))}
            >
              <ChevronRightIcon />
            </Button>
          </div>
        </CardContent>
      </Card>

      <Dialog
        open={selectedConversation !== undefined}
        onOpenChange={(open) => {
          if (!open) setSelectedConversation(undefined)
        }}
      >
        <DialogContent className="sm:max-w-4xl">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 pr-8">
              <BotIcon className="text-primary size-5" />
              <span className="truncate">
                {selectedConversation?.title || t('craterAgent.newConversation')}
              </span>
            </DialogTitle>
            <DialogDescription>
              {selectedConversation &&
                t('agentAudit.dialogDescription', {
                  nickname: selectedConversation.nickname || selectedConversation.username,
                  username: selectedConversation.username,
                  identity:
                    selectedConversation.identityScope === 'admin'
                      ? t('agentAudit.identityAdmin')
                      : t('agentAudit.identityUser'),
                })}
            </DialogDescription>
          </DialogHeader>
          {selectedConversation && <ConversationMessages conversation={selectedConversation} />}
        </DialogContent>
      </Dialog>
    </div>
  )
}
