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
import { useQuery } from '@tanstack/react-query'
import { useCallback, useMemo, useState } from 'react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'
import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import Combobox, { ComboboxFilter, ComboboxItem } from '@/components/form/combobox'
import FormLabelMust from '@/components/form/form-label-must'
import ImageItem from '@/components/form/image-item'

import { ImageInfoResponse } from '@/services/api/imagepack'
import { JobType } from '@/services/api/vcjob'
import { queryBaseImages } from '@/services/query/image'

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
  const { data: images } = useQuery(queryBaseImages(jobType))

  // 使用便利函数创建过滤器
  const filters = useImageFilters(images ?? [], {
    enableTagFilter: true,
    enableArchFilter: true,
  })

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
              current={field.value?.imageLink || ''}
              handleSelect={(value) => {
                // 找到选中的镜像对象
                const selectedImage = images?.find((img) => img.value === value)
                if (selectedImage?.detail) {
                  field.onChange({
                    imageLink: selectedImage.detail.imageLink,
                    archs: selectedImage.detail.archs || [],
                  })
                } else {
                  field.onChange({
                    imageLink: value,
                    archs: [],
                  })
                }
              }}
              renderLabel={(item) => <ImageItem item={item} />}
              formTitle={t('imageFormField.comboboxFormTitle')}
              filters={filters}
            />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
  )
}

// 通用过滤器容器组件
interface FilterContainerProps {
  label: string
  tooltip: string
  children: React.ReactNode
  className?: string
}

function FilterContainer({ label, tooltip, children, className }: FilterContainerProps) {
  return (
    <div className={`px-2 py-2 ${className}`}>
      <div className="flex items-center gap-3">
        <Tooltip>
          <TooltipTrigger asChild>
            <div className="text-muted-foreground w-13 shrink-0 cursor-help text-xs font-medium hover:cursor-help">
              {label}
            </div>
          </TooltipTrigger>
          <TooltipContent>
            <p className="text-xs">{tooltip}</p>
          </TooltipContent>
        </Tooltip>
        <div className="flex min-h-[24px] flex-1 items-center">
          <div className="flex flex-wrap gap-1">{children}</div>
        </div>
      </div>
    </div>
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
    <FilterContainer
      label="标签选择"
      tooltip="筛选同时包含所有选中标签的镜像"
      className={className}
    >
      {tags.map((tag) => (
        <Badge
          key={tag}
          variant={selectedTags.includes(tag) ? 'default' : 'outline'}
          className="cursor-pointer text-xs"
          onClick={(e) => {
            e.preventDefault()
            onTagToggle(tag)
          }}
        >
          {tag}
        </Badge>
      ))}
    </FilterContainer>
  )
}

export interface ArchFilterProps {
  archs: string[]
  selectedArchs: string[]
  onArchToggle: (arch: string) => void
  className?: string
}

export function ArchFilter({ archs, selectedArchs, onArchToggle, className }: ArchFilterProps) {
  if (archs.length === 0) return null

  return (
    <FilterContainer label="架构选择" tooltip="筛选包含任一选中架构的镜像" className={className}>
      {archs.map((arch) => {
        const isArm = arch.toLowerCase().includes('arm')
        return (
          <Badge
            key={arch}
            variant={selectedArchs.includes(arch) ? 'default' : 'outline'}
            className={`cursor-pointer text-xs ${
              selectedArchs.includes(arch)
                ? isArm
                  ? 'bg-orange-500 text-white hover:bg-orange-600'
                  : 'bg-blue-500 text-white hover:bg-blue-600'
                : isArm
                  ? 'border-orange-200 text-orange-700 hover:bg-orange-50'
                  : 'border-blue-200 text-blue-700 hover:bg-blue-50'
            }`}
            onClick={(e) => {
              e.preventDefault()
              onArchToggle(arch)
            }}
          >
            {arch}
          </Badge>
        )
      })}
    </FilterContainer>
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

  // Filter items based on selected tags (AND logic - must have all selected tags)
  const filterItemsByTags = useCallback(
    (itemsToFilter: ComboboxItem<T>[]) => {
      if (selectedTags.length === 0) return itemsToFilter

      return itemsToFilter.filter((item) => {
        if (!item.tags || item.tags.length === 0) return false

        const lowerCaseTags = new Set(item.tags.map((t) => t.toLowerCase()))
        // AND logic: item must have ALL selected tags
        return selectedTags.every((tag) => lowerCaseTags.has(tag.toLowerCase()))
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

// Hook for architecture filtering logic
export function UseArchFilter<T>(
  items: ComboboxItem<T>[],
  externalArchs?: string[]
): {
  allArchs: string[]
  selectedArchs: string[]
  setSelectedArchs: React.Dispatch<React.SetStateAction<string[]>>
  toggleArch: (arch: string) => void
  filterItemsByArchs: (items: ComboboxItem<T>[]) => ComboboxItem<T>[]
} {
  const [selectedArchs, setSelectedArchs] = useState<string[]>([])

  // Get all unique architectures
  const allArchs = useMemo(() => {
    if (externalArchs && externalArchs.length > 0) return externalArchs

    const archSet = new Set<string>()
    items.forEach((item) => {
      // 使用类型断言检查是否有 archs 属性
      const detail = item.detail as { archs?: string[] } | undefined
      const archs = detail?.archs
      if (archs && Array.isArray(archs)) {
        archs.forEach((arch: string) => {
          // 处理 "linux/amd64" 格式，取架构部分
          const archName = arch.includes('/') ? arch.split('/').slice(1).join('/') : arch
          archSet.add(archName)
        })
      }
    })
    return Array.from(archSet).sort()
  }, [items, externalArchs])

  // Toggle architecture selection
  const toggleArch = useCallback((arch: string) => {
    setSelectedArchs((prev) =>
      prev.includes(arch) ? prev.filter((a) => a !== arch) : [...prev, arch]
    )
  }, [])

  // Filter items based on selected architectures
  const filterItemsByArchs = useCallback(
    (itemsToFilter: ComboboxItem<T>[]) => {
      if (selectedArchs.length === 0) return itemsToFilter

      return itemsToFilter.filter((item) => {
        // 使用类型断言检查是否有 archs 属性
        const detail = item.detail as { archs?: string[] } | undefined
        const archs = detail?.archs
        if (!archs) return false

        const itemArchs = archs.map((arch: string) =>
          arch.includes('/') ? arch.split('/').slice(1).join('/') : arch
        )
        return selectedArchs.some((selectedArch) => itemArchs.includes(selectedArch))
      })
    },
    [selectedArchs]
  )

  return {
    allArchs,
    selectedArchs,
    setSelectedArchs,
    toggleArch,
    filterItemsByArchs,
  }
}

// React Hook 组件：标签过滤器
export function useTagFilterComponent<T>(
  items: ComboboxItem<T>[],
  externalTags?: string[]
): ComboboxFilter<T> | null {
  const { allTags, selectedTags, toggleTag, filterItemsByTags } = UseTagFilter(items, externalTags)

  if (allTags.length === 0) return null

  return {
    id: 'tag-filter',
    enabled: true,
    component: <TagFilter tags={allTags} selectedTags={selectedTags} onTagToggle={toggleTag} />,
    filterFn: filterItemsByTags,
  }
}

// React Hook 组件：架构过滤器
export function useArchFilterComponent<T>(
  items: ComboboxItem<T>[],
  externalArchs?: string[]
): ComboboxFilter<T> | null {
  const { allArchs, selectedArchs, toggleArch, filterItemsByArchs } = UseArchFilter(
    items,
    externalArchs
  )

  if (allArchs.length === 0) return null

  return {
    id: 'arch-filter',
    enabled: true,
    component: (
      <ArchFilter archs={allArchs} selectedArchs={selectedArchs} onArchToggle={toggleArch} />
    ),
    filterFn: filterItemsByArchs,
  }
}

// 便利函数：为镜像创建标准过滤器集合
export function useImageFilters(
  items: ComboboxItem<ImageInfoResponse>[],
  options: {
    enableTagFilter?: boolean
    enableArchFilter?: boolean
    externalTags?: string[]
    externalArchs?: string[]
  } = {}
): ComboboxFilter<ImageInfoResponse>[] {
  const { enableTagFilter = true, enableArchFilter = false, externalTags, externalArchs } = options

  const tagFilter = useTagFilterComponent(items, externalTags)
  const archFilter = useArchFilterComponent(items, externalArchs)

  return useMemo(() => {
    const filters: ComboboxFilter<ImageInfoResponse>[] = []

    if (enableTagFilter && tagFilter) {
      filters.push(tagFilter)
    }

    if (enableArchFilter && archFilter) {
      filters.push(archFilter)
    }

    return filters
  }, [enableTagFilter, enableArchFilter, tagFilter, archFilter])
}
