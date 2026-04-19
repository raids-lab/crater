import { Link } from '@tanstack/react-router'
import { ArrowRightIcon, CoinsIcon, Loader2Icon, SaveIcon, SparklesIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Switch } from '@/components/ui/switch'

import CronJobStatusBadge from '@/components/badge/cronjob-status-badge'
import { BillingPeriodFields } from '@/components/form/billing-period-fields'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import {
  IBillingStatus,
  IGrantAllUsersExtraBalanceReq,
  ISetBillingStatusReq,
} from '@/services/api/system-config'

interface BillingSettingsProps {
  status?: IBillingStatus
  isSaving: boolean
  isResettingAll?: boolean
  isGrantingAllExtra?: boolean
  onSave: (payload: ISetBillingStatusReq) => void
  onResetAll: () => void
  onGrantAllExtra: (payload: IGrantAllUsersExtraBalanceReq) => void
}

interface BillingFormState {
  featureEnabled: boolean
  active: boolean
  runningSettlementEnabled: boolean
  runningSettlementIntervalMinutes: number
  jobFreeMinutes: number
  defaultIssueAmount: number
  defaultIssuePeriodMinutes: number
  accountIssueAmountOverrideEnabled: boolean
  accountIssuePeriodOverrideEnabled: boolean
  baseLoopCronStatus: string
  baseLoopCronEnabled: boolean
}

const DEFAULT_FORM_STATE: BillingFormState = {
  featureEnabled: false,
  active: false,
  runningSettlementEnabled: false,
  runningSettlementIntervalMinutes: 5,
  jobFreeMinutes: 0,
  defaultIssueAmount: 1000,
  defaultIssuePeriodMinutes: 43200,
  accountIssueAmountOverrideEnabled: false,
  accountIssuePeriodOverrideEnabled: false,
  baseLoopCronStatus: 'unknown',
  baseLoopCronEnabled: false,
}

export function BillingSettings({
  status,
  isSaving,
  isResettingAll = false,
  isGrantingAllExtra = false,
  onSave,
  onResetAll,
  onGrantAllExtra,
}: BillingSettingsProps) {
  const { t } = useTranslation()
  const [form, setForm] = useState<BillingFormState>(DEFAULT_FORM_STATE)
  const [periodDialogOpen, setPeriodDialogOpen] = useState(false)
  const [pendingPeriodMinutes, setPendingPeriodMinutes] = useState(43200)
  const [grantDialogOpen, setGrantDialogOpen] = useState(false)
  const [grantDelta, setGrantDelta] = useState(100)
  const [grantReason, setGrantReason] = useState('')

  useEffect(() => {
    if (status == null) {
      return
    }
    setForm({
      featureEnabled: Boolean(status.featureEnabled),
      active: Boolean(status.active),
      runningSettlementEnabled: Boolean(status.runningSettlementEnabled),
      runningSettlementIntervalMinutes: status.runningSettlementIntervalMinutes || 5,
      jobFreeMinutes: status.jobFreeMinutes || 0,
      defaultIssueAmount: status.defaultIssueAmount ?? 1000,
      defaultIssuePeriodMinutes: status.defaultIssuePeriodMinutes || 43200,
      accountIssueAmountOverrideEnabled: Boolean(status.accountIssueAmountOverrideEnabled),
      accountIssuePeriodOverrideEnabled: Boolean(status.accountIssuePeriodOverrideEnabled),
      baseLoopCronStatus: status.baseLoopCronStatus || 'unknown',
      baseLoopCronEnabled: Boolean(status.baseLoopCronEnabled),
    })
    setPendingPeriodMinutes(status.defaultIssuePeriodMinutes || 43200)
  }, [status])

  const setNumberField = (field: keyof BillingFormState, value: string) => {
    const num = Number(value)
    setForm((prev) => ({
      ...prev,
      [field]: Number.isFinite(num) ? num : 0,
    }))
  }

  const saveFeatureOnly = (enabled: boolean) => {
    setForm((prev) => ({ ...prev, featureEnabled: enabled }))
    onSave({ featureEnabled: enabled })
  }

  const submitAll = () => {
    onSave({
      featureEnabled: form.featureEnabled,
      active: form.active,
      runningSettlementEnabled: form.runningSettlementEnabled,
      runningSettlementIntervalMinutes: form.runningSettlementIntervalMinutes,
      jobFreeMinutes: form.jobFreeMinutes,
      defaultIssueAmount: form.defaultIssueAmount,
      defaultIssuePeriodMinutes: form.defaultIssuePeriodMinutes,
      accountIssueAmountOverrideEnabled: form.accountIssueAmountOverrideEnabled,
      accountIssuePeriodOverrideEnabled: form.accountIssuePeriodOverrideEnabled,
    })
  }

  const canSubmitGrant =
    Number.isFinite(grantDelta) &&
    Math.abs(grantDelta * 100 - Math.round(grantDelta * 100)) < 1e-8 &&
    grantDelta > 0 &&
    !isGrantingAllExtra

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <CoinsIcon className="text-primary h-5 w-5" />
          <CardTitle>
            {t('systemConfig.billing.title', {
              defaultValue: 'Billing 设置',
            })}
          </CardTitle>
        </div>
        <CardDescription>
          {t('systemConfig.billing.description', {
            defaultValue: '控制计费开关、运行中结算和默认周期发放参数。',
          })}
        </CardDescription>
      </CardHeader>

      <CardContent className="space-y-4">
        <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
          <div className="space-y-0.5">
            <Label className="text-base">
              {t('systemConfig.billing.featureSwitch', {
                defaultValue: '开启 Billing 功能',
              })}
            </Label>
            <p className="text-muted-foreground text-[0.8rem]">
              {t('systemConfig.billing.featureSwitchDesc', {
                defaultValue: '关闭时，Billing 相关字段与操作不显示。',
              })}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {isSaving && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
            <Switch
              checked={form.featureEnabled}
              disabled={isSaving}
              onCheckedChange={saveFeatureOnly}
            />
          </div>
        </div>

        {form.featureEnabled && (
          <>
            <Separator />

            <div className="grid gap-4 md:grid-cols-2">
              <div className="flex items-center justify-between rounded-lg border p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">
                    {t('systemConfig.billing.active', {
                      defaultValue: '计费生效',
                    })}
                  </Label>
                  <p className="text-muted-foreground text-[0.8rem]">
                    {t('systemConfig.billing.activeDesc', {
                      defaultValue: '开启后，作业创建和结算进入计费逻辑。',
                    })}
                  </p>
                </div>
                <Switch
                  checked={form.active}
                  disabled={isSaving}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({
                      ...prev,
                      active: checked,
                      runningSettlementEnabled: checked ? prev.runningSettlementEnabled : false,
                    }))
                  }
                />
              </div>

              <div className="flex items-center justify-between rounded-lg border p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">
                    {t('systemConfig.billing.runningSettlementEnabled', {
                      defaultValue: '开启运行中结算',
                    })}
                  </Label>
                  <p className="text-muted-foreground text-[0.8rem]">
                    {t('systemConfig.billing.runningSettlementEnabledDesc', {
                      defaultValue: '开启后按小周期结算 Running 作业。',
                    })}
                  </p>
                </div>
                <Switch
                  checked={form.runningSettlementEnabled}
                  disabled={
                    isSaving ||
                    !form.active ||
                    (!form.baseLoopCronEnabled && !form.runningSettlementEnabled)
                  }
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({ ...prev, runningSettlementEnabled: checked }))
                  }
                />
              </div>
            </div>
            <div className="rounded-lg border p-4">
              <div className="flex items-start justify-between gap-3">
                <div className="space-y-0.5">
                  <Label className="text-base">
                    {t('systemConfig.billing.baseLoopTitle', {
                      defaultValue: 'Billing 基础循环',
                    })}
                  </Label>
                  <p className="text-muted-foreground text-[0.8rem]">
                    {t('systemConfig.billing.baseLoopDesc', {
                      defaultValue: '负责周期发放和运行中结算触发。',
                    })}
                  </p>
                </div>
                <CronJobStatusBadge status={form.baseLoopCronStatus} />
              </div>
              <div className="mt-3">
                <Button variant="secondary" asChild>
                  <Link to="/admin/cronjobs">
                    {t('systemConfig.billing.goToCronPolicy', {
                      defaultValue: '前往 CronPolicy 配置',
                    })}
                    <ArrowRightIcon className="ml-2 h-4 w-4" />
                  </Link>
                </Button>
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="flex items-center justify-between rounded-lg border p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">
                    {t('systemConfig.billing.accountIssueAmountOverrideEnabled', {
                      defaultValue: '允许账户独立配置发放额度',
                    })}
                  </Label>
                  <p className="text-muted-foreground text-[0.8rem]">
                    {t('systemConfig.billing.accountIssueAmountOverrideEnabledDesc', {
                      defaultValue: '关闭时所有账户统一使用系统默认发放额度。',
                    })}
                  </p>
                </div>
                <Switch
                  checked={form.accountIssueAmountOverrideEnabled}
                  disabled={isSaving}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({ ...prev, accountIssueAmountOverrideEnabled: checked }))
                  }
                />
              </div>
              <div className="flex items-center justify-between rounded-lg border p-4">
                <div className="space-y-0.5">
                  <Label className="text-base">
                    {t('systemConfig.billing.accountIssuePeriodOverrideEnabled', {
                      defaultValue: '允许账户独立配置发放周期',
                    })}
                  </Label>
                  <p className="text-muted-foreground text-[0.8rem]">
                    {t('systemConfig.billing.accountIssuePeriodOverrideEnabledDesc', {
                      defaultValue: '关闭时所有账户统一使用系统默认发放周期。',
                    })}
                  </p>
                </div>
                <Switch
                  checked={form.accountIssuePeriodOverrideEnabled}
                  disabled={isSaving}
                  onCheckedChange={(checked) =>
                    setForm((prev) => ({ ...prev, accountIssuePeriodOverrideEnabled: checked }))
                  }
                />
              </div>
            </div>

            <div className="grid gap-4 md:grid-cols-2">
              <div className="space-y-2">
                <Label>
                  {t('systemConfig.billing.runningSettlementIntervalMinutes', {
                    defaultValue: '运行中结算周期（分钟）',
                  })}
                </Label>
                <Input
                  type="number"
                  min={1}
                  value={form.runningSettlementIntervalMinutes}
                  disabled={isSaving || !form.active || !form.runningSettlementEnabled}
                  onChange={(e) =>
                    setNumberField('runningSettlementIntervalMinutes', e.target.value)
                  }
                />
              </div>
              <div className="space-y-2">
                <Label>
                  {t('systemConfig.billing.jobFreeMinutes', {
                    defaultValue: '作业开始免费时长（分钟）',
                  })}
                </Label>
                <Input
                  type="number"
                  min={0}
                  value={form.jobFreeMinutes}
                  disabled={isSaving}
                  onChange={(e) => setNumberField('jobFreeMinutes', e.target.value)}
                />
                <p className="text-muted-foreground text-xs">
                  {t('systemConfig.billing.jobFreeMinutesDesc', {
                    defaultValue: '作业开始后可先免费运行的分钟数。',
                  })}
                </p>
              </div>
              <div className="space-y-2">
                <Label>
                  {t('systemConfig.billing.defaultIssueAmount', {
                    defaultValue: '默认周期发放额度',
                  })}
                </Label>
                <Input
                  type="number"
                  min={0}
                  step={0.01}
                  value={form.defaultIssueAmount}
                  onChange={(e) => setNumberField('defaultIssueAmount', e.target.value)}
                />
                <p className="text-muted-foreground text-xs">单位为点数，最多保留两位小数。</p>
              </div>
              <div className="space-y-2">
                <Label>
                  {t('systemConfig.billing.defaultIssuePeriodMinutes', {
                    defaultValue: '默认发放周期（分钟）',
                  })}
                </Label>
                <Button
                  variant="outline"
                  type="button"
                  onClick={() => {
                    setPendingPeriodMinutes(form.defaultIssuePeriodMinutes)
                    setPeriodDialogOpen(true)
                  }}
                >
                  编辑默认发放周期（当前 {form.defaultIssuePeriodMinutes} 分钟）
                </Button>
              </div>
            </div>

            <div className="flex flex-wrap justify-end gap-2">
              <Button
                type="button"
                variant="outline"
                disabled={isSaving || isGrantingAllExtra}
                onClick={() => setGrantDialogOpen(true)}
              >
                {isGrantingAllExtra ? (
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <SparklesIcon className="mr-2 h-4 w-4" />
                )}
                给所有用户发放 extra 点数
              </Button>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <Button type="button" variant="outline" disabled={isSaving || isResettingAll}>
                    {isResettingAll ? <Loader2Icon className="mr-2 h-4 w-4 animate-spin" /> : null}
                    重置全平台免费额度
                  </Button>
                </AlertDialogTrigger>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>确认重置全平台免费额度？</AlertDialogTitle>
                    <p className="text-muted-foreground text-sm">
                      这会重置所有账户成员的本周期免费额度。
                    </p>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>取消</AlertDialogCancel>
                    <AlertDialogAction onClick={onResetAll} disabled={isResettingAll}>
                      确认重置
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>
              <Button type="button" disabled={isSaving} onClick={submitAll}>
                {isSaving ? (
                  <Loader2Icon className="mr-2 h-4 w-4 animate-spin" />
                ) : (
                  <SaveIcon className="mr-2 h-4 w-4" />
                )}
                {t('common.save')}
              </Button>
            </div>
          </>
        )}
      </CardContent>
      <Dialog open={periodDialogOpen} onOpenChange={setPeriodDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>默认发放周期</DialogTitle>
          </DialogHeader>
          <BillingPeriodFields
            totalMinutes={pendingPeriodMinutes}
            onChange={(value) => setPendingPeriodMinutes(value.totalMinutes)}
          />
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setPeriodDialogOpen(false)}>
              取消
            </Button>
            <Button
              type="button"
              onClick={() => {
                setForm((prev) => ({ ...prev, defaultIssuePeriodMinutes: pendingPeriodMinutes }))
                setPeriodDialogOpen(false)
              }}
            >
              应用
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
      <Dialog open={grantDialogOpen} onOpenChange={setGrantDialogOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>给所有用户发放 extra 点数</DialogTitle>
          </DialogHeader>
          <div className="space-y-4">
            <div className="space-y-2">
              <Label>发放额度</Label>
              <Input
                type="number"
                min={0}
                step={0.01}
                value={Number.isFinite(grantDelta) ? grantDelta : 0}
                onChange={(e) => setGrantDelta(Number(e.target.value))}
              />
              <p className="text-muted-foreground text-xs">单位为 extra 点数，最多保留两位小数。</p>
            </div>
            <div className="space-y-2">
              <Label>原因备注</Label>
              <Input
                value={grantReason}
                placeholder="例如：节日奖励 / 系统补偿"
                onChange={(e) => setGrantReason(e.target.value)}
              />
            </div>
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setGrantDialogOpen(false)}>
              取消
            </Button>
            <Button
              type="button"
              disabled={!canSubmitGrant}
              onClick={() => {
                onGrantAllExtra({
                  delta: grantDelta,
                  reason: grantReason.trim() || undefined,
                })
                setGrantDialogOpen(false)
              }}
            >
              {isGrantingAllExtra ? <Loader2Icon className="mr-2 h-4 w-4 animate-spin" /> : null}
              确认发放
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  )
}
