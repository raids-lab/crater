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
import { useAtomValue } from 'jotai'

import { atomUserContext } from '@/utils/store'
import { configGrafanaOverviewAtom } from '@/utils/store/config'

import FixedGrafanaIframe from '../layout/embed/fixed-grafana-iframe'

type NvidiaOverviewProps = {
  view: 'user' | 'admin'
}

const NvidiaOverview = ({ view }: NvidiaOverviewProps) => {
  const grafanaOverview = useAtomValue(configGrafanaOverviewAtom)
  const userContext = useAtomValue(atomUserContext)

  const developmentDashboard =
    import.meta.env.MODE === 'development'
      ? import.meta.env.VITE_DEVELOPMENT_GRAFANA_OVERVIEW_MAIN
      : undefined
  const dashboard = developmentDashboard || grafanaOverview.main

  const variables = new URLSearchParams({ 'var-view': view })
  if (view === 'user') {
    variables.set('var-account', userContext?.queue ?? 'default')
  }

  const separator = dashboard.includes('?')
    ? dashboard.endsWith('?') || dashboard.endsWith('&')
      ? ''
      : '&'
    : '?'

  return <FixedGrafanaIframe baseSrc={`${dashboard}${separator}${variables.toString()}`} />
}

export default NvidiaOverview
