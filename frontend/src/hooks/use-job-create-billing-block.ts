import { HTTPError } from 'ky'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { handleApiErrorByCode, markApiErrorHandled } from '@/services/client'
import { ERROR_USER_BANNED } from '@/services/error_code'
import { LEGACY_ERROR_BUSINESS_LOGIC_ERROR } from '@/services/error_code_legacy'
import { IErrorResponse } from '@/services/types'

const BILLING_BLOCK_PREFIX = 'billing precheck blocked:'

export function isJobCreateBillingBlocked(error: unknown) {
  if (!(error instanceof HTTPError)) {
    return false
  }
  const errorWithData = error as HTTPError & { data?: IErrorResponse }
  return (
    errorWithData.data?.code === LEGACY_ERROR_BUSINESS_LOGIC_ERROR &&
    Boolean(errorWithData.data?.msg?.startsWith(BILLING_BLOCK_PREFIX))
  )
}

export function useJobCreateBillingBlockDialog() {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)

  return {
    billingBlockDialogOpen: open,
    setBillingBlockDialogOpen: setOpen,
    handleJobCreateError: (error: unknown) => {
      if (isJobCreateBillingBlocked(error)) {
        markApiErrorHandled(error)
        setOpen(true)
        return true
      }
      return handleApiErrorByCode(error, {
        [ERROR_USER_BANNED]: () =>
          toast.error(t('userBan.errors.jobSubmissionTitle'), {
            description: t('userBan.errors.jobSubmissionDescription'),
          }),
      })
    },
  }
}
