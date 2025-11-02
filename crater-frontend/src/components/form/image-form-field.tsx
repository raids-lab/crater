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
import { useMemo } from 'react'
import { FieldPath, FieldValues, UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form'

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

  // 创建过滤器配置
  const filters: ComboboxFilter<ImageInfoResponse>[] = useMemo(() => {
    return [
      {
        id: 'tag-filter',
        title: '标签',
        enabled: true,
        filterFn: (items: ComboboxItem<ImageInfoResponse>[], selectedTags: string[]) => {
          if (selectedTags.length === 0) return items

          return items.filter((item) => {
            if (!item.tags || item.tags.length === 0) return false
            const lowerCaseTags = new Set(item.tags.map((t) => t.toLowerCase()))
            // AND logic: item must have ALL selected tags
            return selectedTags.every((tag) => lowerCaseTags.has(tag.toLowerCase()))
          })
        },
      },
      {
        id: 'arch-filter',
        title: '架构',
        enabled: true,
        filterFn: (items: ComboboxItem<ImageInfoResponse>[], selectedArchs: string[]) => {
          if (selectedArchs.length === 0) return items

          return items.filter((item) => {
            const detail = item.detail as { archs?: string[] } | undefined
            const archs = detail?.archs
            if (!archs) return false

            const itemArchs = archs.map((arch: string) =>
              arch.includes('/') ? arch.split('/').slice(1).join('/') : arch
            )
            return selectedArchs.some((selectedArch) => itemArchs.includes(selectedArch))
          })
        },
      },
    ]
  }, [])

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
