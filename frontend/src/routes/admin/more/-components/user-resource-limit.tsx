import { useQuery } from '@tanstack/react-query'
import { ChevronDownIcon, PlusIcon, ShieldAlertIcon, TrashIcon } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from '@/components/ui/collapsible'
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

import { type ComboboxItem } from '@/components/form/combobox'

import { type QueueQuotaDraft } from '@/services/api/queue-quota'
import { type Resource, apiResourceList } from '@/services/api/resource'

type QueueQuotaResourceDraft = {
  cpu: string
  memory: string
  gpuCount: string
  gpuModel: string
  others: Record<string, string>
}

const queueQuotaGpuNoneValue = '__queue_quota_gpu_none__'
const quantityPattern =
  /^((\d+(\.\d+)?)|(\.\d+))(([eE][+-]?\d+)?)?(Ki|Mi|Gi|Ti|Pi|Ei|n|u|m|k|M|G|T|P|E)?$/
const positiveIntegerPattern = /^[1-9]\d*$/

const isLikelyGpuResource = (resource: string, gpuNames: Set<string>) =>
  gpuNames.has(resource) || resource.toLowerCase().includes('gpu')

const normalizeQuantity = (value: string) => value.trim()

const hasQuantity = (value: string) => normalizeQuantity(value) !== ''

const isValidQuantity = (value: string) =>
  !hasQuantity(value) || quantityPattern.test(normalizeQuantity(value))

const isValidGpuCount = (value: string) =>
  !hasQuantity(value) || positiveIntegerPattern.test(normalizeQuantity(value))

const parseQueueQuota = (
  quota: Record<string, string>,
  gpuNames: Set<string>
): QueueQuotaResourceDraft => {
  const draft: QueueQuotaResourceDraft = {
    cpu: '',
    memory: '',
    gpuCount: '',
    gpuModel: '',
    others: {},
  }

  for (const [resource, value] of Object.entries(quota)) {
    if (resource === 'cpu') {
      draft.cpu = normalizeQuantity(value)
      continue
    }
    if (resource === 'memory') {
      draft.memory = normalizeQuantity(value)
      continue
    }
    if (draft.gpuModel === '' && isLikelyGpuResource(resource, gpuNames)) {
      draft.gpuModel = resource
      draft.gpuCount = normalizeQuantity(value)
      continue
    }
    draft.others[resource] = normalizeQuantity(value)
  }

  return draft
}

const buildQueueQuota = (draft: QueueQuotaResourceDraft) => {
  const quota = { ...draft.others }

  if (hasQuantity(draft.cpu)) {
    quota.cpu = normalizeQuantity(draft.cpu)
  }

  if (hasQuantity(draft.memory)) {
    quota.memory = normalizeQuantity(draft.memory)
  }

  if (draft.gpuModel !== '' && hasQuantity(draft.gpuCount)) {
    quota[draft.gpuModel] = normalizeQuantity(draft.gpuCount)
  }

  return quota
}

const serializeQuota = (quota: Record<string, string>) =>
  JSON.stringify(
    Object.entries(quota).sort(([keyA], [keyB]) => {
      return keyA.localeCompare(keyB)
    })
  )

interface UserResourceLimitProps {
  configs: QueueQuotaDraft[]
  isPending: boolean
  onConfigsChange: (configs: QueueQuotaDraft[]) => void
  onCreate: (config: QueueQuotaDraft, index: number) => void
  onUpdate: (config: QueueQuotaDraft) => void
  onRemove: (config: QueueQuotaDraft, index: number) => void
}

export function UserResourceLimit({
  configs,
  isPending,
  onConfigsChange,
  onCreate,
  onUpdate,
  onRemove,
}: UserResourceLimitProps) {
  const { t } = useTranslation()
  const { data: gpuResources = [] } = useQuery({
    queryKey: ['resources', 'list', 'queue-quota-gpu'],
    queryFn: () => apiResourceList(true),
    select: (res) =>
      res.data
        .sort((a, b) => b.amountSingleMax - a.amountSingleMax)
        .filter((item) => item.amountSingleMax > 0)
        .map(
          (item) =>
            ({
              value: item.name,
              label: item.label,
              detail: item,
            }) as ComboboxItem<Resource>
        ),
  })

  return (
    <>
      <CardHeader>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldAlertIcon className="h-5 w-5 text-blue-500" />
            <CardTitle>{t('systemConfig.userResourceLimit.title')}</CardTitle>
          </div>
        </div>
        <CardDescription>{t('systemConfig.userResourceLimit.description')}</CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        {configs.map((config, index) => (
          <QueueLimitCard
            key={config.id ?? `new_${index}`}
            index={index}
            config={config}
            isPending={isPending}
            gpuResources={gpuResources}
            onQueueChange={(name) =>
              onConfigsChange(configs.map((item, i) => (i === index ? { ...item, name } : item)))
            }
            onEnabledChange={(enabled) =>
              onConfigsChange(configs.map((item, i) => (i === index ? { ...item, enabled } : item)))
            }
            onCandidateLimitChange={(prequeueCandidateSize) =>
              onConfigsChange(
                configs.map((item, i) => (i === index ? { ...item, prequeueCandidateSize } : item))
              )
            }
            onLimitsChange={(quota) =>
              onConfigsChange(configs.map((item, i) => (i === index ? { ...item, quota } : item)))
            }
            onSubmit={() => {
              if (config.id) {
                onUpdate(config)
                return
              }
              onCreate(config, index)
            }}
            onRemove={() => onRemove(config, index)}
          />
        ))}

        <Button
          type="button"
          variant="outline"
          size="sm"
          onClick={() =>
            onConfigsChange([
              ...configs,
              {
                name: '',
                savedName: undefined,
                enabled: false,
                prequeueCandidateSize: 10,
                quota: {},
              },
            ])
          }
          disabled={isPending}
        >
          <PlusIcon className="mr-1 h-4 w-4" />
          {t('systemConfig.userResourceLimit.addQueue')}
        </Button>
      </CardContent>
    </>
  )
}

function QueueLimitCard({
  index,
  config,
  isPending,
  gpuResources,
  onQueueChange,
  onEnabledChange,
  onCandidateLimitChange,
  onLimitsChange,
  onSubmit,
  onRemove,
}: {
  index: number
  config: QueueQuotaDraft
  isPending: boolean
  gpuResources: ComboboxItem<Resource>[]
  onQueueChange: (queue: string) => void
  onEnabledChange: (enabled: boolean) => void
  onCandidateLimitChange: (prequeueCandidateSize: number) => void
  onLimitsChange: (limits: Record<string, string>) => void
  onSubmit: () => void
  onRemove: () => void
}) {
  const { t } = useTranslation()
  const gpuNames = useMemo(() => new Set(gpuResources.map((item) => item.value)), [gpuResources])
  const [resourceDraft, setResourceDraft] = useState(() => parseQueueQuota(config.quota, gpuNames))
  const gpuItems = useMemo(() => {
    if (
      resourceDraft.gpuModel === '' ||
      gpuResources.some((item) => item.value === resourceDraft.gpuModel)
    ) {
      return gpuResources
    }
    return [
      {
        value: resourceDraft.gpuModel,
        label: resourceDraft.gpuModel,
      } as ComboboxItem<Resource>,
      ...gpuResources,
    ]
  }, [gpuResources, resourceDraft.gpuModel])

  useEffect(() => {
    if (serializeQuota(config.quota) === serializeQuota(buildQueueQuota(resourceDraft))) {
      return
    }
    setResourceDraft(parseQueueQuota(config.quota, gpuNames))
  }, [config.quota, gpuNames, resourceDraft])

  const hasGpuModelWithoutCount = resourceDraft.gpuModel !== '' && resourceDraft.gpuCount === ''
  const hasGpuCountWithoutModel = resourceDraft.gpuModel === '' && resourceDraft.gpuCount !== ''
  const hasInvalidCpuQuantity = !isValidQuantity(resourceDraft.cpu)
  const hasInvalidMemoryQuantity = !isValidQuantity(resourceDraft.memory)
  const hasInvalidGpuCount = !isValidGpuCount(resourceDraft.gpuCount)
  const isPersisted = typeof config.id === 'number'
  const canSubmit =
    config.name.trim() !== '' &&
    !hasGpuModelWithoutCount &&
    !hasGpuCountWithoutModel &&
    !hasInvalidCpuQuantity &&
    !hasInvalidMemoryQuantity &&
    !hasInvalidGpuCount

  const updateResourceDraft = (patch: Partial<QueueQuotaResourceDraft>) => {
    setResourceDraft((currentDraft) => {
      const nextDraft = {
        ...currentDraft,
        ...patch,
      }
      onLimitsChange(buildQueueQuota(nextDraft))
      return nextDraft
    })
  }

  const handleCandidateLimitInputChange = (value: string) => {
    const parsed = Number.parseInt(value, 10)
    if (Number.isNaN(parsed)) {
      onCandidateLimitChange(10)
      return
    }
    onCandidateLimitChange(Math.max(parsed, 1))
  }

  const title = config.id
    ? config.savedName?.trim() ||
      config.name.trim() ||
      t('systemConfig.userResourceLimit.unsavedTitle')
    : t('systemConfig.userResourceLimit.unsavedTitle')

  return (
    <Collapsible defaultOpen className="overflow-hidden rounded-lg border">
      <div className="flex flex-col gap-3 border-b px-4 py-3 md:flex-row md:items-center md:justify-between">
        <CollapsibleTrigger className="flex min-w-0 flex-1 items-center gap-2 text-left text-lg font-semibold">
          <ChevronDownIcon className="h-4 w-4 transition-transform [[data-state=closed]>&]:rotate-[-90deg]" />
          <span className="truncate">
            #{index + 1} - {title}
          </span>
        </CollapsibleTrigger>
        <div className="flex items-center gap-2">
          <Button
            type="button"
            variant="outline"
            size="sm"
            onClick={onSubmit}
            disabled={isPending || !canSubmit}
          >
            {isPersisted ? t('common.save') : t('common.create')}
          </Button>
          <Button
            type="button"
            variant="ghost"
            size="icon"
            className="h-8 w-8"
            onClick={onRemove}
            disabled={isPending}
          >
            <TrashIcon className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <CollapsibleContent className="grid gap-4 px-4 py-4 md:grid-cols-2">
        <div className="flex items-center justify-between gap-4 md:col-span-2">
          <Label className="text-base">{t('systemConfig.userResourceLimit.switchLabel')}</Label>
          <Switch checked={config.enabled} onCheckedChange={onEnabledChange} disabled={isPending} />
        </div>

        <div className="space-y-1.5">
          <Label>{t('systemConfig.userResourceLimit.queueLabel')}</Label>
          <Input
            value={config.name}
            onChange={(e) => onQueueChange(e.target.value)}
            placeholder={t('systemConfig.userResourceLimit.queuePlaceholder')}
            disabled={isPending}
          />
          <p className="text-muted-foreground text-[0.8rem]">
            {t('systemConfig.userResourceLimit.leafQueueNotice')}
          </p>
        </div>

        <div className="space-y-1.5">
          <Label>{t('systemConfig.userResourceLimit.prequeueCandidateLimitLabel')}</Label>
          <Input
            type="number"
            min={1}
            step={1}
            value={config.prequeueCandidateSize}
            onChange={(e) => handleCandidateLimitInputChange(e.target.value)}
            placeholder={t('systemConfig.userResourceLimit.prequeueCandidateLimitPlaceholder')}
            disabled={isPending}
          />
        </div>

        <div className="space-y-3 md:col-span-2">
          <Label>{t('systemConfig.userResourceLimit.limitsLabel')}</Label>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <div className="space-y-1.5">
              <Label>{t('resourceForm.cpuLabel')}</Label>
              <Input
                type="text"
                value={resourceDraft.cpu}
                onChange={(e) => updateResourceDraft({ cpu: e.target.value })}
                placeholder={t('systemConfig.userResourceLimit.cpuPlaceholder')}
                disabled={isPending}
              />
              <p className="text-muted-foreground text-[0.8rem]">
                {t('systemConfig.userResourceLimit.cpuFormatHint')}
              </p>
              {hasInvalidCpuQuantity && (
                <p className="text-destructive text-[0.8rem]">
                  {t('systemConfig.userResourceLimit.invalidCpuQuantity')}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>{t('resourceForm.memoryLabel')}</Label>
              <Input
                type="text"
                value={resourceDraft.memory}
                onChange={(e) => updateResourceDraft({ memory: e.target.value })}
                placeholder={t('systemConfig.userResourceLimit.memoryPlaceholder')}
                disabled={isPending}
              />
              <p className="text-muted-foreground text-[0.8rem]">
                {t('systemConfig.userResourceLimit.memoryFormatHint')}
              </p>
              {hasInvalidMemoryQuantity && (
                <p className="text-destructive text-[0.8rem]">
                  {t('systemConfig.userResourceLimit.invalidMemoryQuantity')}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label>{t('resourceForm.gpuModelLabel')}</Label>
              <Select
                value={
                  resourceDraft.gpuModel === '' ? queueQuotaGpuNoneValue : resourceDraft.gpuModel
                }
                onValueChange={(value) =>
                  updateResourceDraft({
                    gpuModel: value === queueQuotaGpuNoneValue ? '' : value,
                  })
                }
                disabled={isPending || gpuItems.length === 0}
              >
                <SelectTrigger className="w-full">
                  <SelectValue
                    placeholder={t('systemConfig.userResourceLimit.gpuModelPlaceholder')}
                  />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value={queueQuotaGpuNoneValue}>
                    {t('systemConfig.userResourceLimit.gpuModelNoneOption')}
                  </SelectItem>
                  {gpuItems.map((item) => (
                    <SelectItem key={item.value} value={item.value}>
                      {item.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('resourceForm.gpuCountLabel')}</Label>
              <Input
                type="number"
                step={1}
                value={resourceDraft.gpuCount}
                onChange={(e) => updateResourceDraft({ gpuCount: e.target.value })}
                placeholder={t('systemConfig.userResourceLimit.gpuCountPlaceholder')}
                disabled={isPending}
              />
            </div>
          </div>
          {(hasGpuModelWithoutCount || hasGpuCountWithoutModel) && (
            <p className="text-destructive text-[0.8rem]">
              {hasGpuCountWithoutModel
                ? t('systemConfig.userResourceLimit.gpuModelRequired')
                : t('systemConfig.userResourceLimit.gpuCountRequired')}
            </p>
          )}
          {hasInvalidGpuCount && (
            <p className="text-destructive text-[0.8rem]">
              {t('systemConfig.userResourceLimit.invalidGpuCount')}
            </p>
          )}
        </div>
      </CollapsibleContent>
    </Collapsible>
  )
}
