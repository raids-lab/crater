'use client'

import { AlertTriangle, Check, X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'

import { apiConfirmAction } from '@/services/api/agent'

export interface ConfirmActionCardProps {
  confirmId: string
  action: string
  description: string
  onConfirmed: () => void
  onRejected: () => void
}

export function ConfirmActionCard({
  confirmId,
  action,
  description,
  onConfirmed,
  onRejected,
}: ConfirmActionCardProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState<'confirm' | 'reject' | null>(null)
  const [settled, setSettled] = useState<'confirmed' | 'rejected' | null>(null)

  const handleConfirm = async () => {
    if (settled || loading) return
    setLoading('confirm')
    try {
      await apiConfirmAction(confirmId, true)
      setSettled('confirmed')
      onConfirmed()
    } catch {
      // Keep card interactive on error so the user can retry
    } finally {
      setLoading(null)
    }
  }

  const handleReject = async () => {
    if (settled || loading) return
    setLoading('reject')
    try {
      await apiConfirmAction(confirmId, false)
      setSettled('rejected')
      onRejected()
    } catch {
      // Keep card interactive on error
    } finally {
      setLoading(null)
    }
  }

  return (
    <Card className="border-warning/40 bg-warning/5 space-y-3 p-4">
      {/* Title row */}
      <div className="flex items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
        <div className="flex-1 space-y-1">
          <p className="text-sm font-semibold leading-snug">
            {t('aiops.agent.confirm.title', { defaultValue: '需要确认操作' })}
          </p>
          <p className="text-muted-foreground text-xs">{action}</p>
        </div>
      </div>

      {/* Description */}
      {description && (
        <div className="bg-background rounded-md border px-3 py-2 text-xs leading-relaxed">
          {description}
        </div>
      )}

      {/* Settled state */}
      {settled === 'confirmed' && (
        <p className="flex items-center gap-1 text-xs text-green-600 dark:text-green-400">
          <Check className="h-3.5 w-3.5" />
          {t('aiops.agent.confirm.confirmed', { defaultValue: '已确认，正在执行…' })}
        </p>
      )}
      {settled === 'rejected' && (
        <p className="flex items-center gap-1 text-xs text-red-600 dark:text-red-400">
          <X className="h-3.5 w-3.5" />
          {t('aiops.agent.confirm.rejected', { defaultValue: '已拒绝操作' })}
        </p>
      )}

      {/* Action buttons */}
      {!settled && (
        <div className="flex gap-2">
          <Button
            size="sm"
            className="h-7 flex-1 bg-green-600 text-xs text-white hover:bg-green-700"
            onClick={handleConfirm}
            disabled={!!loading}
          >
            {loading === 'confirm' ? (
              <span className="animate-pulse">
                {t('aiops.agent.confirm.confirming', { defaultValue: '确认中…' })}
              </span>
            ) : (
              <>
                <Check className="mr-1 h-3.5 w-3.5" />
                {t('aiops.agent.confirm.confirmBtn', { defaultValue: '确认执行' })}
              </>
            )}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="text-destructive hover:text-destructive h-7 flex-1 border-red-400 text-xs hover:bg-red-50 dark:hover:bg-red-950"
            onClick={handleReject}
            disabled={!!loading}
          >
            {loading === 'reject' ? (
              <span className="animate-pulse">
                {t('aiops.agent.confirm.rejecting', { defaultValue: '拒绝中…' })}
              </span>
            ) : (
              <>
                <X className="mr-1 h-3.5 w-3.5" />
                {t('aiops.agent.confirm.rejectBtn', { defaultValue: '拒绝' })}
              </>
            )}
          </Button>
        </div>
      )}
    </Card>
  )
}
