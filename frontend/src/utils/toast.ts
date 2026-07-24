/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import i18n from '@/i18n'
import { isAxiosError } from 'axios'
import { toast } from 'sonner'

import { ERROR_USER_BANNED } from '@/services/error_code'
import { IErrorResponse } from '@/services/types'

const isToastHandled = (error: unknown) =>
  Boolean(
    error && typeof error === 'object' && (error as { isHandledByBiz?: boolean }).isHandledByBiz
  )

const showUserBanToast = () => {
  toast.error(i18n.t('userBan.errors.operationBlockedTitle'), {
    description: i18n.t('userBan.errors.operationBlockedDescription'),
  })
}

const markToastHandled = (error: unknown) => {
  if (!error || typeof error !== 'object') {
    return
  }

  const handledError = error as {
    isHandledByBiz?: boolean
    fallbackLogTimer?: ReturnType<typeof setTimeout>
  }
  handledError.isHandledByBiz = true

  if (handledError.fallbackLogTimer) {
    clearTimeout(handledError.fallbackLogTimer)
    handledError.fallbackLogTimer = undefined
  }
}

export const showErrorToast = (error: unknown) => {
  if (isToastHandled(error)) {
    return
  }
  markToastHandled(error)

  // 1. Handle AxiosError (for backward compatibility with axios-based code)
  if (isAxiosError(error)) {
    if (error.response?.data) {
      try {
        const errorResponse = error.response.data as IErrorResponse
        const httpStatus = error.response.status
        const businessCode = errorResponse.code
        const errorMsg = errorResponse.msg || error.message

        if (businessCode === ERROR_USER_BANNED) {
          showUserBanToast()
          return
        }

        // Build complete error message: HTTP status code + business error code + error message
        const fullErrorMessage = `[HTTP ${httpStatus}] [Code ${businessCode}] ${errorMsg}`
        toast.error(fullErrorMessage)
      } catch {
        toast.error(`${error.message}`)
      }
    } else {
      toast.error(`${error.message}`)
    }
    return
  }

  // 2. Handle string (display directly)
  if (typeof error === 'string') {
    toast.error(error)
    return
  }

  // 3. Handle HTTPError (from ky library)
  // apiRequest has already mounted errorResponse to error.data and HTTP status code to error.httpStatus
  if (error && typeof error === 'object' && 'data' in error) {
    const errorWithData = error as {
      data?: IErrorResponse
      httpStatus?: number
      response?: { status: number }
    }

    // Try to get HTTP status code from multiple locations
    const httpStatus = errorWithData.httpStatus || errorWithData.response?.status

    if (errorWithData.data) {
      const businessCode = errorWithData.data.code
      const errorMsg = errorWithData.data.msg || 'Request failed'

      if (businessCode === ERROR_USER_BANNED) {
        showUserBanToast()
        return
      }

      // Build complete error message: HTTP status code + business error code + error message
      if (httpStatus !== undefined) {
        const fullErrorMessage = `[HTTP ${httpStatus}] [Code ${businessCode}] ${errorMsg}`
        toast.error(fullErrorMessage)
      } else {
        // If HTTP status code is not available, only show business error code and message
        const fullErrorMessage = `[Code ${businessCode}] ${errorMsg}`
        toast.error(fullErrorMessage)
      }
      return
    }
  }

  // 4. Fallback: display the error object's message property
  const errorMessage = (error as Error)?.message || 'Request failed, please try again later'
  toast.error(errorMessage)
}
