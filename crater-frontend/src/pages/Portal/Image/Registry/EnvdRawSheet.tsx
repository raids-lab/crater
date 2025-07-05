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
import { z } from 'zod'
import { toast } from 'sonner'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'
import LoadableButton from '@/components/button/LoadableButton'
import { PackagePlusIcon } from 'lucide-react'
import FormImportButton from '@/components/form/FormImportButton'
import FormExportButton from '@/components/form/FormExportButton'
import { MetadataFormEnvdRaw } from '@/components/form/types'
import { Input } from '@/components/ui/input'
import {
  apiUserCreateByEnvd,
  FetchAllUniqueImageTagObjects,
  imageNameRegex,
  ImagePackSource,
  imageTagRegex,
} from '@/services/api/imagepack'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import FormLabelMust from '@/components/form/FormLabelMust'
import { DockerfileEditor } from './DockerfileEditor'
import { TagsInput } from '@/components/form/TagsInput'
import { exportToJsonString } from '@/utils/form'
import { useImageTemplateLoader } from '@/hooks/useTemplateLoader'

const envdRawFormSchema = z.object({
  envdScript: z.string().min(1, 'Envd script content is required'),
  description: z.string().min(1, '请为镜像添加描述'),
  imageName: z
    .string()
    .min(1, '镜像名不能为空')
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
    )
    .refine((v) => !v || v.includes('envd'), {
      message: "名称需包含 'envd' ",
    }),
  imageTag: z
    .string()
    .min(1, '镜像标签不能为空')
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
})

export type EnvdRawFormValues = z.infer<typeof envdRawFormSchema>

interface EnvdRawSheetContentProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

function EnvdRawSheetContent({
  closeSheet,
  imagePackName = '',
  setImagePackName,
}: EnvdRawSheetContentProps) {
  const queryClient = useQueryClient()

  const form = useForm<EnvdRawFormValues>({
    resolver: zodResolver(envdRawFormSchema),
    defaultValues: {
      tags: [],
      envdScript: `# syntax=v1

def build():
    base(image="crater-harbor.act.buaa.edu.cn/nvidia/cuda:12.8.1-cudnn-devel-ubuntu22.04",dev=True)
    install.python(version="3.10")
    install.apt_packages(["openssh-server", "build-essential", "iputils-ping", "net-tools", "htop"])
    config.pip_index(url = "https://pypi.tuna.tsinghua.edu.cn/simple")
    config.jupyter()`,
      description: '',
      imageName: '',
      imageTag: '',
    },
  })

  const { mutate: submitEnvdRawSheet, isPending } = useMutation({
    mutationFn: (values: EnvdRawFormValues) =>
      apiUserCreateByEnvd({
        description: values.description,
        envd: values.envdScript,
        name: values.imageName ?? '',
        tag: values.imageTag ?? '',
        python: '',
        base: '',
        tags: values.tags?.map((item) => item.value) ?? [],
        template: exportToJsonString(MetadataFormEnvdRaw, values),
        buildSource: ImagePackSource.EnvdRaw,
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

  const onSubmit = (values: EnvdRawFormValues) => {
    submitEnvdRawSheet(values)
  }

  useImageTemplateLoader({
    form: form,
    metadata: MetadataFormEnvdRaw,
    imagePackName: imagePackName,
    setImagePackName: setImagePackName,
  })

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <FormImportButton metadata={MetadataFormEnvdRaw} form={form} />
              <FormExportButton metadata={MetadataFormEnvdRaw} form={form} />

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
          <div className="grid grid-cols-2 gap-4">
            <FormField
              control={form.control}
              name="imageName"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像名
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormDescription>自定义镜像名</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="imageTag"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    镜像标签
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormDescription>自定义镜像标签</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <FormField
            control={form.control}
            name="envdScript"
            render={({ field }) => (
              <FormItem>
                <FormLabel>
                  Envd 脚本
                  <FormLabelMust />
                </FormLabel>
                <FormControl>
                  <DockerfileEditor
                    value={field.value}
                    onChange={field.onChange}
                    language="python"
                  />
                </FormControl>
                <FormDescription>
                  直接编写 envd 构建脚本，以 # syntax=v1 开头，包含 build() 函数
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SandwichLayout>
      </form>
    </Form>
  )
}

interface EnvdRawSheetProps extends SandwichSheetProps {
  closeSheet: () => void
  imagePackName?: string
  setImagePackName: (imagePackName: string) => void
}

export function EnvdRawSheet({
  closeSheet,
  imagePackName = '',
  setImagePackName,
  ...props
}: EnvdRawSheetProps) {
  return (
    <SandwichSheet {...props}>
      <EnvdRawSheetContent
        imagePackName={imagePackName}
        setImagePackName={setImagePackName}
        closeSheet={closeSheet}
      />
    </SandwichSheet>
  )
}
