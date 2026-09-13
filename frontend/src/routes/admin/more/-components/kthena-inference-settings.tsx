import { CheckCircle2Icon, Loader2Icon, RocketIcon, UnplugIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

interface KthenaInferenceSettingsProps {
  enabled: boolean
  isPending: boolean
  onToggle: (checked: boolean) => void
}

export function KthenaInferenceSettings({
  enabled,
  isPending,
  onToggle,
}: KthenaInferenceSettingsProps) {
  const { t } = useTranslation()

  return (
    <>
      <CardHeader>
        <div className="flex items-center gap-2">
          <RocketIcon
            className={enabled ? 'h-5 w-5 text-green-500' : 'text-muted-foreground h-5 w-5'}
          />
          <CardTitle>
            {t('systemConfig.kthenaInference.title', { defaultValue: '模型部署' })}
          </CardTitle>
        </div>
        <CardDescription>
          {t('systemConfig.kthenaInference.description', {
            defaultValue: '使用 Kthena 提供在线模型服务的部署与调用能力。',
          })}
        </CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
          <div className="space-y-0.5">
            <Label className="text-base">
              {t('systemConfig.kthenaInference.switchLabel', { defaultValue: '启用模型部署' })}
            </Label>
            <p className="text-muted-foreground text-[0.8rem]">
              {t('systemConfig.kthenaInference.switchDescription', {
                defaultValue: '开启后，用户可以创建、管理和调用在线模型服务。',
              })}
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
            <p>
              {t('systemConfig.kthenaInference.disabledWarning', {
                defaultValue: '功能关闭后将隐藏入口，且直接访问模型部署页面也会被拒绝。',
              })}
            </p>
          </div>
        )}
        {enabled && (
          <div className="mt-4 flex items-start gap-2 rounded-md bg-green-50 p-3 text-xs text-green-700 dark:bg-green-950/30 dark:text-green-400">
            <CheckCircle2Icon className="mt-0.5 h-3.5 w-3.5" />
            <p>
              {t('systemConfig.kthenaInference.activeNotice', {
                defaultValue: '模型部署已开启，用户可以使用 Kthena 创建在线模型服务。',
              })}
            </p>
          </div>
        )}
      </CardContent>
    </>
  )
}
