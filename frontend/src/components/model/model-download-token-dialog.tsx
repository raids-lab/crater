/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Input } from '@/components/ui/input'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui-custom/alert-dialog'

interface ModelDownloadTokenDialogProps {
  action: 'resume' | 'retry'
  downloadName: string
  isPending: boolean
  open: boolean
  onOpenChange: (open: boolean) => void
  onSubmit: (token?: string) => void
}

export default function ModelDownloadTokenDialog({
  action,
  downloadName,
  isPending,
  open,
  onOpenChange,
  onSubmit,
}: ModelDownloadTokenDialogProps) {
  const { t } = useTranslation()
  const [token, setToken] = useState('')

  useEffect(() => {
    if (!open) {
      setToken('')
    }
  }, [open])

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {t(`modelDownload.action.${action}.title`, { name: downloadName })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t('modelDownload.action.tokenDescription')}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <Input
          type="password"
          autoComplete="off"
          value={token}
          onChange={(event) => setToken(event.target.value)}
          placeholder={t('modelDownload.action.tokenPlaceholder')}
          aria-label={t('modelDownload.action.tokenLabel')}
          disabled={isPending}
        />
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isPending}>{t('common.cancel')}</AlertDialogCancel>
          <AlertDialogAction
            disabled={isPending}
            onClick={(event) => {
              event.preventDefault()
              onSubmit(token.trim() || undefined)
            }}
          >
            {isPending
              ? t('modelDownload.action.processing')
              : t(`modelDownload.action.${action}.confirm`)}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
