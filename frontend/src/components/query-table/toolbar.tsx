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
    <div className="flex items-center justify-between">
      <div className="flex flex-row items-center space-x-2">
        {children}
        <div className="relative ml-auto h-9 flex-1 md:grow-0">
          <SearchIcon className="text-muted-foreground absolute top-2.5 left-2.5 size-4" />
          {globalSearch?.enabled && (
            <Input
              placeholder={
                globalSearch.placeholder ?? t('dataTableToolbar.globalSearchPlaceholder')
              }
              value={table.getState().globalFilter || ''}
              onChange={(event) => table.setGlobalFilter(event.target.value)}
              className="bg-background h-9 w-[150px] pl-8 lg:w-[250px]"
            />
          )}
          {filterInput && (
            <Input
              placeholder={filterInput.placeholder}
              value={(table.getColumn(filterInput.key)?.getFilterValue() as string) ?? ''}
              onChange={(event) =>
                table.getColumn(filterInput.key)?.setFilterValue(event.target.value)
              }
              className="bg-background h-9 w-[150px] pl-8 lg:w-[250px]"
            />
          )}
        </div>
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
      <DataTableViewOptions table={table} getHeader={getHeader} />
    </div>
  )
}
