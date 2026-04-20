import { HTTPError } from 'ky'
import { useState } from 'react'

import { ERROR_BUSINESS_LOGIC_ERROR } from '@/services/error_code'
import { IErrorResponse } from '@/services/types'

const BILLING_BLOCK_PREFIX = 'billing precheck blocked:'

export function isJobCreateBillingBlocked(error: unknown) {
  if (!(error instanceof HTTPError)) {
    return false
  }
  const errorWithData = error as HTTPError & { data?: IErrorResponse }
  return (
    errorWithData.data?.code === ERROR_BUSINESS_LOGIC_ERROR &&
    Boolean(errorWithData.data?.msg?.startsWith(BILLING_BLOCK_PREFIX))
  )
}

export function useJobCreateBillingBlockDialog() {
  const [open, setOpen] = useState(false)

  return {
    billingBlockDialogOpen: open,
    setBillingBlockDialogOpen: setOpen,
    handleJobCreateError: (error: unknown) => {
      if (isJobCreateBillingBlocked(error)) {
        setOpen(true)
        return true
      }
      return false
    },
  }
}
