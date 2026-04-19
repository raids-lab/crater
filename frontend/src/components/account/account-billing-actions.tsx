import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2Icon, RotateCcwIcon, Settings2Icon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

import {
  BillingPeriodFields,
  billingPeriodFromMinutes,
} from '@/components/form/billing-period-fields'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import {
  apiAdminGetAccountBillingConfig,
  apiAdminResetAccountBillingBalance,
  apiAdminUpdateAccountBillingConfig,
  apiGetAccountBillingConfig,
  apiResetAccountBillingBalance,
  apiUpdateAccountBillingConfig,
} from '@/services/api/billing'
import { apiGetBillingStatus } from '@/services/api/system-config'

import { isBillingVisible } from '@/utils/billing-visibility'

interface AccountBillingActionsProps {
  accountId: number
  isAdminView: boolean
  editable?: boolean
}

const hasAtMostTwoDecimalPlaces = (value: number) =>
  Math.abs(value * 100 - Math.round(value * 100)) < 1e-8

const isBillingAmountValueValid = (value: number | null | undefined) =>
  value != null && Number.isFinite(value) && value >= 0 && hasAtMostTwoDecimalPlaces(value)

const formatPeriodSummary = (totalMinutes?: number) => {
  if ((totalMinutes ?? 0) <= 0) {
    return '已关闭'
  }
  const { days, hours, minutes } = billingPeriodFromMinutes(totalMinutes)
  const parts = [
    days > 0 ? `${days} 天` : null,
    hours > 0 ? `${hours} 小时` : null,
    minutes > 0 ? `${minutes} 分钟` : null,
  ].filter(Boolean)
  return parts.join(' ') || `${totalMinutes} 分钟`
}

export function AccountBillingActions({
  accountId,
  isAdminView,
  editable = true,
}: AccountBillingActionsProps) {
  const queryClient = useQueryClient()
  const [dialogOpen, setDialogOpen] = useState(false)
  const { data: billingStatus } = useQuery({
    queryKey: ['system-config', 'billing-status'],
    queryFn: () => apiGetBillingStatus().then((res) => res.data),
    enabled: editable,
  })
  const billingEnabled = editable && isBillingVisible(billingStatus, isAdminView ? 'admin' : 'user')
  const amountOverrideEnabled =
    billingEnabled && (billingStatus?.accountIssueAmountOverrideEnabled ?? false)
  const periodOverrideEnabled =
    billingEnabled && (billingStatus?.accountIssuePeriodOverrideEnabled ?? false)

  const getAccountBillingConfig = isAdminView
    ? apiAdminGetAccountBillingConfig
    : apiGetAccountBillingConfig
  const updateAccountBillingConfig = isAdminView
    ? apiAdminUpdateAccountBillingConfig
    : apiUpdateAccountBillingConfig
  const resetAccountBilling = isAdminView
    ? apiAdminResetAccountBillingBalance
    : apiResetAccountBillingBalance

  const accountBillingConfigQuery = useQuery({
    queryKey: ['account', accountId, 'billing-config', isAdminView ? 'admin' : 'user'],
    queryFn: () => getAccountBillingConfig(accountId).then((res) => res.data),
    enabled: billingEnabled,
  })

  const accountBillingConfig = accountBillingConfigQuery.data

  const [issueAmountInput, setIssueAmountInput] = useState('')
  const [issueAmountDirty, setIssueAmountDirty] = useState(false)
  const [issuePeriodMinutes, setIssuePeriodMinutes] = useState<number | undefined>(undefined)
  const [issuePeriodDirty, setIssuePeriodDirty] = useState(false)

  useEffect(() => {
    if (!dialogOpen) {
      return
    }
    setIssueAmountInput(
      accountBillingConfig?.issueAmount != null
        ? String(accountBillingConfig.issueAmount)
        : accountBillingConfig?.effectiveIssueAmount != null
          ? String(accountBillingConfig.effectiveIssueAmount)
          : ''
    )
    setIssueAmountDirty(false)
    setIssuePeriodMinutes(
      accountBillingConfig?.issuePeriodMinutes ??
        accountBillingConfig?.effectiveIssuePeriodMinutes ??
        0
    )
    setIssuePeriodDirty(false)
  }, [
    accountBillingConfig?.effectiveIssueAmount,
    accountBillingConfig?.effectiveIssuePeriodMinutes,
    accountBillingConfig?.issueAmount,
    accountBillingConfig?.issuePeriodMinutes,
    dialogOpen,
  ])

  const parsedIssueAmount = issueAmountInput.trim()
  const hasIssueAmount = parsedIssueAmount !== ''
  const nextIssueAmount = hasIssueAmount ? Number(parsedIssueAmount) : undefined
  const isIssueAmountValid = !hasIssueAmount || isBillingAmountValueValid(nextIssueAmount)

  const payload = useMemo(() => {
    const nextPayload: { issueAmount?: number; issuePeriodMinutes?: number } = {}
    if (
      amountOverrideEnabled &&
      issueAmountDirty &&
      hasIssueAmount &&
      isIssueAmountValid &&
      nextIssueAmount !== undefined
    ) {
      nextPayload.issueAmount = nextIssueAmount
    }
    if (periodOverrideEnabled && issuePeriodDirty && issuePeriodMinutes !== undefined) {
      nextPayload.issuePeriodMinutes = issuePeriodMinutes
    }
    return nextPayload
  }, [
    amountOverrideEnabled,
    hasIssueAmount,
    isIssueAmountValid,
    issueAmountDirty,
    issuePeriodDirty,
    issuePeriodMinutes,
    nextIssueAmount,
    periodOverrideEnabled,
  ])

  const canSave = isIssueAmountValid && Object.keys(payload).length > 0
  const isUsingDefaultBillingIssueAmount = accountBillingConfig?.issueAmount == null
  const isUsingDefaultBillingIssuePeriod = accountBillingConfig?.issuePeriodMinutes == null

  const effectiveBillingSummary = useMemo(() => {
    if (!accountBillingConfig) {
      return '当前生效额度以系统默认配置为准。'
    }
    if (accountBillingConfig.effectiveIssuePeriodMinutes <= 0) {
      return `当前生效额度为 ${accountBillingConfig.effectiveIssueAmount} 点，周期发放已关闭。`
    }
    return `当前生效配置：每 ${formatPeriodSummary(accountBillingConfig.effectiveIssuePeriodMinutes)} 发放 ${accountBillingConfig.effectiveIssueAmount} 点。`
  }, [accountBillingConfig])

  const { mutate: saveBillingConfig, isPending: isSavingBillingConfig } = useMutation({
    mutationFn: () => updateAccountBillingConfig(accountId, payload),
    onSuccess: async () => {
      toast.success('点数发放配置已更新')
      setDialogOpen(false)
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'billing-config'],
      })
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'billing-members'],
      })
    },
    onError: () => {
      toast.error('点数发放配置更新失败')
    },
  })

  const { mutate: resetAccountBillingNow, isPending: isResettingBilling } = useMutation({
    mutationFn: () => resetAccountBilling(accountId),
    onSuccess: async () => {
      toast.success('已重置当前账户全部成员免费额度')
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'billing-config'],
      })
      await queryClient.invalidateQueries({
        queryKey: ['account', accountId, 'billing-members'],
      })
    },
    onError: () => {
      toast.error('重置免费额度失败')
    },
  })

  if (!billingEnabled) {
    return null
  }

  return (
    <div className="flex flex-wrap justify-end gap-3">
      <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
        <DialogTrigger asChild>
          <Button>
            <Settings2Icon className="mr-2 size-4" />
            点数发放配置
          </Button>
        </DialogTrigger>
        <DialogContent className="max-h-[85vh] overflow-y-auto p-0 sm:max-w-2xl">
          <DialogHeader className="px-6 pt-6">
            <DialogTitle>配置账户点数发放</DialogTitle>
            <DialogDescription>配置当前账户的默认点数发放规则。</DialogDescription>
          </DialogHeader>
          <div className="space-y-5 px-6 pb-6">
            <div className="bg-muted/30 rounded-xl border p-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="bg-background rounded-lg border p-4">
                  <p className="text-muted-foreground text-xs">当前生效发放额度</p>
                  <p className="mt-2 text-2xl font-semibold">
                    {accountBillingConfig?.effectiveIssueAmount ?? 0}
                    <span className="text-muted-foreground ml-1 text-sm font-medium">点</span>
                  </p>
                </div>
                <div className="bg-background rounded-lg border p-4">
                  <p className="text-muted-foreground text-xs">当前生效发放周期</p>
                  <p className="mt-2 text-lg font-semibold">
                    {formatPeriodSummary(accountBillingConfig?.effectiveIssuePeriodMinutes)}
                  </p>
                </div>
              </div>
              <p className="text-muted-foreground mt-3 text-xs">{effectiveBillingSummary}</p>
            </div>

            <div className="space-y-4">
              <section className="space-y-3 rounded-xl border p-4">
                <div className="space-y-1">
                  <Label>周期发放额度</Label>
                  <p className="text-muted-foreground text-xs">
                    {amountOverrideEnabled
                      ? isUsingDefaultBillingIssueAmount
                        ? `当前展示的是生效值 ${accountBillingConfig?.effectiveIssueAmount ?? 0} 点。`
                        : '当前展示的是账户已配置额度。'
                      : '当前由系统默认发放额度控制。'}
                  </p>
                </div>
                <Input
                  type="number"
                  min={0}
                  step={0.01}
                  disabled={!amountOverrideEnabled || accountBillingConfigQuery.isLoading}
                  value={issueAmountInput}
                  onChange={(e) => {
                    setIssueAmountInput(e.target.value)
                    setIssueAmountDirty(true)
                  }}
                  placeholder="留空表示沿用当前配置"
                />
                {!isIssueAmountValid ? (
                  <p className="text-destructive text-xs">请输入非负点数，最多保留两位小数。</p>
                ) : null}
              </section>

              <section className="space-y-3 rounded-xl border p-4">
                <div className="space-y-1">
                  <Label>发放周期</Label>
                  <p className="text-muted-foreground text-xs">
                    {periodOverrideEnabled
                      ? isUsingDefaultBillingIssuePeriod
                        ? `当前展示的是生效周期 ${formatPeriodSummary(accountBillingConfig?.effectiveIssuePeriodMinutes)}。`
                        : '填 0 表示关闭当前账户的周期发放。'
                      : '当前由系统默认发放周期控制。'}
                  </p>
                </div>
                <BillingPeriodFields
                  totalMinutes={issuePeriodMinutes}
                  disabled={!periodOverrideEnabled || accountBillingConfigQuery.isLoading}
                  onChange={(value) => {
                    setIssuePeriodMinutes(value.totalMinutes)
                    setIssuePeriodDirty(true)
                  }}
                />
              </section>
            </div>
          </div>
          <DialogFooter className="border-t px-6 py-4">
            <DialogClose asChild>
              <Button variant="outline">取消</Button>
            </DialogClose>
            <Button
              onClick={() => saveBillingConfig()}
              disabled={isSavingBillingConfig || !canSave}
            >
              {isSavingBillingConfig ? <Loader2Icon className="mr-2 size-4 animate-spin" /> : null}
              保存
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button variant="outline" disabled={isResettingBilling}>
            {isResettingBilling ? (
              <Loader2Icon className="mr-2 size-4 animate-spin" />
            ) : (
              <RotateCcwIcon className="mr-2 size-4" />
            )}
            一键重置免费额度
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认重置当前账户免费额度？</AlertDialogTitle>
            <AlertDialogDescription>
              将重置当前账户所有成员的本周期免费额度。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => resetAccountBillingNow()}
              disabled={isResettingBilling}
            >
              确认重置
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
