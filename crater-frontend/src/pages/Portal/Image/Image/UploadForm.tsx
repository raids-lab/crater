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

import { zodResolver } from '@hookform/resolvers/zod'
import { z } from 'zod'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { useForm } from 'react-hook-form'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import {
  apiUserUploadImage,
  FetchAllUniqueImageTagObjects,
  imageLinkRegex,
  parseImageLink,
} from '@/services/api/imagepack'
import FormLabelMust from '@/components/form/FormLabelMust'
import { JobType } from '@/services/api/vcjob'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'
import LoadableButton from '@/components/button/LoadableButton'
import { PackagePlusIcon } from 'lucide-react'
import { TagsInput } from '@/components/form/TagsInput'

const formSchema = z.object({
  imageLink: z
    .string()
    .min(1, { message: '镜像链接不能为空' })
    .refine((value) => imageLinkRegex.test(value), {
      message: '镜像链接格式不正确',
    }),
  description: z.string().min(1, { message: '镜像描述不能为空' }),
  tags: z
    .array(
      z.object({
        value: z.string(),
      })
    )
    .optional(),
})

type FormSchema = z.infer<typeof formSchema>

interface ImageUploadSheetContentProps {
  closeSheet: () => void
}

function ImageUploadSheetContent({ closeSheet }: ImageUploadSheetContentProps) {
  const queryClient = useQueryClient()

  const { mutate: uploadImagePack, isPending } = useMutation({
    mutationFn: (values: FormSchema) => {
      const { imageName, imageTag } = parseImageLink(values.imageLink)
      return apiUserUploadImage({
        imageLink: values.imageLink,
        imageName: imageName,
        imageTag: imageTag,
        description: values.description,
        taskType: JobType.Custom,
        tags: values.tags?.map((item) => item.value) ?? [],
      })
    },
    onSuccess: async (_, { imageLink }) => {
      await queryClient.invalidateQueries({
        queryKey: ['imagelink', 'list'],
      })
      toast.success(`镜像 ${imageLink} 上传成功`)
      closeSheet()
    },
  })

  const { data: allImageTags } = useQuery({
    queryKey: ['allImageTags'],
    queryFn: FetchAllUniqueImageTagObjects,
  })

  // 1. Define your form.
  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      imageLink: '',
      description: '',
      tags: [],
    },
  })

  // 2. Define a submit handler.
  const onSubmit = (values: FormSchema) => {
    // Do something with the form values.
    // ✅ This will be type-safe and validated.
    uploadImagePack(values)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 px-6">
        <SandwichLayout
          footer={
            <LoadableButton isLoading={isPending} isLoadingText="正在提交" type="submit">
              <PackagePlusIcon />
              确认提交
            </LoadableButton>
          }
        >
          <FormField
            control={form.control}
            name="imageLink"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像链接
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  镜像描述
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} className="font-mono" />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <TagsInput
            form={form}
            tagsPath="tags"
            label={`镜像标签`}
            description={`为镜像添加标签，以便分类和搜索`}
            customTags={allImageTags}
          />
        </SandwichLayout>
      </form>
    </Form>
  )
}
interface ImageUploadFormProps extends SandwichSheetProps {
  closeSheet: () => void
}

export function ImageUploadForm({ closeSheet, ...props }: ImageUploadFormProps) {
  return (
    <SandwichSheet {...props}>
      <ImageUploadSheetContent closeSheet={closeSheet} />
    </SandwichSheet>
  )
}
