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
// i18n-processed-v1.1.0
// Modified code
import { type FC, useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import useFixedLayout from '@/hooks/useFixedLayout'

import { useTheme } from '@/utils/theme'

export const GrafanaIframe = ({ baseSrc }: { baseSrc: string }) => {
  const { t } = useTranslation()
  const { theme } = useTheme()

  // if baseURL does not have parameter, add ?, else if baseURL does not ends with &, add &
  const base = useMemo(() => {
    if (baseSrc.indexOf('?') === -1) {
      return `${baseSrc}?`
    } else if (baseSrc.endsWith('&') === false) {
      return `${baseSrc}&`
    } else {
      return baseSrc
    }
  }, [baseSrc])

  return (
    <iframe
      title={t('grafanaIframe.title')}
      src={`${base}theme=${theme}&kiosk&timezone=Asia%2FShanghai`}
      height={'100%'}
      width={'100%'}
    />
  )
}

const Monitor: FC<{ baseSrc: string }> = ({ baseSrc }: { baseSrc: string }) => {
  useFixedLayout()
  return (
    <div className="h-[calc(100vh_-_80px)] w-full">
      <GrafanaIframe baseSrc={baseSrc || ''} />
    </div>
  )
}

export default Monitor
