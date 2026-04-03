'use client'

import { AlertTriangle, Check, X } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'

import { apiConfirmAction } from '@/services/api/agent'
import type {
  AgentConfirmResponseData,
  AgentConfirmationField,
  AgentConfirmationForm,
} from '@/services/api/agent'

export interface ConfirmActionCardProps {
  confirmId: string
  action: string
  description: string
  interaction?: string
  form?: AgentConfirmationForm
  onConfirmed: (result: AgentConfirmResponseData) => void
  onRejected: (result: AgentConfirmResponseData) => void
}

export function ConfirmActionCard({
  confirmId,
  action,
  description,
  interaction,
  form,
  onConfirmed,
  onRejected,
}: ConfirmActionCardProps) {
  const { t } = useTranslation()
  const [loading, setLoading] = useState<'confirm' | 'reject' | null>(null)
  const [settled, setSettled] = useState<'confirmed' | 'rejected' | null>(null)
  const [errorText, setErrorText] = useState<string | null>(null)
  const [dialogOpen, setDialogOpen] = useState(false)
  const [formValues, setFormValues] = useState<Record<string, string>>({})

  const fields = useMemo(() => form?.fields ?? [], [form?.fields])

  useEffect(() => {
    const nextValues: Record<string, string> = {}
    for (const field of fields) {
      if (field.defaultValue === undefined || field.defaultValue === null) {
        nextValues[field.key] = ''
        continue
      }
      nextValues[field.key] = String(field.defaultValue)
    }
    setFormValues(nextValues)
  }, [fields, confirmId])

  const getErrorMessage = (error: unknown) => {
    if (error && typeof error === 'object' && 'data' in error) {
      const backend = (error as { data?: { msg?: string; message?: string } }).data
      if (backend?.msg || backend?.message) {
        return backend.msg || backend.message || null
      }
    }
    if (error instanceof Error && error.message) {
      return error.message
    }
    return t('aiops.common.unknownError', { defaultValue: '未知错误' })
  }

  const buildConfirmPayload = () => {
    const payload: Record<string, unknown> = {}
    for (const field of fields) {
      const rawValue = formValues[field.key] ?? ''
      const trimmed = rawValue.trim()
      if (trimmed === '') {
        continue
      }
      if (field.type === 'number') {
        const parsed = Number(trimmed)
        if (!Number.isFinite(parsed)) {
          throw new Error(`${field.label} 需要是数字`)
        }
        payload[field.key] = parsed
        continue
      }
      payload[field.key] = trimmed
    }
    return payload
  }

  const validateForm = () => {
    for (const field of fields) {
      if (!field.required) continue
      const value = (formValues[field.key] ?? '').trim()
      if (!value) {
        return `${field.label}不能为空`
      }
    }
    return null
  }

  const submitConfirm = async (payload?: Record<string, unknown>) => {
    setErrorText(null)
    setLoading('confirm')
    try {
      const response = await apiConfirmAction(confirmId, true, payload)
      setSettled('confirmed')
      setDialogOpen(false)
      onConfirmed(response.data)
    } catch (error) {
      setErrorText(getErrorMessage(error))
    } finally {
      setLoading(null)
    }
  }

  const handleConfirm = async () => {
    if (settled || loading) return
    if (fields.length === 0 || interaction !== 'form') {
      await submitConfirm()
      return
    }
    const validationError = validateForm()
    if (validationError) {
      setErrorText(validationError)
      return
    }
    try {
      const payload = buildConfirmPayload()
      await submitConfirm(payload)
    } catch (error) {
      setErrorText(getErrorMessage(error))
    }
  }

  const handleReject = async () => {
    if (settled || loading) return
    setErrorText(null)
    setLoading('reject')
    try {
      const response = await apiConfirmAction(confirmId, false)
      setSettled('rejected')
      setDialogOpen(false)
      onRejected(response.data)
    } catch (error) {
      setErrorText(getErrorMessage(error))
    } finally {
      setLoading(null)
    }
  }

  const renderField = (field: AgentConfirmationField) => {
    const value = formValues[field.key] ?? ''
    if (field.type === 'textarea') {
      return (
        <Textarea
          value={value}
          placeholder={field.placeholder}
          onChange={(event) =>
            setFormValues((prev) => ({ ...prev, [field.key]: event.target.value }))
          }
        />
      )
    }
    if (field.type === 'select') {
      return (
        <Select
          value={value || undefined}
          onValueChange={(nextValue) =>
            setFormValues((prev) => ({ ...prev, [field.key]: nextValue }))
          }
        >
          <SelectTrigger className="w-full">
            <SelectValue placeholder={field.placeholder || field.label} />
          </SelectTrigger>
          <SelectContent>
            {(field.options ?? []).map((option) => (
              <SelectItem key={`${field.key}-${option.value}`} value={option.value}>
                {option.label}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      )
    }
    return (
      <Input
        type={field.type === 'number' ? 'number' : 'text'}
        value={value}
        placeholder={field.placeholder}
        onChange={(event) =>
          setFormValues((prev) => ({ ...prev, [field.key]: event.target.value }))
        }
      />
    )
  }

  return (
    <Card className="border-warning/40 bg-warning/5 min-w-0 space-y-3 overflow-hidden p-4">
      {/* Title row */}
      <div className="flex min-w-0 items-start gap-2">
        <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0 text-amber-500" />
        <div className="min-w-0 flex-1 space-y-1">
          <p className="text-sm leading-snug font-semibold">
            {t('aiops.agent.confirm.title', { defaultValue: '需要确认操作' })}
          </p>
          <p className="text-muted-foreground text-xs [overflow-wrap:anywhere] break-words">
            {action}
          </p>
        </div>
      </div>

      {/* Description */}
      {description && (
        <div className="bg-background rounded-md border px-3 py-2 text-xs leading-relaxed [overflow-wrap:anywhere] break-words">
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
          {interaction === 'form' && fields.length > 0 ? (
            <>
              <Button
                size="sm"
                className="h-7 flex-1 bg-green-600 text-xs text-white hover:bg-green-700"
                onClick={() => {
                  setErrorText(null)
                  setDialogOpen(true)
                }}
                disabled={!!loading}
              >
                <Check className="mr-1 h-3.5 w-3.5" />
                {form?.submitLabel ||
                  t('aiops.agent.confirm.confirmBtn', { defaultValue: '确认执行' })}
              </Button>
              <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogContent className="max-w-2xl">
                  <DialogHeader>
                    <DialogTitle>{form?.title || '补全配置'}</DialogTitle>
                    <DialogDescription>
                      {form?.description || description || action}
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4">
                    {fields.map((field) => (
                      <div key={field.key} className="space-y-1.5">
                        <div className="flex items-center gap-1 text-sm font-medium">
                          <span>{field.label}</span>
                          {field.required && <span className="text-red-500">*</span>}
                        </div>
                        {renderField(field)}
                        {field.description && (
                          <p className="text-muted-foreground text-xs">{field.description}</p>
                        )}
                      </div>
                    ))}
                    {errorText && <p className="text-xs text-red-500">{errorText}</p>}
                  </div>
                  <DialogFooter>
                    <Button variant="outline" onClick={handleReject} disabled={!!loading}>
                      {loading === 'reject'
                        ? t('aiops.agent.confirm.rejecting', { defaultValue: '拒绝中…' })
                        : t('aiops.agent.confirm.rejectBtn', { defaultValue: '拒绝' })}
                    </Button>
                    <Button
                      className="bg-green-600 text-white hover:bg-green-700"
                      onClick={handleConfirm}
                      disabled={!!loading}
                    >
                      {loading === 'confirm'
                        ? t('aiops.agent.confirm.confirming', { defaultValue: '确认中…' })
                        : form?.submitLabel ||
                          t('aiops.agent.confirm.confirmBtn', { defaultValue: '确认执行' })}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </>
          ) : (
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
          )}
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
      {errorText && !settled && <p className="text-xs text-red-500">{errorText}</p>}
    </Card>
  )
}
