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

import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Textarea } from '@/components/ui/textarea'
import { z } from 'zod'
import { toast } from 'sonner'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'
import LoadableButton from '@/components/button/LoadableButton'
import { PackagePlusIcon } from 'lucide-react'
import FormImportButton from '@/components/form/FormImportButton'
import FormExportButton from '@/components/form/FormExportButton'
import { MetadataFormPipApt } from '@/components/form/types'
import FormLabelMust from '@/components/form/FormLabelMust'
import { JobType } from '@/services/api/vcjob'
import Combobox from '@/components/form/Combobox'
import ImageItem from '@/components/form/ImageItem'
import useImageQuery from '@/hooks/query/useImageQuery'
import { Input } from '@/components/ui/input'
import {
  apiUserCreateKaniko,
  FetchAllUniqueImageTagObjects,
  imageNameRegex,
  ImagePackSource,
  imageTagRegex,
} from '@/services/api/imagepack'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ImageSettingsFormCard } from '@/components/form/ImageSettingsFormCard'
import { TagsInput } from '@/components/form/TagsInput'
import { exportToJsonString } from '@/utils/form'
import { useImageTemplateLoader } from '@/hooks/useTemplateLoader'

const pipAptFormSchema = z.object({
  baseImage: z.string().min(1, '基础镜像是必填项'),
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
  requirements: z
    .string()
    .optional()
    .refine(
      (v) => {
        if (v) {
          try {
            v.split('\n').forEach((line) => {
              if (line.trim().startsWith('#')) {
                return
              }
              if (!line.trim()) {
                return
              }
              // relation:
              // ==：等于
              // >：大于版本
              // >=：大于等于
              // <：小于版本
              // <=：小于等于版本
              // ~=：兼容版本

              // 基于 relation 将每一行的内容进行分割，分割后的内容为：name, relation, version
              // 可以只有 name
              const regex = /([a-zA-Z0-9_]+)([<>=!~]+)?([a-zA-Z0-9_.]+)?/
              const match = line.match(regex)
              if (!match) {
                throw new Error('Invalid requirement format')
              }
              if (match.length < 2) {
                throw new Error('Invalid requirement format')
              }
            })
          } catch {
            return false
          }
        }
        return true
      },
      {
        message: 'requirements.txt 文件格式无效',
      }
    ),
  aptPackages: z.string().optional(),
  description: z.string().min(1, '请为镜像添加描述'),
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
    .optional(),
})

export type PipAptFormValues = z.infer<typeof pipAptFormSchema>

interface PipAptSheetContentProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

function PipAptSheetContent({
  closeSheet,
  imagePackName = '',
  setImagePackName,
}: PipAptSheetContentProps) {
  const queryClient = useQueryClient()

  const form = useForm<PipAptFormValues>({
    resolver: zodResolver(pipAptFormSchema),
    defaultValues: {
      baseImage: '',
      imageName: '',
      imageTag: '',
      requirements: '',
      aptPackages: '',
      description: '',
      tags: [],
      imageArchs: [{ value: 'linux/amd64' }],
    },
  })

  const { mutate: submitDockerfileSheet, isPending } = useMutation({
    mutationFn: (values: PipAptFormValues) =>
      apiUserCreateKaniko({
        description: values.description,
        image: values.baseImage,
        requirements: values.requirements ?? '',
        packages: values.aptPackages ?? '',
        name: values.imageName ?? '',
        tag: values.imageTag ?? '',
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormPipApt, values),
        buildSource: ImagePackSource.PipApt,
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

  const { data: allImageTags } = useQuery({
    queryKey: ['allImageTags'],
    queryFn: FetchAllUniqueImageTagObjects,
  })

  const onSubmit = (values: PipAptFormValues) => {
    if (isPending) {
      toast.error('正在提交，请稍后')
      return
    }
    if (
      values.baseImage.includes('jupyter') &&
      values.imageName != '' &&
      !values.imageName?.includes('jupyter')
    ) {
      toast.error('基础镜像为 Jupyter 相关镜像时，镜像名称必须包含 jupyter')
      return
    }
    if (
      values.baseImage.includes('envd') &&
      values.imageName != '' &&
      !values.imageName?.includes('envd')
    ) {
      toast.error('基础镜像为 envd 方式构建的镜像时，镜像名称必须包含 envd')
      return
    }
    if (
      values.baseImage.includes('nvidia') &&
      values.imageName != '' &&
      !values.imageName?.includes('nvidia')
    ) {
      toast.error('基础镜像为 NVIDIA 相关镜像时，镜像名称必须包含 nvidia')
      return
    }
    submitDockerfileSheet(values)
  }
  const { data: images } = useImageQuery(JobType.Jupyter)

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormPipApt,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton metadata={MetadataFormPipApt} form={form} />
              <FormExportButton metadata={MetadataFormPipApt} form={form} />
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
                  关于此镜像的简短描述，如包含的软件版本、用途等，将作为镜像标识显示
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="baseImage"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  现有镜像
                  <FormLabelMust />
                </FormLabel>
                <FormControl autoFocus={true}>
                  <Combobox
                    items={images ?? []}
                    current={field.value}
                    handleSelect={(value) => field.onChange(value)}
                    renderLabel={(item) => <ImageItem item={item} />}
                    formTitle="镜像"
                  />
                </FormControl>
                <FormDescription>选择一个带有所需 CUDA 和 Python 版本的基础镜像</FormDescription>
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
            name="aptPackages"
            render={({ field }) => (
              <FormItem>
                <FormLabel>APT Packages</FormLabel>
                <FormControl>
                  <Textarea placeholder="git curl" className="h-24 font-mono" {...field} />
                </FormControl>
                <FormDescription>输入要安装的 APT 包，使用空格分隔多个包</FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
          <FormField
            control={form.control}
            name="requirements"
            render={({ field }) => (
              <FormItem>
                <FormLabel>Python 依赖</FormLabel>
                <FormControl>
                  <Textarea
                    placeholder={`transformers>=4.46.3
diffusers==0.31.0`}
                    className="h-24 font-mono"
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  请粘贴 requirements.txt 文件的内容，以便安装所需的 Python 包
                </FormDescription>
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

export function PipAptSheet({
  closeSheet,
  imagePackName = '',
  setImagePackName,
  ...props
}: DockerfileSheetProps) {
  return (
    <SandwichSheet {...props}>
      <PipAptSheetContent
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
        closeSheet={closeSheet}
      />
    </SandwichSheet>
  )
}
