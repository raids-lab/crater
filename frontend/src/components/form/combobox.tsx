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
import { Check, ChevronsUpDown, ListFilter, XIcon } from 'lucide-react'
// import { ScrollArea } from "@/components/ui/scroll-area";
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from '@/components/ui/command'
import { FormControl } from '@/components/ui/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Separator } from '@/components/ui/separator'

import { cn } from '@/lib/utils'

import TooltipButton from '../button/tooltip-button'

export interface ComboboxItem<T> {
  label: string
  value: string
  selectedLabel?: string
  tags?: string[]
  detail?: T
}

// 过滤器选项接口
export interface ComboboxFilterOption {
  label: string
  value: string
  icon?: React.ComponentType<{ className?: string }>
}

// 通用过滤器接口 - 类似 DataTableFacetedFilter
export interface ComboboxFilter<T> {
  id: string // 过滤器唯一标识
  title: string // 过滤器标题
  options?: ComboboxFilterOption[] // 过滤器选项，如果不提供则从 items 中推断
  filterFn: (items: ComboboxItem<T>[], selectedValues: string[]) => ComboboxItem<T>[] // 过滤函数
  enabled?: boolean // 是否启用，默认true
}

type ComboboxProps<T> = React.HTMLAttributes<HTMLDivElement> & {
  formTitle: string
  items: ComboboxItem<T>[]
  current: string
  disabled?: boolean
  handleSelect: (value: string) => void
  renderLabel?: (item: ComboboxItem<T>) => React.ReactNode
  className?: string
  useDialog?: boolean
  filters?: ComboboxFilter<T>[] // 通用过滤器数组
}

const getSelectedLabel = <T,>(items: ComboboxItem<T>[], current: string): string | undefined => {
  const selectedItem = items.find((item) => item.value === current)
  return selectedItem ? (selectedItem.selectedLabel ?? selectedItem.label) : undefined
}

// 类似 DataTableFacetedFilter 的 Combobox 过滤器组件
interface ComboboxFacetedFilterProps<T> {
  filter: ComboboxFilter<T>
  items: ComboboxItem<T>[]
  selectedValues: Set<string>
  onValuesChange: (values: string[]) => void
}

function ComboboxFacetedFilter<T>({
  filter,
  items,
  selectedValues,
  onValuesChange,
}: ComboboxFacetedFilterProps<T>) {
  const { t } = useTranslation()

  // 从 items 中推断选项（如果未提供）
  const options = useMemo(() => {
    if (filter.options && filter.options.length > 0) {
      return filter.options
    }

    // 根据 filter.id 从 items 中推断选项
    const valueSet = new Set<string>()
    items.forEach((item) => {
      if (filter.id === 'tag-filter' && item.tags) {
        item.tags.forEach((tag) => valueSet.add(tag))
      } else if (filter.id === 'arch-filter') {
        const detail = item.detail as { archs?: string[] } | undefined
        const archs = detail?.archs
        if (archs) {
          archs.forEach((arch: string) => {
            const archName = arch.includes('/') ? arch.split('/').slice(1).join('/') : arch
            valueSet.add(archName)
          })
        }
      }
    })

    return Array.from(valueSet).map((value) => ({
      label: value,
      value,
    }))
  }, [filter.id, filter.options, items])

  if (options.length === 0) return null

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button variant="outline" size="sm" className="h-8 border-dashed">
          <ListFilter className="size-4" />
          <span className="text-xs">{filter.title}</span>
          {selectedValues.size > 0 && (
            <>
              <Separator orientation="vertical" className="h-4" />
              <Badge variant="secondary" className="rounded-sm px-1 font-normal lg:hidden">
                {selectedValues.size}
              </Badge>
              <div className="hidden space-x-1 lg:flex">
                {selectedValues.size > 2 ? (
                  <Badge variant="secondary" className="rounded-sm px-1 font-normal">
                    {selectedValues.size} 已选
                  </Badge>
                ) : (
                  options
                    .filter((option) => selectedValues.has(option.value))
                    .map((option) => (
                      <Badge
                        variant="secondary"
                        key={option.value}
                        className="rounded-sm px-1 font-normal"
                      >
                        {option.label}
                      </Badge>
                    ))
                )}
              </div>
            </>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-[200px] p-0" align="start">
        <Command shouldFilter={false}>
          <CommandList>
            <CommandEmpty>{t('combobox.noResults', { formTitle: filter.title })}</CommandEmpty>
            <CommandGroup>
              {options.map((option) => {
                const isSelected = selectedValues.has(option.value)
                return (
                  <CommandItem
                    key={option.value}
                    onSelect={() => {
                      const newValues = new Set(selectedValues)
                      if (isSelected) {
                        newValues.delete(option.value)
                      } else {
                        newValues.add(option.value)
                      }
                      onValuesChange(Array.from(newValues))
                    }}
                  >
                    <div
                      className={cn(
                        'border-primary flex size-4 items-center justify-center rounded-sm border',
                        isSelected
                          ? 'bg-primary text-primary-foreground'
                          : 'opacity-50 [&_svg]:invisible'
                      )}
                    >
                      <Check className={cn('text-primary-foreground size-4')} />
                    </div>
                    <span>{option.label}</span>
                  </CommandItem>
                )
              })}
            </CommandGroup>
            {selectedValues.size > 0 && (
              <>
                <CommandSeparator />
                <CommandGroup>
                  <CommandItem
                    onSelect={() => onValuesChange([])}
                    className="justify-center text-center"
                  >
                    清除筛选
                  </CommandItem>
                </CommandGroup>
              </>
            )}
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

function Combobox<T>({
  formTitle,
  items,
  current,
  disabled,
  handleSelect,
  renderLabel,
  className,
  useDialog = false,
  filters = [],
}: ComboboxProps<T>) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')

  // 为每个过滤器维护状态
  const [filterStates, setFilterStates] = useState<Record<string, string[]>>({})

  // 启用的过滤器
  const enabledFilters = filters.filter((filter) => filter.enabled !== false)

  const filteredItems = useMemo(() => {
    let result = items

    // 应用所有启用的过滤器
    enabledFilters.forEach((filter) => {
      const selectedValues = filterStates[filter.id] || []
      result = filter.filterFn(result, selectedValues)
    })

    // 应用搜索过滤
    if (searchQuery) {
      result = result.filter((item) => item.label.toLowerCase().includes(searchQuery.toLowerCase()))
    }

    return result
  }, [items, searchQuery, enabledFilters, filterStates])

  const triggerButton = (
    <FormControl>
      <Button
        variant="outline"
        role="combobox"
        type="button"
        aria-expanded={open}
        aria-describedby=""
        className={cn(
          'w-full justify-between px-3 font-normal text-ellipsis whitespace-nowrap',
          'data-[state=open]:border-ring data-[state=open]:ring-ring/50 data-[state=open]:ring-[3px]',
          !current && 'text-muted-foreground',
          className
        )}
        disabled={disabled}
        onClick={() => !useDialog || setOpen(true)}
      >
        <p className="truncate">
          {current ? getSelectedLabel(items, current) : t('combobox.select', { formTitle })}
        </p>
        <ChevronsUpDown className="ml-2 size-4 shrink-0 opacity-50" />
      </Button>
    </FormControl>
  )

  const commandContent = (
    <Command>
      <CommandInput
        placeholder={t('combobox.search', { formTitle })}
        className="h-9"
        onValueChange={setSearchQuery}
      />

      {/* 渲染所有启用的过滤器组件 - 类似 toolbar */}
      {enabledFilters.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 border-b p-2">
          {enabledFilters.map((filter) => (
            <ComboboxFacetedFilter
              key={filter.id}
              filter={filter}
              items={items}
              selectedValues={new Set(filterStates[filter.id] || [])}
              onValuesChange={(values) => {
                setFilterStates((prev) => ({
                  ...prev,
                  [filter.id]: values,
                }))
              }}
            />
          ))}
          {/* 如果当前存在选中的过滤States，那么在最右侧提供一个一键清除按钮 */}
          <div className="flex flex-1 flex-row items-center justify-end">
            {Object.values(filterStates).some((vals) => vals.length > 0) && (
              <TooltipButton
                tooltipContent="清除筛选"
                variant="outline"
                size="icon"
                className="size-8 border-dashed"
                onClick={() => setFilterStates({})}
              >
                <XIcon />
              </TooltipButton>
            )}
          </div>
        </div>
      )}

      <CommandList>
        <CommandEmpty>{t('combobox.noResults', { formTitle })}</CommandEmpty>
        <CommandGroup>
          <div key={`items-${filteredItems.length}`}>
            {filteredItems.map((item) => (
              <CommandItem
                value={item.label}
                key={item.value}
                onSelect={() => {
                  handleSelect(item.value)
                  setOpen(false)
                }}
                className="animate-in fade-in-0 flex w-full flex-row items-center justify-between duration-200"
              >
                <div className="flex w-full flex-col">
                  <div>{renderLabel ? renderLabel(item) : item.label}</div>
                </div>
                <Check
                  className={cn(
                    'ml-auto size-4',
                    item.value === current ? 'opacity-100' : 'opacity-0'
                  )}
                />
              </CommandItem>
            ))}
          </div>
        </CommandGroup>
      </CommandList>
    </Command>
  )

  return useDialog ? (
    <>
      {triggerButton}
      <CommandDialog open={open} onOpenChange={setOpen} className="sm:max-w-[calc(100%-16rem)]">
        {commandContent}
      </CommandDialog>
    </>
  ) : (
    <Popover modal open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>{triggerButton}</PopoverTrigger>
      <PopoverContent
        className="z-50 p-0"
        style={{
          width: 'var(--radix-popover-trigger-width)',
          maxHeight: 'var(--radix-popover-content-available-height)',
        }}
      >
        {commandContent}
      </PopoverContent>
    </Popover>
  )
}

export default Combobox
