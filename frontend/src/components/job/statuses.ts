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
import { t } from 'i18next'

import { jobPhases } from '@/components/badge/job-phase-badge'
import { jobTypes } from '@/components/badge/job-type-badge'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

export const getHeader = (key: string): string => {
  switch (key) {
    case 'id':
      return t('jobs.headers.id')
    case 'name':
      return t('jobs.headers.jobName')
    case 'jobType':
      return t('jobs.headers.jobType')
    case 'queue':
      return t('jobs.headers.queue')
    case 'owner':
      return t('jobs.headers.owner')
    case 'status':
      return t('jobs.headers.status')
    case 'nodes':
      return t('jobs.headers.nodes')
    case 'resources':
      return t('jobs.headers.resources')
    case 'priority':
      return t('jobs.headers.priority')
    case 'profileStatus':
      return t('jobs.headers.profileStatus')
    case 'createdAt':
      return t('jobs.headers.createdAt')
    case 'startedAt':
      return t('jobs.headers.startedAt')
    case 'completedAt':
      return t('jobs.headers.completedAt')
    default:
      return key
  }
}

export const jobToolbarConfig: DataTableToolbarConfig = {
  filterInput: {
    placeholder: t('jobs.toolbar.searchPlaceholder'),
    key: 'name',
  },
  filterOptions: [
    {
      key: 'jobType',
      title: t('jobs.filters.jobType'),
      option: jobTypes,
    },
    {
      key: 'status',
      title: t('jobs.filters.status'),
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
}
