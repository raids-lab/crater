import { CheckCircle2Icon, Loader2Icon, RocketIcon, UnplugIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { useKthenaInferenceSettings } from './use-kthena-inference-settings'

export function KthenaInferenceSettings() {
  const { t } = useTranslation()
  const { enabled, isLoading, isError, isFetching, isPending, refetch, onToggle } =
    useKthenaInferenceSettings()
  const hasStatus = typeof enabled === 'boolean' && !isLoading && !isError
  const controlsDisabled = !hasStatus || isPending

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <RocketIcon
            className={
              hasStatus && enabled ? 'h-5 w-5 text-green-500' : 'text-muted-foreground h-5 w-5'
            }
          />
          <CardTitle>{t('systemConfig.kthenaInference.title')}</CardTitle>
        </div>
        <CardDescription>{t('systemConfig.kthenaInference.description')}</CardDescription>
      </CardHeader>
      <CardContent aria-busy={isLoading || isPending}>
        {isLoading && (
          <div className="text-muted-foreground mb-4 flex items-center gap-2 text-sm" role="status">
            <Loader2Icon className="size-4 animate-spin" />
            {t('common.loading')}
          </div>
        )}
        {isError && (
          <div
            className="border-destructive/50 bg-destructive/5 text-destructive mb-4 flex items-center justify-between gap-3 rounded-md border p-3 text-sm"
            role="alert"
          >
            <span>{t('common.error')}</span>
            <Button
              type="button"
              variant="outline"
              size="sm"
              disabled={isFetching}
              onClick={() => void refetch()}
            >
              {t('common.refresh')}
            </Button>
          </div>
        )}
        <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
          <div className="space-y-0.5">
            <Label htmlFor="kthena-inference-enabled" className="text-base">
              {t('systemConfig.kthenaInference.switchLabel')}
            </Label>
            <p id="kthena-inference-description" className="text-muted-foreground text-[0.8rem]">
              {t('systemConfig.kthenaInference.switchDescription')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {isPending && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
            <Switch
              id="kthena-inference-enabled"
              aria-describedby="kthena-inference-description"
              checked={enabled ?? false}
              onCheckedChange={onToggle}
              disabled={controlsDisabled}
            />
          </div>
        </div>

        {hasStatus && !enabled && (
          <div className="mt-4 flex items-start gap-2 rounded-md bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
            <UnplugIcon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.kthenaInference.disabledWarning')}</p>
          </div>
        )}
        {hasStatus && enabled && (
          <div className="mt-4 flex items-start gap-2 rounded-md bg-green-50 p-3 text-xs text-green-700 dark:bg-green-950/30 dark:text-green-400">
            <CheckCircle2Icon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.kthenaInference.activeNotice')}</p>
          </div>
        )}
      </CardContent>
    </>
  )
}
