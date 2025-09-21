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
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { Share2 } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
// 修改这一行添加 useEffect
import { useForm } from 'react-hook-form'
import { FieldValues, UseFormReturn } from 'react-hook-form'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
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
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'

import TooltipButton from '@/components/button/tooltip-button'
import FormLabelMust from '@/components/form/FormLabelMust'
import SandwichSheet, { SandwichLayout } from '@/components/sheet/SandwichSheet'

import { getJobTemplate } from '@/services/api/jobtemplate'
import { createJobTemplate, updateJobTemplate } from '@/services/api/jobtemplate'

import { atomUserInfo } from '@/utils/store'

export type PublishSearch = {
  fromTemplate?: number
  fromJob?: string
}

export const publishValidateSearch = (search: Record<string, unknown>): PublishSearch => {
  return {
    fromTemplate: Number(search.fromTemplate) || undefined,
    fromJob: (search.fromJob as string) || undefined,
  }
}

// Define the form schema with Zod
const formSchema = z.object({
  name: z.string().min(1, { message: 'Name is required' }),
  description: z.string().optional(),
  document: z.string().optional(),
})

type FormValues = z.infer<typeof formSchema>

interface PublishConfigFormProps<T extends FieldValues> {
  config: object // The configuration object to be published
  configform: UseFormReturn<T>
  fromTemplate?: number
}

export function PublishConfigForm<T extends FieldValues>({
  config,
  configform,
  fromTemplate,
}: PublishConfigFormProps<T>) {
  const queryClient = useQueryClient()

  const user = useAtomValue(atomUserInfo)

  const [isOpen, setIsOpen] = useState(false)
  const { data: templateData } = useQuery({
    queryKey: ['jobTemplate', fromTemplate],
    queryFn: () => getJobTemplate(Number(fromTemplate)),
    enabled: !!fromTemplate,
    select: (data) => data?.data,
  })

  const isUpdate = useMemo(() => {
    return fromTemplate && templateData?.userInfo.username === user?.name
  }, [fromTemplate, templateData, user?.name])

  const data = configform?.watch()

  // 生成字符串 C（带格式化缩进）
  const formattedConfig = useMemo(() => {
    // 解析为 JSON 对象
    const objconfig = typeof config === 'string' ? JSON.parse(config) : config
    // 合并到新对象
    const objcombinedConfig = {
      ...objconfig,
      data, // 将 B 作为 data 属性
    }
    const json = JSON.stringify(objcombinedConfig, null, 2)
    // 特殊处理，将 /home/username 替换为 /home/${username}
    if (json.includes(`/home/${user?.name}`)) {
      return json.replace(new RegExp(`/home/${user?.name}`, 'g'), `/home/\${CRATER_USERNAME}`)
    }
    return json
  }, [config, data, user?.name])

  const { mutate: createTemplate } = useMutation({
    mutationFn: (values: FormValues) =>
      isUpdate
        ? updateJobTemplate({
            id: Number(fromTemplate),
            name: values.name,
            describe: values.description || '',
            template: formattedConfig,
            document: values.document || '',
          })
        : createJobTemplate({
            name: values.name,
            describe: values.description || '',
            template: formattedConfig,
            document: values.document || '',
          }),
    onSuccess: async (_, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ['data', 'jobtemplate'],
      })
      if (isUpdate) {
        toast.success(`Job template ${name} updated successfully`)
      } else {
        toast.success(`Job template ${name} created successfully`)
      }
      setIsOpen(false)
    },
  })
  // Initialize react-hook-form
  // 提取默认值逻辑，避免重复的三元表达式
  const defaultValues =
    isUpdate && templateData
      ? {
          name: templateData.name || '',
          description: templateData.describe || '',
          document: templateData.document || '',
        }
      : {
          name: '',
          description: '',
          document: '',
        }

  const form = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues,
  })
  useEffect(() => {
    if (templateData) {
      form.reset({
        name: templateData.name || '',
        description: templateData.describe || '',
        document: templateData.document || '',
      })
    }
  }, [templateData, form])

  const handleSubmit = form.handleSubmit((data) => {
    createTemplate(data)
  })

  return (
    <SandwichSheet
      isOpen={isOpen}
      onOpenChange={setIsOpen}
      title={isUpdate ? '更新模板' : '创建模板'}
      description={isUpdate ? '更新此模板' : '将此配置创建成模板分享给其他用户'}
      trigger={
        <TooltipButton
          variant="outline"
          tooltipContent={isUpdate ? '更新已分享的模板' : '与平台用户共享，便于快速复现实验'}
        >
          <Share2 />
          {isUpdate ? '更新模板' : '创建模板'}
        </TooltipButton>
      }
      className="sm:max-w-2xl"
    >
      <Form {...form}>
        <form onSubmit={handleSubmit}>
          <SandwichLayout
            footer={
              <Button
                type="button"
                onClick={async () => {
                  const isValid = await configform.trigger()
                  if (isValid) {
                    handleSubmit()
                  } else {
                    toast.error('请检查作业配置是否填写正确')
                  }
                }}
              >
                <Share2 />
                {isUpdate ? '更新模板' : '创建模板'}
              </Button>
            }
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    模板名称
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormDescription>
                    给此模板文件起一个名字，将作为配置文件的标题显示
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>描述</FormLabel>
                  <FormControl>
                    <Textarea {...field} />
                  </FormControl>
                  <FormDescription>
                    关于此配置的简短描述，如包含的软件版本、用途等，将作为配置标识显示
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="document"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>作业模板详细文档</FormLabel>
                  <FormControl>
                    <Textarea {...field} className="min-h-[200px]" />
                  </FormControl>
                  <FormDescription>关于模板的详细使用说明，支持 Markdown 格式</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="space-y-2">
              <Label htmlFor="config-preview">配置预览</Label>
              <Textarea value={formattedConfig} className="max-w-full font-mono" />
              <Label className="text-muted-foreground font-normal">
                请确保您不会在此配置中包含任何敏感信息，如密码、密钥等
                {/* <span className="font-mono">/home/{user?.name}</span> 将被替换为
                <span className="font-mono">
                  /home/${"{"}CRATER_USERNAME{"}"}
                </span>
                ，以保护您的隐私。 */}
              </Label>
            </div>
          </SandwichLayout>
        </form>
      </Form>
    </SandwichSheet>
  )
}
