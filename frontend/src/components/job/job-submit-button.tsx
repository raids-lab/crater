import { CirclePlus } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import LoadableButton from '@/components/button/loadable-button'

interface JobSubmitButtonProps {
  isLoading: boolean
  isLoadingText?: string
}

export function JobSubmitButton({ isLoading, isLoadingText }: JobSubmitButtonProps) {
  const { t } = useTranslation()

  return (
    <LoadableButton
      isLoading={isLoading}
      isLoadingText={isLoadingText ?? t('resourceLimit.submitting')}
      type="submit"
    >
      <CirclePlus className="size-4" />
      {t('resourceLimit.submitJob')}
    </LoadableButton>
  )
}
