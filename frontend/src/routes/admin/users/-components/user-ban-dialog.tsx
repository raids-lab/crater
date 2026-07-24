import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2Icon } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'

import { DurationFields } from '@/components/form/duration-fields'

import {
  IUserBanRestrictions,
  apiAdminGetUserBanStatus,
  apiAdminUpdateUserBanStatus,
} from '@/services/api/admin/user'

import { showErrorToast } from '@/utils/toast'

import { AdminUserRow } from './types'

interface UserBanDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  user: AdminUserRow | null
  mode: UserBanDialogMode
}

export type UserBanDialogMode = 'ban' | 'extend' | 'unban'

const EMPTY_RESTRICTIONS: IUserBanRestrictions = {
  platformAccess: false,
  jobSubmission: false,
  imageBuild: false,
  modelDownload: false,
  datasetDownload: false,
}

const restrictionFields: Array<{
  key: keyof IUserBanRestrictions
  labelKey: string
}> = [
  {
    key: 'platformAccess',
    labelKey: 'userBan.restrictions.platformAccess',
  },
  {
    key: 'jobSubmission',
    labelKey: 'userBan.restrictions.jobSubmission',
  },
  {
    key: 'imageBuild',
    labelKey: 'userBan.restrictions.imageBuild',
  },
  {
    key: 'modelDownload',
    labelKey: 'userBan.restrictions.modelDownload',
  },
  {
    key: 'datasetDownload',
    labelKey: 'userBan.restrictions.datasetDownload',
  },
]

export function UserBanDialog({ open, onOpenChange, user, mode }: UserBanDialogProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [reason, setReason] = useState('')
  const [isPermanent, setIsPermanent] = useState(false)
  const [duration, setDuration] = useState({ days: 0, hours: 0 })
  const [restrictions, setRestrictions] = useState<IUserBanRestrictions>(EMPTY_RESTRICTIONS)
  const restrictionsEditedRef = useRef(false)
  const isUnban = mode === 'unban'
  const currentBanQuery = useQuery({
    queryKey: ['admin', 'users', user?.name, 'ban'],
    queryFn: () => apiAdminGetUserBanStatus(user?.name ?? '').then((res) => res.data),
    enabled: Boolean(open && mode === 'extend' && user?.name),
  })

  useEffect(() => {
    if (open) {
      setReason('')
      setIsPermanent(false)
      setDuration({ days: 0, hours: 0 })
      restrictionsEditedRef.current = false
      setRestrictions(
        mode === 'extend' && user ? { ...user.banRestrictions } : { ...EMPTY_RESTRICTIONS }
      )
    }
  }, [mode, open, user])

  useEffect(() => {
    if (open && mode === 'extend' && currentBanQuery.data && !restrictionsEditedRef.current) {
      setRestrictions({ ...currentBanQuery.data.banRestrictions })
    }
  }, [currentBanQuery.data, mode, open])

  const mutation = useMutation({
    mutationFn: () => {
      if (!user) throw new Error('No user selected')
      return apiAdminUpdateUserBanStatus(user.name, {
        banned: !isUnban,
        isPermanent: !isUnban && isPermanent,
        days: isUnban || isPermanent ? 0 : duration.days,
        hours: isUnban || isPermanent ? 0 : duration.hours,
        minutes: 0,
        banRestrictions: isUnban ? EMPTY_RESTRICTIONS : restrictions,
        reason: reason.trim(),
      })
    },
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: ['admin', 'userlist'] }),
        queryClient.invalidateQueries({ queryKey: ['admin', 'users', user?.name, 'ban'] }),
        queryClient.invalidateQueries({ queryKey: ['users', user?.name, 'ban'] }),
        queryClient.invalidateQueries({ queryKey: ['current-user', 'ban'] }),
        queryClient.invalidateQueries({ queryKey: ['user', user?.name] }),
      ])
      toast.success(t(`userBan.toast.${mode}Success`))
      onOpenChange(false)
    },
    onError: showErrorToast,
  })

  const hasDuration = duration.days > 0 || duration.hours > 0
  const hasRestriction = Object.values(restrictions).some(Boolean)
  const nickname = user?.attributes.nickname?.trim()
  const displayName = nickname || user?.name
  const canSubmit = Boolean(
    user &&
      (isUnban || (reason.trim() && (isPermanent || hasDuration) && hasRestriction)) &&
      !mutation.isPending
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-h-[calc(100vh-2rem)] w-[calc(100vw-2rem)] overflow-y-auto sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle>{t(`userBan.dialog.${mode}Title`)}</DialogTitle>
          <DialogDescription>
            {user && (
              <>
                <span className="text-foreground font-medium">{displayName}</span>
                {displayName !== user.name && <span> (@{user.name})</span>}
                {' — '}
                {t(`userBan.dialog.${mode}Description`, { name: displayName })}
              </>
            )}
          </DialogDescription>
        </DialogHeader>
        {!isUnban && (
          <>
            <div className="space-y-2">
              <Label>{t('userBan.restrictions.title')}</Label>
              {/* 增大内边距，让 checkbox 区域更宽松 */}
              <div className="grid grid-cols-1 gap-x-4 gap-y-2.5 rounded-lg border p-4 min-[420px]:grid-cols-2">
                {restrictionFields.map((field) => {
                  const id = `user-ban-restriction-${field.key}`
                  return (
                    <label
                      key={field.key}
                      htmlFor={id}
                      className="flex min-h-7 cursor-pointer items-center gap-2 text-sm"
                    >
                      <Checkbox
                        id={id}
                        checked={restrictions[field.key]}
                        onCheckedChange={(checked) => {
                          restrictionsEditedRef.current = true
                          setRestrictions((current) => ({
                            ...current,
                            [field.key]: Boolean(checked),
                          }))
                        }}
                      />
                      <span>{t(field.labelKey)}</span>
                    </label>
                  )
                })}
              </div>
            </div>
            <Separator />
            <div className="space-y-2">
              <div className="flex items-center justify-between gap-3">
                <Label>{t('userBan.dialog.duration')}</Label>
                <div className="flex items-center gap-2">
                  <Label
                    htmlFor="user-ban-permanent"
                    className="text-muted-foreground cursor-pointer font-normal"
                  >
                    {t('userBan.dialog.permanent')}
                  </Label>
                  <Switch
                    id="user-ban-permanent"
                    checked={isPermanent}
                    onCheckedChange={setIsPermanent}
                  />
                </div>
              </div>
              {!isPermanent && (
                <DurationFields
                  value={duration}
                  onChange={(value) => setDuration({ days: value.days, hours: value.hours })}
                  origin={
                    mode === 'extend' ? (currentBanQuery.data?.bannedTimestamp ?? null) : null
                  }
                  showPreview={true}
                />
              )}
            </div>
            <Separator />
          </>
        )}
        <div className="space-y-2">
          <div className="flex items-center justify-between gap-3">
            <Label htmlFor="user-ban-reason">
              {t(isUnban ? 'userBan.dialog.reasonOptional' : 'userBan.dialog.reason')}
            </Label>
            <span className="text-muted-foreground text-xs">{reason.length}/500</span>
          </div>
          <Textarea
            id="user-ban-reason"
            value={reason}
            maxLength={500}
            rows={3}
            className="resize-none"
            placeholder={t('userBan.dialog.reasonPlaceholder')}
            onChange={(event) => setReason(event.target.value)}
          />
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
            variant={isUnban ? 'default' : 'destructive'}
            disabled={!canSubmit}
            onClick={() => mutation.mutate()}
          >
            {mutation.isPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
            {mutation.isPending ? t('common.saving') : t(`userBan.actions.${mode}`)}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
