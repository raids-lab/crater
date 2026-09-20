import { CircleHelpIcon, Layers3Icon, Loader2Icon, SaveIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import SimpleTooltip from '@/components/label/simple-tooltip'

interface PrequeueSettingsProps {
  backfillEnabled: boolean
  queueQuotaEnabled: boolean
  isPending: boolean
  waitingToleranceSeconds: string
  activateTickerIntervalSeconds: string
  maxTotalActivationsPerRound: string
  prequeueCandidateSize: string
  onBackfillEnabledChange: (enabled: boolean) => void
  onQueueQuotaEnabledChange: (enabled: boolean) => void
  onWaitingToleranceSecondsChange: (value: string) => void
  onActivateTickerIntervalSecondsChange: (value: string) => void
  onMaxTotalActivationsPerRoundChange: (value: string) => void
  onPrequeueCandidateSizeChange: (value: string) => void
  onSubmit: () => void
}

export function PrequeueSettings({
  backfillEnabled,
  queueQuotaEnabled,
  isPending,
  waitingToleranceSeconds,
  activateTickerIntervalSeconds,
  maxTotalActivationsPerRound,
  prequeueCandidateSize,
  onBackfillEnabledChange,
  onQueueQuotaEnabledChange,
  onWaitingToleranceSecondsChange,
  onActivateTickerIntervalSecondsChange,
  onMaxTotalActivationsPerRoundChange,
  onPrequeueCandidateSizeChange,
  onSubmit,
}: PrequeueSettingsProps) {
  const { t } = useTranslation()
  const hasEnabledFeature = backfillEnabled || queueQuotaEnabled

  return (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Layers3Icon
              className={
                hasEnabledFeature ? 'h-5 w-5 text-blue-500' : 'text-muted-foreground h-5 w-5'
              }
            />
            <CardTitle>{t('systemConfig.prequeue.title')}</CardTitle>
          </div>
        </div>
        <CardDescription>{t('systemConfig.prequeue.description')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2">
          <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
            <div>
              <Label htmlFor="prequeue-backfill-enabled" className="text-base">
                {t('systemConfig.prequeue.backfillSwitchLabel')}
              </Label>
              <p className="text-muted-foreground mt-1 text-sm">
                {t('systemConfig.prequeue.backfillSwitchDescription')}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {isPending && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
              <Switch
                id="prequeue-backfill-enabled"
                checked={backfillEnabled}
                onCheckedChange={onBackfillEnabledChange}
                disabled={isPending}
              />
            </div>
          </div>

          <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
            <div>
              <Label htmlFor="prequeue-quota-enabled" className="text-base">
                {t('systemConfig.prequeue.queueQuotaSwitchLabel')}
              </Label>
              <p className="text-muted-foreground mt-1 text-sm">
                {t('systemConfig.prequeue.queueQuotaSwitchDescription')}
              </p>
            </div>
            <div className="flex items-center gap-2">
              {isPending && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
              <Switch
                id="prequeue-quota-enabled"
                checked={queueQuotaEnabled}
                onCheckedChange={onQueueQuotaEnabledChange}
                disabled={isPending}
              />
            </div>
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="prequeue-waiting-tolerance">
            {t('systemConfig.prequeue.waitingToleranceLabel')}
            <SimpleTooltip
              tooltip={t('systemConfig.prequeue.waitingToleranceDescription')}
              ariaLabel={t('systemConfig.prequeue.waitingToleranceLabel')}
            >
              <CircleHelpIcon className="text-muted-foreground size-4" />
            </SimpleTooltip>
          </Label>
          <Input
            id="prequeue-waiting-tolerance"
            type="number"
            min={1}
            value={waitingToleranceSeconds}
            onChange={(event) => onWaitingToleranceSecondsChange(event.target.value)}
            disabled={isPending}
            placeholder={t('systemConfig.prequeue.waitingTolerancePlaceholder')}
          />
        </div>

        <details className="rounded-lg border p-4">
          <summary className="cursor-pointer text-sm font-medium">
            {t('systemConfig.prequeue.advanced')}
          </summary>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <Label htmlFor="prequeue-activate-ticker">
                {t('systemConfig.prequeue.activateTickerLabel')}
              </Label>
              <p className="text-muted-foreground text-xs">
                {t('systemConfig.prequeue.activateTickerDescription')}
              </p>
              <Input
                id="prequeue-activate-ticker"
                type="number"
                min={1}
                value={activateTickerIntervalSeconds}
                onChange={(event) => onActivateTickerIntervalSecondsChange(event.target.value)}
                disabled={isPending}
                placeholder={t('systemConfig.prequeue.positiveIntegerPlaceholder')}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="prequeue-max-activations">
                {t('systemConfig.prequeue.maxActivationsLabel')}
              </Label>
              <p className="text-muted-foreground text-xs">
                {t('systemConfig.prequeue.maxActivationsDescription')}
              </p>
              <Input
                id="prequeue-max-activations"
                type="number"
                min={1}
                value={maxTotalActivationsPerRound}
                onChange={(event) => onMaxTotalActivationsPerRoundChange(event.target.value)}
                disabled={isPending}
                placeholder={t('systemConfig.prequeue.positiveIntegerPlaceholder')}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="prequeue-candidate-size">
                {t('systemConfig.prequeue.candidateSizeLabel')}
              </Label>
              <p className="text-muted-foreground text-xs">
                {t('systemConfig.prequeue.candidateSizeDescription')}
              </p>
              <Input
                id="prequeue-candidate-size"
                type="number"
                min={1}
                value={prequeueCandidateSize}
                onChange={(event) => onPrequeueCandidateSizeChange(event.target.value)}
                disabled={isPending}
                placeholder={t('systemConfig.prequeue.positiveIntegerPlaceholder')}
              />
            </div>
          </div>
        </details>
      </CardContent>
      <CardFooter className="bg-muted/10 justify-end border-t px-6 py-4">
        <Button type="button" onClick={onSubmit} disabled={isPending}>
          <SaveIcon />
          {t('systemConfig.prequeue.save')}
        </Button>
      </CardFooter>
    </>
  )
}
