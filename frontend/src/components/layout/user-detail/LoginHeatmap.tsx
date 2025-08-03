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
import { ResponsiveTimeRange } from '@nivo/calendar'
import { format, subDays } from 'date-fns'
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'

import useNivoTheme from '@/hooks/useNivoTheme'

const generateRandomData = (days: number) => {
  const data = []
  const today = new Date()

  for (let i = 0; i < days; i++) {
    const date = subDays(today, i)
    const formattedDate = format(date, 'yyyy-MM-dd')
    const value = Math.floor(Math.random() * 10) - 7 // 生成0到5之间的随机值
    if (value < 0) {
      continue
    }
    data.push({ day: formattedDate, value })
  }

  return data
}

export default function LoginHeatmap() {
  const { t } = useTranslation()
  const data = generateRandomData(365)
  const { nivoTheme, theme } = useNivoTheme()
  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('loginHeatmap.title')}</CardTitle>
      </CardHeader>
      <CardContent>
        <div style={{ height: '200px' }}>
          <ResponsiveTimeRange
            data={data}
            from="2023-12-13"
            to="2024-12-13"
            emptyColor={theme === 'dark' ? '#1f283b' : '#eeeeee'}
            colors={['#61cdbb', '#97e3d5', '#e8c1a0', '#f47560']}
            margin={{ top: 40, right: 20, bottom: 20, left: 20 }}
            dayBorderWidth={2}
            dayBorderColor={theme === 'dark' ? '#10172a' : '#ffffff'}
            firstWeekday="monday"
            theme={nivoTheme}
          />
        </div>
      </CardContent>
    </Card>
  )
}
