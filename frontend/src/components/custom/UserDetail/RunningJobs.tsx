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
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'

export default function RunningJobs() {
  const { t } = useTranslation()

  const jobs = [
    {
      id: 1,
      name: t('jobs.dataPreprocessing'),
      status: t('statuses.running'),
      progress: '75%',
    },
    {
      id: 2,
      name: t('jobs.modelTraining'),
      status: t('statuses.waiting'),
      progress: '0%',
    },
    {
      id: 3,
      name: t('jobs.resultAnalysis'),
      status: t('statuses.queued'),
      progress: '0%',
    },
  ]

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t('runningJobs.title')}</CardTitle>
      </CardHeader>
      <CardContent>
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t('runningJobs.headers.jobName')}</TableHead>
              <TableHead>{t('runningJobs.headers.status')}</TableHead>
              <TableHead>{t('runningJobs.headers.progress')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {jobs.map((job) => (
              <TableRow key={job.id}>
                <TableCell>{job.name}</TableCell>
                <TableCell>{job.status}</TableCell>
                <TableCell>{job.progress}</TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </CardContent>
    </Card>
  )
}
