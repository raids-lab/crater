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
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-nocheck
import { ChevronsUpDown, CircleHelpIcon, XIcon } from 'lucide-react'
import * as React from 'react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import { cn } from '@/lib/utils'

import { NothingCore } from '../placeholder/nothing'

// Sample predefined tags
const predefinedTags = [
  { value: 'react', label: 'React' },
  { value: 'nextjs', label: 'Next.js' },
  { value: 'tailwind', label: 'Tailwind CSS' },
  { value: 'ui', label: 'UI' },
  { value: 'design', label: 'Design' },
  { value: 'api', label: 'API' },
  { value: 'database', label: 'Database' },
  { value: 'javascript', label: 'JavaScript' },
  { value: 'typescript', label: 'TypeScript' },
  { value: 'node', label: 'Node.js' },
  { value: 'express', label: 'Express' },
  { value: 'graphql', label: 'GraphQL' },
  { value: 'redux', label: 'Redux' },
  { value: 'vue', label: 'Vue.js' },
  { value: 'angular', label: 'Angular' },
  { value: 'svelte', label: 'Svelte' },
  { value: 'docker', label: 'Docker' },
  { value: 'kubernetes', label: 'Kubernetes' },
  { value: 'aws', label: 'AWS' },
  { value: 'azure', label: 'Azure' },
  { value: 'gcp', label: 'Google Cloud' },
  { value: 'firebase', label: 'Firebase' },
  { value: 'mongodb', label: 'MongoDB' },
  { value: 'postgresql', label: 'PostgreSQL' },
  { value: 'mysql', label: 'MySQL' },
  { value: 'redis', label: 'Redis' },
  { value: 'testing', label: 'Testing' },
  { value: 'jest', label: 'Jest' },
  { value: 'cypress', label: 'Cypress' },
  { value: 'storybook', label: 'Storybook' },
]

export interface Tag {
  value: string
  label?: string
}

interface TagsInputProps<T extends FieldValues> {
  /** The form object to populate */
  form: UseFormReturn<T>
  /** The path to the tags field in the form data */
  tagsPath: FieldPath<T>
  /** Optional label for the form field */
  label?: string
  /** Optional description for the form field */
  description?: string
  /** Optional custom predefined tags */
  customTags?: Tag[]
  /** Whether the field is required */
  required?: boolean
  /** Whether the search and add field is required */
  imageBuildArch?: boolean
}

export function TagsInput<T extends FieldValues>({
  form,
  tagsPath,
  label,
  description,
  customTags,
  required = false,
  imageBuildArch = false,
}: TagsInputProps<T>) {
  const { t } = useTranslation()
  const [open, setOpen] = React.useState(false)
  const [inputValue, setInputValue] = React.useState('')

  // Use the provided tags or default to predefined ones
  const availableTags = customTags || predefinedTags

  // Get current tags from form
  const tags = form.watch(tagsPath) || []

  // Add a new custom tag
  const addCustomTag = () => {
    if (
      inputValue.trim() !== '' &&
      !tags.some((tag: Tag) => tag.value.toLowerCase() === inputValue.toLowerCase())
    ) {
      const newTag = {
        value: inputValue,
        label: inputValue,
      }
      form.setValue(tagsPath, [...tags, newTag], { shouldValidate: true })
      setInputValue('')
    }
  }

  // Remove a tag
  const removeTag = (tagToRemove: string) => {
    form.setValue(
      tagsPath,
      tags.filter((tag: Tag) => tag.value !== tagToRemove),
      { shouldValidate: true }
    )
  }

  // Handle key press in the input field
  const handleKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter') {
      e.preventDefault()
      if (inputValue.trim() !== '') {
        addCustomTag()
      }
    }
  }

  return (
    <FormField
      control={form.control}
      name={tagsPath}
      render={() => (
        <FormItem>
          <FormLabel>
            {label || t('tagsInput.label')}
            {required && <span className="text-destructive ml-1">*</span>}
            <TooltipProvider delayDuration={100}>
              <Tooltip>
                <TooltipTrigger asChild>
                  <CircleHelpIcon className="text-muted-foreground size-4 hover:cursor-help" />
                </TooltipTrigger>
                <TooltipContent>
                  <p>{description}</p>
                </TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </FormLabel>
          <FormControl className="flex flex-col gap-2">
            <div className="flex gap-2">
              {/* Dropdown for existing tags and adding new tags */}
              <Popover modal open={open} onOpenChange={setOpen}>
                <PopoverTrigger asChild>
                  <Button
                    type="button"
                    variant={'outline'}
                    className={cn(
                      'flex h-full min-h-[36px] w-full cursor-pointer items-center justify-between px-3 py-1 font-normal whitespace-nowrap',
                      'data-[state=open]:border-ring data-[state=open]:ring-ring/50 data-[state=open]:ring-[3px]'
                    )}
                  >
                    <div
                      className={cn(
                        'items-center gap-1 overflow-hidden text-sm',
                        'flex grow flex-wrap'
                      )}
                    >
                      {tags.length > 0 ? (
                        <div className="flex flex-wrap gap-2">
                          {tags.map((tag: Tag) => (
                            <span
                              key={tag.value}
                              className="bg-secondary text-secondary-foreground focus:ring-ring inline-flex items-center gap-1 rounded-md border py-0.5 pr-1 pl-2 text-xs font-medium transition-colors focus:ring-2 focus:ring-offset-2 focus:outline-hidden"
                            >
                              <span>{tag.label ?? tag.value}</span>
                              <span
                                onClick={(e) => {
                                  e.preventDefault()
                                  removeTag(tag.value)
                                }}
                                className="text-muted-foreground/60 hover:bg-accent hover:text-muted-foreground flex cursor-pointer items-center rounded-sm px-[1px]"
                              >
                                <XIcon className="size-3" />
                              </span>
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span className="text-muted-foreground mr-auto">
                          {imageBuildArch
                            ? t('tagsInput.noArchsLabel')
                            : t('tagsInput.noTagsLabel')}
                        </span>
                      )}
                    </div>
                    <div className="text-muted-foreground hover:text-foreground flex items-center self-stretch pl-1 [&>div]:flex [&>div]:items-center [&>div]:self-stretch">
                      {tags.length > 0 ? (
                        <div
                          onClick={(e) => {
                            e.preventDefault()
                            form.setValue(tagsPath, [], {
                              shouldValidate: true,
                            })
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
                <PopoverContent className="w-[var(--radix-popover-trigger-width)] p-0">
                  <Command>
                    <CommandInput
                      placeholder={
                        imageBuildArch
                          ? t('tagsInput.archSearchPlaceholder')
                          : t('tagsInput.searchPlaceholder')
                      }
                    />
                    <CommandList className="max-h-[200px] overflow-y-auto">
                      <CommandEmpty>
                        <NothingCore className="gap-2" />
                      </CommandEmpty>
                      {availableTags.filter((tag) => !tags.some((t: Tag) => t.value === tag.value))
                        .length > 0 && (
                        <CommandGroup
                          heading={
                            imageBuildArch
                              ? t('tagsInput.commonArchsLabel')
                              : t('tagsInput.commonTagsLabel')
                          }
                        >
                          {availableTags
                            .filter((tag) => !tags.some((t: Tag) => t.value === tag.value))
                            .map((tag) => (
                              <CommandItem
                                key={tag.value}
                                value={tag.value}
                                onSelect={() => {
                                  form.setValue(tagsPath, [...tags, tag], {
                                    shouldValidate: true,
                                  })
                                }}
                                className="cursor-pointer"
                              >
                                {tag.label ?? tag.value}
                              </CommandItem>
                            ))}
                        </CommandGroup>
                      )}
                    </CommandList>
                    {!imageBuildArch && (
                      <div className="border-t p-2">
                        <div className="text-muted-foreground mb-2 ml-1 text-xs font-medium">
                          {t('tagsInput.addNewLabel')}
                        </div>
                        <div className="flex gap-2">
                          <Input
                            placeholder={t('tagsInput.newTagPlaceholder')}
                            value={inputValue}
                            onChange={(e) => setInputValue(e.target.value)}
                            onKeyDown={handleKeyDown}
                            className="flex-1 text-sm"
                          />
                          <Button
                            type="button"
                            variant="secondary"
                            onClick={() => {
                              addCustomTag()
                            }}
                            disabled={!inputValue.trim()}
                          >
                            {t('tagsInput.addButton')}
                          </Button>
                        </div>
                      </div>
                    )}
                  </Command>
                </PopoverContent>
              </Popover>
            </div>
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}
