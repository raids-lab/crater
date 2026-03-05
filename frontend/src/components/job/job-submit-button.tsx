import { useQuery } from '@tanstack/react-query'
import { CirclePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import LoadableButton from '@/components/button/loadable-button'

import { IResourceLimitDetail, apiCheckResourceLimit } from '@/services/api/system-config'

import { REFETCH_INTERVAL } from '@/lib/constants'

interface JobSubmitButtonProps {
  isLoading: boolean
  isLoadingText?: string
  requestedResources?: Record<string, string>
}

export function JobSubmitButton({
  isLoading,
  isLoadingText,
  requestedResources,
}: JobSubmitButtonProps) {
  const { t } = useTranslation()

  const { data: limitCheck } = useQuery({
    queryKey: ['context', 'resource-limit-check', requestedResources],
    queryFn: () => apiCheckResourceLimit(requestedResources).then((res) => res.data),
    refetchInterval: REFETCH_INTERVAL,
  })

  const isExceeded = limitCheck?.enabled && limitCheck?.exceeded
  const exceededDetails = limitCheck?.details?.filter((d: IResourceLimitDetail) => d.exceeded) ?? []

  const tooltipContent = isExceeded
    ? exceededDetails
        .map(
          (d: IResourceLimitDetail) =>
            `${d.resource}: ${t('resourceLimit.used')} ${d.used} / ${t('resourceLimit.limit')} ${d.limit}`
        )
        .join('\n')
    : ''

  const button = (
    <LoadableButton
      isLoading={isLoading}
      isLoadingText={isLoadingText ?? t('resourceLimit.submitting')}
      type="submit"
      disabled={isExceeded}
    >
      <CirclePlus className="size-4" />
      {t('resourceLimit.submitJob')}
    </LoadableButton>
  )

  if (!isExceeded) return button

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span tabIndex={0} className="inline-flex">
          {button}
        </span>
      </TooltipTrigger>
      <TooltipContent side="bottom" className="max-w-xs whitespace-pre-line">
        <p className="font-semibold">{t('resourceLimit.exceededTitle')}</p>
        <p className="mt-1 text-xs">{tooltipContent}</p>
      </TooltipContent>
    </Tooltip>
  )
}
