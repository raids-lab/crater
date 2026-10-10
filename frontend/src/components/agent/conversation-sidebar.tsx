import {
  CheckIcon,
  HistoryIcon,
  MoreHorizontalIcon,
  PencilIcon,
  PlusIcon,
  Trash2Icon,
  XIcon,
} from 'lucide-react'
import { type FormEvent, useState } from 'react'
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
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { Input } from '@/components/ui/input'

import type { AgentConversation } from '@/services/api/agent'

import { cn } from '@/lib/utils'

type ConversationSidebarProps = {
  conversations: AgentConversation[]
  activeConversationID: string
  busy: boolean
  onCreate: () => void
  onSelect: (conversation: AgentConversation) => void
  onRename: (conversationID: string, title: string) => void
  onDelete: (conversationID: string) => void
}

export function ConversationSidebar({
  conversations,
  activeConversationID,
  busy,
  onCreate,
  onSelect,
  onRename,
  onDelete,
}: ConversationSidebarProps) {
  const { t } = useTranslation()
  const [editingID, setEditingID] = useState<string>()
  const [editingTitle, setEditingTitle] = useState('')
  const [deleteTarget, setDeleteTarget] = useState<AgentConversation>()
  const [hoveredID, setHoveredID] = useState<string>()
  const [menuOpenID, setMenuOpenID] = useState<string>()

  const beginRename = (conversation: AgentConversation) => {
    setEditingID(conversation.id)
    setEditingTitle(conversation.title || t('craterAgent.newConversation'))
  }

  const cancelRename = () => {
    setEditingID(undefined)
    setEditingTitle('')
  }

  const submitRename = (event: FormEvent) => {
    event.preventDefault()
    const title = editingTitle.trim()
    if (!editingID || !title) return
    onRename(editingID, title)
    cancelRename()
  }

  return (
    <aside className="bg-muted/25 flex h-full w-52 shrink-0 flex-col border-r">
      <div className="flex h-12 items-center justify-between gap-2 border-b px-3">
        <div className="flex min-w-0 items-center gap-2 text-sm font-semibold">
          <HistoryIcon className="text-muted-foreground size-4 shrink-0" />
          <span className="truncate">{t('craterAgent.history')}</span>
        </div>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 shrink-0"
          disabled={busy}
          aria-label={t('craterAgent.newConversation')}
          onClick={onCreate}
        >
          <PlusIcon className="size-4" />
        </Button>
      </div>
      <div className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
        <div className="w-full min-w-0 space-y-1 p-2">
          {!conversations.some((conversation) => conversation.id === activeConversationID) && (
            <div className="bg-accent text-accent-foreground truncate rounded-lg px-2 py-2.5 text-xs">
              {t('craterAgent.newConversation')}
            </div>
          )}
          {conversations.length === 0 && (
            <p className="text-muted-foreground px-2 py-4 text-xs leading-5">
              {t('craterAgent.emptyHistory')}
            </p>
          )}
          {conversations.map((conversation) =>
            editingID === conversation.id ? (
              <form
                key={conversation.id}
                className="bg-background flex items-center gap-1 rounded-lg border p-1"
                onSubmit={submitRename}
              >
                <Input
                  autoFocus
                  value={editingTitle}
                  className="h-8 min-w-0 flex-1 text-xs"
                  maxLength={80}
                  aria-label={t('craterAgent.renameConversation')}
                  onChange={(event) => setEditingTitle(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === 'Escape') cancelRename()
                  }}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-7 shrink-0"
                  aria-label={t('craterAgent.cancelEdit')}
                  title={t('craterAgent.cancelEdit')}
                  onClick={cancelRename}
                >
                  <XIcon className="size-3.5" />
                </Button>
                <Button
                  type="submit"
                  variant="ghost"
                  size="icon"
                  className="size-7 shrink-0"
                  disabled={!editingTitle.trim()}
                  aria-label={t('craterAgent.saveTitle')}
                  title={t('craterAgent.saveTitle')}
                >
                  <CheckIcon className="size-3.5" />
                </Button>
              </form>
            ) : (
              <div
                key={conversation.id}
                className={cn(
                  'relative flex items-center rounded-lg transition-colors',
                  conversation.id === activeConversationID
                    ? 'bg-accent text-accent-foreground'
                    : 'hover:bg-accent/60'
                )}
                onMouseEnter={() => setHoveredID(conversation.id)}
                onMouseLeave={() => {
                  if (menuOpenID !== conversation.id) setHoveredID(undefined)
                }}
              >
                <button
                  type="button"
                  className="min-w-0 flex-1 truncate py-2.5 pr-10 pl-2 text-left text-xs"
                  disabled={busy}
                  title={conversation.title || t('craterAgent.newConversation')}
                  onClick={() => onSelect(conversation)}
                >
                  {conversation.title || t('craterAgent.newConversation')}
                </button>
                <DropdownMenu
                  open={menuOpenID === conversation.id}
                  onOpenChange={(menuOpen) => {
                    setMenuOpenID(menuOpen ? conversation.id : undefined)
                    if (!menuOpen) setHoveredID(undefined)
                  }}
                >
                  <DropdownMenuTrigger asChild>
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon"
                      className="bg-accent absolute right-1 z-20 size-7 shrink-0 transition-opacity"
                      style={{
                        opacity:
                          hoveredID === conversation.id || menuOpenID === conversation.id ? 1 : 0,
                        pointerEvents:
                          hoveredID === conversation.id || menuOpenID === conversation.id
                            ? 'auto'
                            : 'none',
                      }}
                      disabled={busy}
                      aria-label={t('craterAgent.conversationActions')}
                      title={t('craterAgent.conversationActions')}
                    >
                      <MoreHorizontalIcon className="size-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent align="end" side="right">
                    <DropdownMenuItem onSelect={() => beginRename(conversation)}>
                      <PencilIcon />
                      {t('craterAgent.renameConversation')}
                    </DropdownMenuItem>
                    <DropdownMenuItem
                      variant="destructive"
                      onSelect={() => setDeleteTarget(conversation)}
                    >
                      <Trash2Icon />
                      {t('craterAgent.deleteConversation')}
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              </div>
            )
          )}
        </div>
      </div>
      <AlertDialog
        open={deleteTarget !== undefined}
        onOpenChange={(dialogOpen) => {
          if (!dialogOpen) setDeleteTarget(undefined)
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('craterAgent.deleteConversationTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('craterAgent.deleteConversationDescription', {
                title: deleteTarget?.title || t('craterAgent.newConversation'),
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('craterAgent.cancelEdit')}</AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button
                variant="destructive"
                onClick={() => {
                  if (deleteTarget) onDelete(deleteTarget.id)
                  setDeleteTarget(undefined)
                }}
              >
                {t('craterAgent.deleteConversation')}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </aside>
  )
}
