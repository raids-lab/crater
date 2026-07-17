import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Loader2Icon, ShieldBanIcon, ShieldCheckIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import { apiAdminUpdateUserBanStatus } from '@/services/api/admin/user'

import { showErrorToast } from '@/utils/toast'

import { AdminUserRow } from './types'

interface UserBanDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: AdminUserRow | null
}

export function UserBanDialog({ open, onOpenChange, user }: UserBanDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [reason, setReason] = useState('')
  const isBanned = Boolean(user?.bannedAt)

  useEffect(() => {
    if (open) setReason('')
  }, [open])

  const mutation = useMutation({
    mutationFn: () => {
      if (!user) throw new Error('No user selected')
      return apiAdminUpdateUserBanStatus(user.name, {
        banned: !isBanned,
        reason: reason.trim(),
      })
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] }),
        queryClient.invalidateQueries({ queryKey: ['admin', 'users', user?.name, 'ban'] }),
      ])
      toast.success(t(isBanned ? 'userBan.toast.unbanSuccess' : 'userBan.toast.banSuccess'))
      onOpenChange(false)
    },
    onError: showErrorToast,
  })

  const canSubmit = Boolean(user && reason.trim() && !mutation.isPending)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="w-[calc(100vw-2rem)] sm:max-w-[460px]">
        <DialogHeader>
          <DialogTitle>
            {t(isBanned ? 'userBan.dialog.unbanTitle' : 'userBan.dialog.banTitle')}
          </DialogTitle>
          <DialogDescription>
            {t(isBanned ? 'userBan.dialog.unbanDescription' : 'userBan.dialog.banDescription', {
              name: user?.name,
            })}
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2">
          <Label htmlFor="user-ban-reason">{t('userBan.dialog.reason')}</Label>
          <Textarea
            id="user-ban-reason"
            value={reason}
            maxLength={500}
            rows={4}
            placeholder={t('userBan.dialog.reasonPlaceholder')}
            onChange={(event) => setReason(event.target.value)}
          />
          <p className="text-muted-foreground text-xs">{reason.length}/500</p>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => onOpenChange(false)}
            disabled={mutation.isPending}
          >
            {t('common.cancel')}
          </Button>
          <Button
            variant={isBanned ? 'default' : 'destructive'}
            disabled={!canSubmit}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending ? (
              <Loader2Icon className="mr-2 size-4 animate-spin" />
            ) : isBanned ? (
              <ShieldCheckIcon className="mr-2 size-4" />
            ) : (
              <ShieldBanIcon className="mr-2 size-4" />
            )}
            {mutation.isPending
              ? t('common.saving')
              : t(isBanned ? 'userBan.actions.unban' : 'userBan.actions.ban')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
