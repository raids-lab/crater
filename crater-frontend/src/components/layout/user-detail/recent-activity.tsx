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
import { useTranslation } from 'react-i18next'

import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'

export default function RecentActivity() {
  const { t } = useTranslation()

  const activities = [
    { id: 1, action: t('recentActivity.uploadDataset'), date: '2023-03-01' },
    { id: 2, action: t('recentActivity.updateModel'), date: '2023-02-28' },
    { id: 3, action: t('recentActivity.publishPaper'), date: '2023-02-25' },
    { id: 4, action: t('recentActivity.joinWorkshop'), date: '2023-02-20' },
    { id: 5, action: t('recentActivity.shareRepo'), date: '2023-02-15' },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('recentActivity.title')}</CardTitle>
      </CardHeader>
      <CardContent>
        <ScrollArea className="h-[200px]">
          {activities.map((activity) => (
            <div key={activity.id} className="mb-4">
              <p className="font-semibold">{activity.action}</p>
              <p className="text-muted-foreground text-sm">{activity.date}</p>
            </div>
          ))}
        </ScrollArea>
      </CardContent>
    </Card>
  )
}
