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
import { useTranslation } from 'react-i18next'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import FormLabelMust from '@/components/form/FormLabelMust'
import Combobox, { ComboboxItem } from '@/components/form/Combobox'
import ImageItem from '@/components/form/ImageItem'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import useImageQuery from '@/hooks/query/useImageQuery'
import { JobType } from '@/services/api/vcjob'
import { useCallback, useMemo, useState } from 'react'
import { Badge } from '@/components/ui/badge'

interface ImageFormFieldProps<T extends FieldValues> {
  form: UseFormReturn<T>
  name: FieldPath<T>
  jobType?: JobType
  required?: boolean
  label?: string
  className?: string
}

export function ImageFormField<T extends FieldValues>({
  form,
  name,
  jobType = JobType.Jupyter,
  required = true,
  label,
  className,
}: ImageFormFieldProps<T>) {
  const { t } = useTranslation()
  const { data: images } = useImageQuery(jobType)

  return (
    <FormField
      control={form.control}
      name={name}
      render={({ field }) => (
        <FormItem className={className}>
          <FormLabel>
            {label || t('imageFormField.label')}
            {required && <FormLabelMust />}
          </FormLabel>

          <FormControl>
            <Combobox
              items={images ?? []}
              current={field.value}
              handleSelect={(value) => field.onChange(value)}
              renderLabel={(item) => <ImageItem item={item} />}
              formTitle={t('imageFormField.comboboxFormTitle')}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

export interface TagFilterProps {
  tags: string[]
  selectedTags: string[]
  onTagToggle: (tag: string) => void
  className?: string
  // label?: string;
}

export function TagFilter({ tags, selectedTags, onTagToggle, className }: TagFilterProps) {
  if (tags.length === 0) return null

  return (
    <div className={`border-t border-b px-2 py-2 ${className}`}>
      <div className="flex flex-wrap gap-1 pb-1">
        {tags.map((tag) => (
          <Badge
            key={tag}
            variant={selectedTags.includes(tag) ? 'default' : 'outline'}
            className="mb-1 cursor-pointer"
            onClick={(e) => {
              e.preventDefault()
              onTagToggle(tag)
            }}
          >
            {tag}
          </Badge>
        ))}
      </div>
    </div>
  )
}

// Hook for tag filtering logic
export function UseTagFilter<T>(
  items: ComboboxItem<T>[],
  externalTags?: string[]
): {
  allTags: string[]
  selectedTags: string[]
  setSelectedTags: React.Dispatch<React.SetStateAction<string[]>>
  toggleTag: (tag: string) => void
  filterItemsByTags: (items: ComboboxItem<T>[]) => ComboboxItem<T>[]
} {
  const [selectedTags, setSelectedTags] = useState<string[]>([])

  // Get all unique tags
  const allTags = useMemo(() => {
    if (externalTags && externalTags.length > 0) return externalTags

    const tagSet = new Set<string>()
    const lowerCaseTagSet = new Set<string>()
    items.forEach((item) => {
      if (item.tags && Array.isArray(item.tags)) {
        item.tags.forEach((tag) => {
          const lowerTag = tag.toLowerCase()
          if (!lowerCaseTagSet.has(lowerTag)) {
            lowerCaseTagSet.add(lowerTag)
            tagSet.add(tag) // 保留原始大小写
          }
        })
      }
    })
    return Array.from(tagSet)
  }, [items, externalTags])

  // Toggle tag selection
  const toggleTag = useCallback((tag: string) => {
    setSelectedTags((prev) => (prev.includes(tag) ? prev.filter((t) => t !== tag) : [...prev, tag]))
  }, [])

  // Filter items based on selected tags
  const filterItemsByTags = useCallback(
    (itemsToFilter: ComboboxItem<T>[]) => {
      if (selectedTags.length === 0) return itemsToFilter

      return itemsToFilter.filter((item) => {
        const lowerCaseTags = new Set(item.tags?.map((t) => t.toLowerCase()))
        return item.tags && selectedTags.some((tag) => lowerCaseTags.has(tag.toLowerCase()))
      })
    },
    [selectedTags]
  )

  return {
    allTags,
    selectedTags,
    setSelectedTags,
    toggleTag,
    filterItemsByTags,
  }
}
