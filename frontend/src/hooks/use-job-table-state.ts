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
import { useEffect } from 'react'

import { type JobDisplayPhaseValue, JobPhase, encodeJobFilterPhase } from '@/services/api/vcjob'

import useRemoteTableState from './use-remote-table-state'

// Migrate only Volcano job selections; leave other tables and their saved settings intact.
export default function useJobTableState(...args: Parameters<typeof useRemoteTableState>) {
  const state = useRemoteTableState(...args)
  const { columnFilters, setColumnFilters } = state
  useEffect(() => {
    const legacy = columnFilters.find(({ id }) => id === 'status')
    if (!Array.isArray(legacy?.value) || !legacy.value.includes(JobPhase.Prequeue)) return
    setColumnFilters(
      columnFilters.map((filter) =>
        filter.id === 'status'
          ? {
              ...filter,
              value: [
                ...new Set(
                  (filter.value as string[]).map((value) =>
                    encodeJobFilterPhase(value as JobDisplayPhaseValue)
                  )
                ),
              ],
            }
          : filter
      )
    )
  }, [columnFilters, setColumnFilters])
  return state
}
