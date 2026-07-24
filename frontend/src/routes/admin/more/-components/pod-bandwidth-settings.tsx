import { CircleAlertIcon, CircleCheckIcon, Loader2Icon, NetworkIcon, SaveIcon } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'

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

import type { IPodBandwidthConfig, IUpdatePodBandwidthConfig } from '@/services/api/system-config'

const bandwidthUnits = ['K', 'M', 'G'] as const
type BandwidthUnit = (typeof bandwidthUnits)[number]

interface BandwidthValue {
  amount: string
  unit: BandwidthUnit
}

interface BandwidthFormState {
  modelDownload: BandwidthValue
  jobIngress: BandwidthValue
  jobEgress: BandwidthValue
}

interface BandwidthFieldProps {
  id: string
  label: string
  description: string
  value: BandwidthValue
  disabled: boolean
  onChange: (value: BandwidthValue) => void
}

interface PodBandwidthSettingsProps {
  config?: IPodBandwidthConfig
  isPending: boolean
  isLoading: boolean
  isError: boolean
  onRetry: () => void
  onSubmit: (config: IUpdatePodBandwidthConfig) => Promise<unknown>
}

const defaultBandwidth: BandwidthValue = { amount: '1', unit: 'G' }

function parseBandwidth(value: string): BandwidthValue {
  const match = /^(\d+(?:\.\d+)?)(K|M|G)$/.exec(value.trim())
  if (!match) return { ...defaultBandwidth }
  return { amount: match[1], unit: match[2] as BandwidthUnit }
}

function formatBandwidth(value: BandwidthValue): string {
  return `${value.amount.trim()}${value.unit}`
}

function isPositiveAmount(value: string): boolean {
  const amount = Number(value)
  return Number.isFinite(amount) && amount > 0
}

function BandwidthField({
  id,
  label,
  description,
  value,
  disabled,
  onChange,
}: BandwidthFieldProps) {
  const { t } = useTranslation()

  return (
    <div className="grid gap-3 px-4 py-3 sm:grid-cols-[minmax(0,1fr)_minmax(16rem,22rem)] sm:items-center sm:gap-6">
      <div className="space-y-1">
        <Label htmlFor={`${id}-amount`}>{label}</Label>
        <p className="text-muted-foreground text-xs leading-relaxed">{description}</p>
      </div>
      <div className="flex min-w-0 sm:w-full sm:justify-self-end">
        <Input
          id={`${id}-amount`}
          type="number"
          min="0"
          step="any"
          inputMode="decimal"
          value={value.amount}
          onChange={(event) => onChange({ ...value, amount: event.target.value })}
          disabled={disabled}
          className="min-w-0 rounded-r-none"
        />
        <Select
          value={value.unit}
          onValueChange={(unit) => onChange({ ...value, unit: unit as BandwidthUnit })}
          disabled={disabled}
        >
          <SelectTrigger
            aria-label={t('systemConfig.podBandwidth.unit')}
            className="w-[7.5rem] shrink-0 rounded-l-none border-l-0"
          >
            <SelectValue />
          </SelectTrigger>
          <SelectContent>
            {bandwidthUnits.map((unit) => (
              <SelectItem key={unit} value={unit}>
                {t(`systemConfig.podBandwidth.unit${unit}`)}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
    </div>
  )
}

export function PodBandwidthSettings({
  config,
  isPending,
  isLoading,
  isError,
  onRetry,
  onSubmit,
}: PodBandwidthSettingsProps) {
  const { t } = useTranslation()
  const [enabled, setEnabled] = useState(false)
  const [limits, setLimits] = useState<BandwidthFormState>({
    modelDownload: { ...defaultBandwidth },
    jobIngress: { ...defaultBandwidth },
    jobEgress: { ...defaultBandwidth },
  })
  const [pendingConfig, setPendingConfig] = useState<IUpdatePodBandwidthConfig | null>(null)
  const [isConfirming, setIsConfirming] = useState(false)

  useEffect(() => {
    if (!config) return
    setEnabled(config.enabled)
    setLimits({
      modelDownload: parseBandwidth(config.modelDownloadBandwidth),
      jobIngress: parseBandwidth(config.jobIngressBandwidth),
      jobEgress: parseBandwidth(config.jobEgressBandwidth),
    })
  }, [config])

  const updateLimit = (key: keyof BandwidthFormState, value: BandwidthValue) => {
    setLimits((current) => ({ ...current, [key]: value }))
  }

  const handleSubmit = () => {
    if (Object.values(limits).some((value) => !isPositiveAmount(value.amount))) {
      toast.error(t('systemConfig.podBandwidth.invalidBandwidth'))
      return
    }
    if (enabled && !config?.capabilityAvailable) {
      toast.error(t('systemConfig.podBandwidth.capabilityRequired'))
      return
    }
    setPendingConfig({
      enabled,
      modelDownloadBandwidth: formatBandwidth(limits.modelDownload),
      jobIngressBandwidth: formatBandwidth(limits.jobIngress),
      jobEgressBandwidth: formatBandwidth(limits.jobEgress),
    })
  }

  const controlsDisabled = isPending || isLoading || isError || !config
  const confirmationPending = isPending || isConfirming
  const cannotEnable = !config?.capabilityAvailable && !enabled
  const displayBandwidth = (value: string) => {
    const parsed = parseBandwidth(value)
    return `${parsed.amount} ${t(`systemConfig.podBandwidth.unit${parsed.unit}`)}`
  }

  return (
    <>
      <CardHeader className="pb-3">
        <div className="flex flex-wrap items-center gap-2">
          <NetworkIcon
            className={enabled ? 'size-5 text-blue-500' : 'text-muted-foreground size-5'}
          />
          <CardTitle>{t('systemConfig.podBandwidth.title')}</CardTitle>

          {isLoading ? (
            <Badge variant="outline" className="text-muted-foreground w-fit gap-1.5">
              <Loader2Icon className="size-3.5 animate-spin" />
              {t('systemConfig.podBandwidth.checkingCapability')}
            </Badge>
          ) : (
            config && (
              <Badge
                variant="outline"
                className={
                  config.capabilityAvailable
                    ? 'w-fit gap-1.5 border-emerald-500/40 text-emerald-700 dark:text-emerald-400'
                    : 'text-destructive border-destructive/40 w-fit gap-1.5'
                }
              >
                {config.capabilityAvailable ? (
                  <CircleCheckIcon className="size-3.5" />
                ) : (
                  <CircleAlertIcon className="size-3.5" />
                )}
                {config.capabilityAvailable
                  ? t('systemConfig.podBandwidth.capabilityAvailable')
                  : t('systemConfig.podBandwidth.capabilityUnavailable')}
              </Badge>
            )
          )}
        </div>
        <CardDescription>{t('systemConfig.podBandwidth.description')}</CardDescription>

        {config && !config.capabilityAvailable && config.capabilityMessage && (
          <p className="text-destructive mt-1 text-xs break-words" role="alert">
            {config.capabilityMessage}
          </p>
        )}
      </CardHeader>

      <CardContent className="space-y-4 pt-0">
        {isError && (
          <div
            className="border-destructive/50 bg-destructive/5 text-destructive flex flex-col gap-3 rounded-md border p-3 text-sm sm:flex-row sm:items-center sm:justify-between"
            role="alert"
          >
            <span>{t('systemConfig.podBandwidth.loadError')}</span>
            <Button type="button" variant="outline" size="sm" onClick={onRetry}>
              {t('systemConfig.podBandwidth.retryLoad')}
            </Button>
          </div>
        )}

        <div className="bg-muted/35 flex items-center justify-between gap-6 rounded-lg px-4 py-3">
          <div className="space-y-1">
            <Label htmlFor="pod-bandwidth-enabled">{t('systemConfig.podBandwidth.enabled')}</Label>
            <p className="text-muted-foreground text-xs leading-relaxed">
              {t('systemConfig.podBandwidth.enabledDescription')}
            </p>
          </div>
          <Switch
            id="pod-bandwidth-enabled"
            checked={enabled}
            onCheckedChange={setEnabled}
            disabled={controlsDisabled || cannotEnable}
          />
        </div>

        {enabled && (
          <div className="divide-y overflow-hidden rounded-lg border">
            <BandwidthField
              id="model-download-bandwidth"
              label={t('systemConfig.podBandwidth.modelDownload')}
              description={t('systemConfig.podBandwidth.modelDownloadDescription')}
              value={limits.modelDownload}
              disabled={controlsDisabled}
              onChange={(value) => updateLimit('modelDownload', value)}
            />
            <BandwidthField
              id="job-ingress-bandwidth"
              label={t('systemConfig.podBandwidth.jobIngress')}
              description={t('systemConfig.podBandwidth.jobIngressDescription')}
              value={limits.jobIngress}
              disabled={controlsDisabled}
              onChange={(value) => updateLimit('jobIngress', value)}
            />
            <BandwidthField
              id="job-egress-bandwidth"
              label={t('systemConfig.podBandwidth.jobEgress')}
              description={t('systemConfig.podBandwidth.jobEgressDescription')}
              value={limits.jobEgress}
              disabled={controlsDisabled}
              onChange={(value) => updateLimit('jobEgress', value)}
            />
          </div>
        )}

        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          {enabled ? (
            <p className="text-muted-foreground text-xs leading-relaxed">
              {t('systemConfig.podBandwidth.existingPodsNote')}
            </p>
          ) : (
            <span />
          )}
          <Button
            className="shrink-0 self-end"
            onClick={handleSubmit}
            disabled={controlsDisabled || (enabled && !config?.capabilityAvailable)}
          >
            {isPending ? (
              <Loader2Icon className="mr-2 size-4 animate-spin" />
            ) : (
              <SaveIcon className="mr-2 size-4" />
            )}
            {t('common.save')}
          </Button>
        </div>
      </CardContent>

      <AlertDialog
        open={pendingConfig !== null}
        onOpenChange={(open) => !open && !confirmationPending && setPendingConfig(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('systemConfig.podBandwidth.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('systemConfig.podBandwidth.confirmDescription')}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingConfig && (
            <dl className="bg-muted/50 grid grid-cols-[minmax(8rem,auto)_1fr] gap-x-4 gap-y-2 rounded-md p-4 text-sm">
              <dt className="text-muted-foreground">
                {t('systemConfig.podBandwidth.confirmEnabled')}
              </dt>
              <dd>
                {pendingConfig.enabled
                  ? t('systemConfig.podBandwidth.statusEnabled')
                  : t('systemConfig.podBandwidth.statusDisabled')}
              </dd>
              {pendingConfig.enabled && (
                <>
                  <dt className="text-muted-foreground">
                    {t('systemConfig.podBandwidth.modelDownload')}
                  </dt>
                  <dd>{displayBandwidth(pendingConfig.modelDownloadBandwidth)}</dd>
                  <dt className="text-muted-foreground">
                    {t('systemConfig.podBandwidth.jobIngress')}
                  </dt>
                  <dd>{displayBandwidth(pendingConfig.jobIngressBandwidth)}</dd>
                  <dt className="text-muted-foreground">
                    {t('systemConfig.podBandwidth.jobEgress')}
                  </dt>
                  <dd>{displayBandwidth(pendingConfig.jobEgressBandwidth)}</dd>
                </>
              )}
            </dl>
          )}
          <p className="text-muted-foreground text-sm">
            {t('systemConfig.podBandwidth.confirmScope')}
          </p>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={confirmationPending}>
              {t('common.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={confirmationPending || pendingConfig === null}
              onClick={(event) => {
                event.preventDefault()
                if (!pendingConfig || confirmationPending) return
                setIsConfirming(true)
                void onSubmit(pendingConfig)
                  .then(() => setPendingConfig(null))
                  .catch(() => undefined)
                  .finally(() => setIsConfirming(false))
              }}
            >
              {confirmationPending && <Loader2Icon className="mr-2 size-4 animate-spin" />}
              {confirmationPending
                ? t('systemConfig.podBandwidth.saving')
                : t('systemConfig.podBandwidth.confirmSave')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
