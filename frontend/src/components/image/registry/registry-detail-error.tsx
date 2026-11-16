// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import { ErrorComponentProps, useRouter } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'

import NotFound from '@/components/placeholder/not-found'

import { IErrorResponse } from '@/services/types'

export default function RegistryDetailError({ error }: ErrorComponentProps) {
  const { t } = useTranslation()
  const router = useRouter()
  const apiError = error as Error & {
    data?: IErrorResponse
    httpStatus?: number
    response?: { status: number }
  }
  const status = apiError.httpStatus ?? apiError.response?.status

  if (status === 404) {
    return <NotFound />
  }

  return (
    <div className="flex flex-col items-center gap-4 p-8" role="alert">
      <h2 className="text-lg font-semibold">{t('imageBuild.detail.loadError')}</h2>
      <p>{apiError.data?.msg || error.message}</p>
      <p className="text-muted-foreground text-sm">
        {status !== undefined && `HTTP ${status}`}
        {apiError.data?.code !== undefined && ` / Code ${apiError.data.code}`}
      </p>
      <Button onClick={() => void router.invalidate()}>{t('imageBuild.detail.retry')}</Button>
    </div>
  )
}
