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
import { PaginationState } from '@tanstack/react-table'
import { useState } from 'react'
import { useLocalStorage } from 'usehooks-ts'

const usePaginationWithStorage = (tableKey: string = 'default-table') => {
  const [pageIndex, setPageIndex] = useState(0)
  const [pageSize, setPageSize] = useLocalStorage(`${tableKey}-page-size`, 10)

  const pagination: PaginationState = {
    pageIndex,
    pageSize,
  }

  const setPagination = (
    updater: PaginationState | ((state: PaginationState) => PaginationState)
  ) => {
    const newState = typeof updater === 'function' ? updater(pagination) : updater
    setPageIndex(newState.pageIndex)
    setPageSize(newState.pageSize)
  }

  return [pagination, setPagination] as const
}

export default usePaginationWithStorage
