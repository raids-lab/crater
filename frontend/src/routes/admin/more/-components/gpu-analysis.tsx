import { CheckCircle2Icon, GpuIcon, Loader2Icon, UnplugIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

interface GpuAnalysisProps {
  enabled: boolean
  isPending: boolean
  onToggle: (checked: boolean) => void
}

export function GpuAnalysis({ enabled, isPending, onToggle }: GpuAnalysisProps) {
  const { t } = useTranslation()

  return (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <GpuIcon
              className={enabled ? 'h-5 w-5 text-green-500' : 'text-muted-foreground h-5 w-5'}
            />
            <CardTitle>{t('systemConfig.gpuAnalysis.title')}</CardTitle>
          </div>
        </div>
        <CardDescription>{t('systemConfig.gpuAnalysis.description')}</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
          <div className="space-y-0.5">
            <Label className="text-base">{t('systemConfig.gpuAnalysis.switchLabel')}</Label>
            <p className="text-muted-foreground text-[0.8rem]">
              {t('systemConfig.gpuAnalysis.switchDescription')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {isPending && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
            <Switch checked={enabled} onCheckedChange={onToggle} disabled={isPending} />
          </div>
        </div>

        {!enabled && (
          <div className="mt-4 flex items-start gap-2 rounded-md bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
            <UnplugIcon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.gpuAnalysis.disabledWarning')}</p>
          </div>
        )}
        {enabled && (
          <div className="mt-4 flex items-start gap-2 rounded-md bg-green-50 p-3 text-xs text-green-700 dark:bg-green-950/30 dark:text-green-400">
            <CheckCircle2Icon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.gpuAnalysis.activeNotice')}</p>
          </div>
        )}
      </CardContent>
    </>
  )
}
