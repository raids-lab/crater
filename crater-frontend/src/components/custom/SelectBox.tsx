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
import { CheckIcon, Cross2Icon } from '@radix-ui/react-icons'
import React from 'react'

import { cn } from '@/lib/utils'

import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { DialogOverlay } from '@radix-ui/react-dialog'
import { ChevronsUpDown, XIcon } from 'lucide-react'

interface Option {
  value: string
  label: string
  labelNote?: string
}

interface SelectBoxProps {
  options: Option[]
  value?: string[]
  onChange?: (values: string[]) => void
  onInputChange?: (value: string) => void
  placeholder?: string
  inputPlaceholder?: string
  emptyPlaceholder?: string
  className?: string
}

const SelectBox = ({
  ref,
  inputPlaceholder,
  emptyPlaceholder,
  placeholder,
  className,
  options,
  value,
  onChange,
  onInputChange,
}: SelectBoxProps & {
  ref?: React.RefObject<HTMLInputElement>
}) => {
  const { t } = useTranslation()
  const [searchTerm, setSearchTerm] = React.useState<string>('')
  const [isOpen, setIsOpen] = React.useState(false)

  const handleSelect = (selectedValue: string) => {
    const newValue =
      value?.includes(selectedValue) && Array.isArray(value)
        ? value.filter((v) => v !== selectedValue)
        : [...(value ?? []), selectedValue]
    onChange?.(newValue)
  }

  const handleClear = () => {
    onChange?.([])
  }

  const handleInputChange = (value: string) => {
    setSearchTerm(value)
    onInputChange?.(value)
  }

  return (
    <div>
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <PopoverTrigger asChild>
          <Button
            variant={'outline'}
            className={cn(
              'flex h-full min-h-[36px] w-full cursor-pointer items-center justify-between px-3 py-1 font-normal whitespace-nowrap',
              'data-[state=open]:border-ring data-[state=open]:ring-ring/50 data-[state=open]:ring-[3px]',
              className
            )}
          >
            <div
              className={cn('items-center gap-1 overflow-hidden text-sm', 'flex grow flex-wrap')}
            >
              {value && value.length > 0 ? (
                options
                  .filter((option) => value.includes(option.value))
                  ?.map((option) => (
                    <span
                      key={option.value}
                      className="bg-secondary text-secondary-foreground focus:ring-ring inline-flex items-center gap-1 rounded-md border py-0.5 pr-1 pl-2 text-xs font-medium transition-colors focus:ring-2 focus:ring-offset-2 focus:outline-hidden"
                    >
                      <span>{option.label}</span>
                      <span
                        onClick={(e) => {
                          e.preventDefault()
                          handleSelect(option.value)
                        }}
                        className="text-muted-foreground/60 hover:bg-accent hover:text-muted-foreground flex items-center rounded-sm px-[1px]"
                      >
                        <Cross2Icon />
                      </span>
                    </span>
                  ))
              ) : (
                <span className="text-muted-foreground mr-auto">{placeholder}</span>
              )}
            </div>
            <div className="text-muted-foreground hover:text-foreground flex items-center self-stretch pl-1 [&>div]:flex [&>div]:items-center [&>div]:self-stretch">
              {value && value.length > 0 ? (
                <div
                  onClick={(e) => {
                    e.preventDefault()
                    handleClear()
                  }}
                >
                  <XIcon className="size-4" />
                </div>
              ) : (
                <div>
                  <ChevronsUpDown className="size-4 shrink-0 opacity-50" />
                </div>
              )}
            </div>
          </Button>
        </PopoverTrigger>
        <DialogOverlay>
          <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0" align="start">
            <Command>
              <div className="relative">
                <CommandInput
                  value={searchTerm}
                  onValueChange={handleInputChange}
                  ref={ref}
                  placeholder={inputPlaceholder ?? t('selectBox.inputPlaceholder')}
                  className="h-9"
                />
                {searchTerm && (
                  <div
                    className="text-muted-foreground hover:text-foreground absolute inset-y-0 right-0 flex cursor-pointer items-center pr-3"
                    onClick={() => {
                      setSearchTerm('')
                      onInputChange?.('')
                    }}
                  >
                    <Cross2Icon className="size-4" />
                  </div>
                )}
              </div>
              <CommandList>
                <CommandEmpty>{emptyPlaceholder ?? t('selectBox.emptyPlaceholder')}</CommandEmpty>
                <CommandGroup>
                  <ScrollArea>
                    <div className="max-h-64">
                      {options?.map((option) => {
                        const isSelected = Array.isArray(value) && value.includes(option.value)
                        return (
                          <CommandItem
                            key={option.value}
                            onSelect={() => handleSelect(option.value)}
                          >
                            <div
                              className={cn(
                                'border-primary mr-2 flex size-4 items-center justify-center rounded-sm border',
                                isSelected
                                  ? 'bg-primary text-primary-foreground'
                                  : 'opacity-50 [&_svg]:invisible'
                              )}
                            >
                              <CheckIcon className="text-primary-foreground" />
                            </div>
                            <span>
                              {option.label}
                              {option.labelNote && (
                                <span className="text-muted-foreground ml-2">
                                  {'@'}
                                  {option.labelNote}
                                </span>
                              )}
                            </span>
                          </CommandItem>
                        )
                      })}
                    </div>
                  </ScrollArea>
                </CommandGroup>
              </CommandList>
            </Command>
          </PopoverContent>
        </DialogOverlay>
      </Popover>
    </div>
  )
}

SelectBox.displayName = 'SelectBox'

export default SelectBox
