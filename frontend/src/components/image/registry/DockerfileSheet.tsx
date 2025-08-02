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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PackagePlusIcon } from 'lucide-react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'

import LoadableButton from '@/components/button/loadable-button'
import FormExportButton from '@/components/form/FormExportButton'
import FormImportButton from '@/components/form/FormImportButton'
import FormLabelMust from '@/components/form/FormLabelMust'
import { ImageSettingsFormCard } from '@/components/form/ImageSettingsFormCard'
import { TagsInput } from '@/components/form/TagsInput'
import { MetadataFormDockerfile } from '@/components/form/types'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'

import {
  FetchAllUniqueImageTagObjects,
  ImagePackSource,
  apiUserCreateByDockerfile,
  dockerfileImageLinkRegex,
  imageNameRegex,
  imageTagRegex,
} from '@/services/api/imagepack'

import { useImageTemplateLoader } from '@/hooks/useTemplateLoader'

import { exportToJsonString } from '@/utils/form'

import { DockerfileEditor } from './DockerfileEditor'

const dockerfileFormSchema = z.object({
  dockerfile: z.string().min(1, 'Dockerfile content is required'),
  description: z.string().min(1, '请为镜像添加描述'),
  imageName: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageNameRegex.test(v)
        } else {
          return true
        }
      },
      {
        message: '仅允许小写字母、数字、. _ -，且不能以分隔符开头/结尾',
      }
    ),
  imageTag: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          return imageTagRegex.test(v)
        } else {
          return true
        }
      },
      {
        message: '仅允许字母、数字、_ . + -，且不能以 . 或 - 开头/结尾',
      }
    )
    .refine((value) => value !== 'latest', {
      message: "镜像标签不能为: 'latest'",
    }),
  tags: z
    .array(
      z.object({
        value: z.string(),
      })
    )
    .optional(),
  imageArchs: z
    .array(
      z.object({
        value: z.string(),
      })
    )
    .min(1, '至少选择一个架构'),
})

export type DockerfileFormValues = z.infer<typeof dockerfileFormSchema>

interface DockerfileSheetContentProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

function DockerfileSheetContent({
  closeSheet,
  imagePackName = '',
  setImagePackName,
}: DockerfileSheetContentProps) {
  const queryClient = useQueryClient()

  const form = useForm<DockerfileFormValues>({
    resolver: zodResolver(dockerfileFormSchema),
    defaultValues: {
      dockerfile:
        'FROM node:14\n\nWORKDIR /app\n\nCOPY package*.json ./\n\nRUN npm install\n\nCOPY . .\n\nEXPOSE 3000\n\nCMD ["npm", "start"]',
      description: '',
      imageName: '',
      imageTag: '',
      tags: [],
      imageArchs: [{ value: 'linux/amd64' }],
    },
  })

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: DockerfileFormValues) =>
      apiUserCreateByDockerfile({
        description: values.description,
        dockerfile: values.dockerfile,
        name: values.imageName ?? '',
        tag: values.imageTag ?? '',
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormDockerfile, values),
        buildSource: ImagePackSource.Dockerfile,
        archs: values.imageArchs?.map((item) => item.value) ?? [],
      }),
    onSuccess: async () => {
      await new Promise((resolve) => setTimeout(resolve, 500)).then(() =>
        queryClient.invalidateQueries({ queryKey: ['imagepack', 'list'] })
      )
      closeSheet()
      toast.success(`镜像开始制作，请在下方列表中查看制作状态`)
    },
  })

  const onSubmit = (values: DockerfileFormValues) => {
    if (isPending) {
      toast.error('正在提交，请稍后')
      return
    }
    const matches = Array.from(values.dockerfile.matchAll(dockerfileImageLinkRegex))
    const baseImages = matches.map((match) => match[1])
    const baseImage = baseImages[0]
    if (
      baseImage.includes('jupyter') &&
      values.imageName != '' &&
      !values.imageName?.includes('jupyter')
    ) {
      toast.error('基础镜像为 Jupyter 相关镜像时，镜像名称必须包含 jupyter')
      return
    }
    if (
      baseImage.includes('nvidia') &&
      values.imageName != '' &&
      !values.imageName?.includes('nvidia')
    ) {
      toast.error('基础镜像为 NVIDIA 相关镜像时，镜像名称必须包含 nvidia')
      return
    }
    submitDockerfileSheet(values)
  }

  const { data: allImageTags } = useQuery({
    queryKey: ['allImageTags'],
    queryFn: FetchAllUniqueImageTagObjects,
  })

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormDockerfile,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton metadata={MetadataFormDockerfile} form={form} />
              <FormExportButton metadata={MetadataFormDockerfile} form={form} />
              <LoadableButton isLoading={isPending} isLoadingText="正在提交" type="submit">
                <PackagePlusIcon />
                开始制作
              </LoadableButton>
            </>
          }
        >
          <FormField
            control={form.control}
            name="description"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  描述
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <Input {...field} />
                </FormControl>
                <FormDescription>
                  关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示。
                </FormDescription>
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
          <FormField
            control={form.control}
            name="dockerfile"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Dockerfile
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <DockerfileEditor value={field.value} onChange={field.onChange} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />
          <ImageSettingsFormCard
            form={form}
            imageNamePath="imageName"
            imageTagPath="imageTag"
            imageBuildArchPath="imageArchs"
          />
        </SandwichLayout>
      </form>
    </Form>
  )
}

interface DockerfileSheetProps extends SandwichSheetProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

export function DockerfileSheet({
  closeSheet,
  imagePackName = '',
  setImagePackName,
  ...props
}: DockerfileSheetProps) {
  return (
    <SandwichSheet {...props}>
      <DockerfileSheetContent
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
        closeSheet={closeSheet}
      />
    </SandwichSheet>
  )
}
