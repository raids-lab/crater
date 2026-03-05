import {
  CheckCircle2Icon,
  ChevronDownIcon,
  Loader2Icon,
  PlusIcon,
  ShieldAlertIcon,
  TrashIcon,
  UnplugIcon,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'

import { IQueueResourceLimit } from '@/services/api/system-config'

interface UserResourceLimitProps {
  enabled: boolean
  configs: IQueueResourceLimit[]
  isPending: boolean
  onToggle: (checked: boolean) => void
  onConfigsChange: (configs: IQueueResourceLimit[]) => void
  onSave: () => void
}

export function UserResourceLimit({
  enabled,
  configs,
  isPending,
  onToggle,
  onConfigsChange,
  onSave,
}: UserResourceLimitProps) {
  const { t } = useTranslation()

  const handleAddQueue = () => {
    onConfigsChange([...configs, { queue: '', limits: {} }])
  }

  const handleRemoveQueue = (index: number) => {
    onConfigsChange(configs.filter((_, i) => i !== index))
  }

  const handleQueueChange = (index: number, queue: string) => {
    const next = configs.map((c, i) => (i === index ? { ...c, queue } : c))
    onConfigsChange(next)
  }

  const handleLimitsChange = (index: number, limits: Record<string, string>) => {
    const next = configs.map((c, i) => (i === index ? { ...c, limits } : c))
    onConfigsChange(next)
  }

  return (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldAlertIcon
              className={enabled ? 'h-5 w-5 text-blue-500' : 'text-muted-foreground h-5 w-5'}
            />
            <CardTitle>{t('systemConfig.userResourceLimit.title')}</CardTitle>
          </div>
        </div>
        <CardDescription>{t('systemConfig.userResourceLimit.description')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="flex items-center justify-between rounded-lg border p-4 shadow-sm">
          <div className="space-y-0.5">
            <Label className="text-base">{t('systemConfig.userResourceLimit.switchLabel')}</Label>
            <p className="text-muted-foreground text-[0.8rem]">
              {t('systemConfig.userResourceLimit.switchDescription')}
            </p>
          </div>
          <div className="flex items-center gap-2">
            {isPending && <Loader2Icon className="text-muted-foreground h-4 w-4 animate-spin" />}
            <Switch checked={enabled} onCheckedChange={onToggle} disabled={isPending} />
          </div>
        </div>

        {enabled && (
          <>
            {configs.map((cfg, idx) => (
              <QueueLimitCard
                key={idx}
                index={idx}
                config={cfg}
                onQueueChange={(q) => handleQueueChange(idx, q)}
                onLimitsChange={(l) => handleLimitsChange(idx, l)}
                onRemove={() => handleRemoveQueue(idx)}
              />
            ))}

            <Button type="button" variant="outline" size="sm" onClick={handleAddQueue}>
              <PlusIcon className="mr-1 h-4 w-4" />
              {t('systemConfig.userResourceLimit.addQueue')}
            </Button>
          </>
        )}

        {!enabled && (
          <div className="flex items-start gap-2 rounded-md bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-400">
            <UnplugIcon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.userResourceLimit.disabledWarning')}</p>
          </div>
        )}
        {enabled && (
          <div className="flex items-start gap-2 rounded-md bg-green-50 p-3 text-xs text-green-700 dark:bg-green-950/30 dark:text-green-400">
            <CheckCircle2Icon className="mt-0.5 h-3.5 w-3.5" />
            <p>{t('systemConfig.userResourceLimit.activeNotice')}</p>
          </div>
        )}
      </CardContent>
      {enabled && (
        <CardFooter className="bg-muted/10 px-6 py-4">
          <Button onClick={onSave} disabled={isPending}>
            {isPending ? t('common.saving') : t('common.save')}
          </Button>
        </CardFooter>
      )}
    </>
  )
}

function QueueLimitCard({
  index,
  config,
  onQueueChange,
  onLimitsChange,
  onRemove,
}: {
  index: number
  config: IQueueResourceLimit
  onQueueChange: (queue: string) => void
  onLimitsChange: (limits: Record<string, string>) => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const limitEntries = Object.entries(config.limits)
  const hasEmptyKey = limitEntries.some(([key]) => key === '')

  const handleAddLimit = () => {
    onLimitsChange({ ...config.limits, '': '' })
  }

  const handleRemoveLimit = (key: string) => {
    const next = { ...config.limits }
    delete next[key]
    onLimitsChange(next)
  }

  const handleLimitKeyChange = (oldKey: string, newKey: string) => {
    const entries = Object.entries(config.limits)
    const next: Record<string, string> = {}
    for (const [k, v] of entries) {
      next[k === oldKey ? newKey : k] = v
    }
    onLimitsChange(next)
  }

  const handleLimitValueChange = (key: string, value: string) => {
    onLimitsChange({ ...config.limits, [key]: value })
  }

  const title = config.queue || t('systemConfig.userResourceLimit.queuePlaceholder')

  return (
    <Collapsible defaultOpen className="rounded-lg border">
      <div className="flex items-center justify-between px-4 py-3">
        <CollapsibleTrigger className="flex flex-1 items-center gap-2 text-sm font-medium">
          <ChevronDownIcon className="h-4 w-4 transition-transform [[data-state=closed]>&]:rotate-[-90deg]" />
          <span>
            #{index + 1} — {title}
          </span>
        </CollapsibleTrigger>
        <Button type="button" variant="ghost" size="icon" className="h-8 w-8" onClick={onRemove}>
          <TrashIcon className="h-4 w-4" />
        </Button>
      </div>

      <CollapsibleContent className="space-y-3 px-4 pb-4">
        <div className="space-y-1.5">
          <Label>{t('systemConfig.userResourceLimit.queueLabel')}</Label>
          <Input
            value={config.queue}
            onChange={(e) => onQueueChange(e.target.value)}
            placeholder={t('systemConfig.userResourceLimit.queuePlaceholder')}
          />
        </div>

        <div className="space-y-1.5">
          <Label>{t('systemConfig.userResourceLimit.limitsLabel')}</Label>
          <p className="text-muted-foreground text-[0.8rem]">
            {t('systemConfig.userResourceLimit.limitsDescription')}
          </p>
          <div className="space-y-2">
            {limitEntries.map(([key, value], i) => (
              <div key={key || `_new_${i}`} className="flex items-center gap-2">
                <Input
                  className="flex-1"
                  value={key}
                  onChange={(e) => handleLimitKeyChange(key, e.target.value)}
                  placeholder={t('systemConfig.userResourceLimit.resourceNamePlaceholder')}
                />
                <Input
                  className="flex-1"
                  value={value}
                  onChange={(e) => handleLimitValueChange(key, e.target.value)}
                  placeholder={t('systemConfig.userResourceLimit.resourceValuePlaceholder')}
                />
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  onClick={() => handleRemoveLimit(key)}
                >
                  <TrashIcon className="h-4 w-4" />
                </Button>
              </div>
            ))}
          </div>
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={handleAddLimit}
            disabled={hasEmptyKey}
          >
            <PlusIcon className="mr-1 h-4 w-4" />
            {t('systemConfig.userResourceLimit.addLimit')}
          </Button>
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
