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
import { Check, ChevronsUpDown } from 'lucide-react'
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
} from '@/components/ui/command'
import { FormControl } from '@/components/ui/form'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import { cn } from '@/lib/utils'

import { TagFilter, UseTagFilter } from './ImageFormField'

export interface ComboboxItem<T> {
  label: string
  value: string
  selectedLabel?: string
  tags?: string[]
  detail?: T
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
  tags?: string[] // Optional predefined tags
  tagFilter?: React.ReactNode // Optional custom tag filter component
}

const getSelectedLabel = <T,>(items: ComboboxItem<T>[], current: string): string | undefined => {
  const selectedItem = items.find((item) => item.value === current)
  return selectedItem ? (selectedItem.selectedLabel ?? selectedItem.label) : undefined
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
  tags,
  tagFilter,
}: ComboboxProps<T>) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const { allTags, selectedTags, toggleTag, filterItemsByTags } = UseTagFilter(items, tags)

  const filteredItems = useMemo(() => {
    const tagFilteredItems = filterItemsByTags(items)
    return tagFilteredItems.filter((item) => {
      return searchQuery === '' || item.label.toLowerCase().includes(searchQuery.toLowerCase())
    })
  }, [items, searchQuery, filterItemsByTags])

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

      {tagFilter ||
        (allTags.length > 0 && (
          <TagFilter tags={allTags} selectedTags={selectedTags} onTagToggle={toggleTag} />
        ))}

      <CommandList>
        <CommandEmpty>{t('combobox.noResults', { formTitle })}</CommandEmpty>
        <CommandGroup>
          <div>
            {filteredItems.map((item) => (
              <CommandItem
                value={item.label}
                key={item.value}
                onSelect={() => {
                  handleSelect(item.value)
                  setOpen(false)
                }}
                className="flex w-full flex-row items-center justify-between"
              >
                <div className="flex flex-col">
                  <div>{renderLabel ? renderLabel(item) : item.label}</div>
                  {item.tags && item.tags.length > 0 && (
                    <div className="mt-1 flex gap-1">
                      {item.tags.map((tag) => (
                        <Badge key={tag} variant="secondary" className="text-xs">
                          {tag}
                        </Badge>
                      ))}
                    </div>
                  )}
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
      <CommandDialog open={open} onOpenChange={setOpen}>
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
