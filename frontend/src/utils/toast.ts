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
import { isAxiosError } from 'axios'
import { toast } from 'sonner'

import { IErrorResponse } from '@/services/types'

export const showErrorToast = (error: unknown) => {
  if (isAxiosError(error)) {
    if (error.response?.data) {
      try {
        const errorResponse = error.response.data as IErrorResponse
        if (errorResponse.msg) {
          toast.error(`${errorResponse.msg}`)
        } else {
          toast.error(`${error.message}`)
        }
      } catch {
        toast.error(`${error.message}`)
      }
    } else {
      toast.error(`${error.message}`)
    }
  } else if (typeof error === 'string') {
    toast.error(error)
  } else {
    toast.error((error as Error).message)
  }
}
