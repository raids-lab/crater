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

import {
  getJobFilterOptions,
  getJobPhaseLabel,
  jobPhases,
} from '@/components/badge/job-phase-badge'
import { jobTypes } from '@/components/badge/job-type-badge'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import { JobDisplayPhase, type JobDisplayPhaseValue, JobPhase } from '@/services/api/vcjob'
import type { IFacetResponse } from '@/services/types'

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
    get placeholder() {
      return t('jobs.toolbar.searchPlaceholder')
    },
    key: 'name',
  },
  filterOptions: [
    {
      key: 'jobType',
      get title() {
        return t('jobs.filters.jobType')
      },
      option: jobTypes,
    },
    {
      key: 'status',
      get title() {
        return t('jobs.filters.status')
      },
      option: jobPhases,
    },
  ],
  getHeader: getHeader,
}

export function getRemoteJobToolbarConfig(
  facets?: IFacetResponse,
  allowedJobTypes?: ReadonlyArray<string>
): DataTableToolbarConfig {
  const facetKeys: Record<string, string> = {
    jobType: 'job_type',
    status: 'phase',
  }
  return {
    globalSearch: { enabled: true, placeholder: t('jobs.toolbar.searchPlaceholder') },
    filterOptions: jobToolbarConfig.filterOptions.map((filter) => ({
      ...filter,
      remoteFacets: true,
      showZeroCountOptions: filter.key === 'status',
      option: (filter.key === 'status' ? getJobFilterOptions() : filter.option)
        ?.filter(
          (option) =>
            filter.key !== 'jobType' ||
            !allowedJobTypes ||
            allowedJobTypes.includes(String(option.value))
        )
        .map((option) => {
          const count = facets?.facets[facetKeys[filter.key]]?.find(
            (item) => item.value === String(option.value)
          )?.count
          return {
            ...option,
            count: facets ? (count ?? 0) : undefined,
          }
        }),
    })),
    getHeader,
  }
}

export function getJobOverviewSummary(facets?: IFacetResponse) {
  const phases = (facets?.facets.phase ?? [])
    .filter(({ value }) => value !== JobPhase.Deleted && value !== JobPhase.Freed)
    .map(({ value, count }) => ({
      id: value,
      label: getJobPhaseLabel(value === 'Init' ? JobPhase.Init : (value as JobDisplayPhaseValue))
        .label,
      value: count,
    }))
  const waitingPhases: ReadonlyArray<string> = [
    JobPhase.Pending,
    JobDisplayPhase.Inqueue,
    JobDisplayPhase.Starting,
  ]
  return {
    phases,
    runningCount: phases.find(({ id }) => id === JobPhase.Running)?.value ?? 0,
    waitingCount: phases
      .filter(({ id }) => waitingPhases.includes(id))
      .reduce((total, { value }) => total + value, 0),
  }
}
