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
import { Table } from '@tanstack/react-table'
import { SearchIcon, XIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

import { DataTableFacetedFilter, DataTableFacetedFilterOption } from './faceted-filter'
import { DataTableViewOptions } from './view-options'

export type DataTableToolbarConfig = {
  filterOptions: readonly {
    key: string
    title: string
    option?: DataTableFacetedFilterOption[]
    defaultValues?: string[]
  }[]
  getHeader: (key: string) => string
} & (
  | {
      filterInput: { placeholder: string; key: string }
      globalSearch?: undefined
    }
  | {
      filterInput?: undefined
      globalSearch: { enabled: boolean; placeholder?: string }
    }
)

interface DataTableToolbarProps<TData> extends React.HTMLAttributes<HTMLDivElement> {
  table: Table<TData>
  config: DataTableToolbarConfig
  isLoading: boolean
}

export function DataTableToolbar<TData>({
  table,
  config: { filterInput, filterOptions, getHeader, globalSearch },
  isLoading,
  children,
}: DataTableToolbarProps<TData>) {
  const { t } = useTranslation()
  const isFiltered =
    table.getState().columnFilters.length > 0 ||
    (globalSearch?.enabled && Boolean(table.getState().globalFilter))

  return (
    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
      <div className="flex w-full min-w-0 flex-wrap items-center gap-2 sm:w-auto sm:flex-nowrap">
        {children}
        {(globalSearch?.enabled || filterInput) && (
          <div className="relative h-9 w-full min-w-0 sm:ml-auto sm:w-auto sm:flex-none">
            <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
            {globalSearch?.enabled && (
              <Input
                placeholder={
                  globalSearch.placeholder ?? t('dataTableToolbar.globalSearchPlaceholder')
                }
                value={table.getState().globalFilter || ''}
                onChange={(event) => table.setGlobalFilter(event.target.value)}
                className="bg-background h-9 w-full min-w-0 pl-8 sm:w-[150px] lg:w-[250px]"
              />
            )}
            {filterInput && (
              <Input
                placeholder={filterInput.placeholder}
                value={(table.getColumn(filterInput.key)?.getFilterValue() as string) ?? ''}
                onChange={(event) =>
                  table.getColumn(filterInput.key)?.setFilterValue(event.target.value)
                }
                className="bg-background h-9 w-full min-w-0 pl-8 sm:w-[150px] lg:w-[250px]"
              />
            )}
          </div>
        )}
        {filterOptions.map(
          (filterOption) =>
            table.getColumn(filterOption.key) && (
              <DataTableFacetedFilter
                key={filterOption.key}
                column={table.getColumn(filterOption.key)}
                title={filterOption.title}
                options={filterOption.option}
                defaultValues={filterOption.defaultValues}
              />
            )
        )}
        {isFiltered && !isLoading && (
          <Button
            variant="outline"
            size="icon"
            title={t('dataTableToolbar.clearFiltersButtonTitle')}
            type="button"
            onClick={() => {
              table.resetColumnFilters()
              if (globalSearch?.enabled) {
                table.setGlobalFilter('')
              }
            }}
            className="size-9 border-dashed"
          >
            <XIcon className="size-4" />
          </Button>
        )}
      </div>
      <div className="shrink-0 self-start sm:self-auto">
        <DataTableViewOptions table={table} getHeader={getHeader} />
      </div>
    </div>
  )
}
