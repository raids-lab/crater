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
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import LoadableButton from '@/components/button/loadable-button'
import { SandwichLayout } from '@/components/sheet/sandwich-sheet'

import { CreateModelDownloadReq, apiCreateModelDownload } from '@/services/api/modeldownload'

const formSchema = z.object({
  name: z
    .string()
    .min(1, { message: '请输入名称' })
    .regex(/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/, {
      message: '格式应为: owner/name,如 qwen/Qwen2.5-Coder-7B-Instruct',
    }),
  revision: z.string().optional(),
  source: z.enum(['modelscope', 'huggingface']),
  category: z.enum(['model', 'dataset']),
})

interface ModelDownloadDialogProps {
  closeSheet: () => void
}

export function ModelDownloadDialog({ closeSheet }: ModelDownloadDialogProps) {
  const queryClient = useQueryClient()

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      revision: '',
      source: 'modelscope',
      category: 'model',
    },
  })

  const source = form.watch('source')

  const sourcePlaceholder =
    source === 'modelscope'
      ? '例如: qwen/Qwen2.5-Coder-7B-Instruct'
      : '例如: meta-llama/Llama-2-7b-hf'

  const { mutate, status } = useMutation({
    mutationFn: (data: CreateModelDownloadReq) => apiCreateModelDownload(data),
    onSuccess: (response) => {
      // 如果后端返回了消息（如"资源已存在"），显示该消息；否则显示默认消息
      const message = response.msg || '已提交模型下载任务'
      toast.success(message)
      queryClient.invalidateQueries({ queryKey: ['model-downloads'] })
      closeSheet()
      form.reset()
    },
    onError: (error: unknown) => {
      const message =
        error && typeof error === 'object' && 'response' in error
          ? ((error as { response?: { data?: { msg?: string } } }).response?.data?.msg ??
            '提交失败，请重试')
          : '提交失败，请重试'
      toast.error(message)
    },
  })

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    mutate({
      name: values.name,
      revision: values.revision || undefined,
      source: values.source,
      category: values.category,
    })
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <LoadableButton
              type="submit"
              isLoading={status === 'pending'}
              isLoadingText="提交中..."
            >
              开始下载
            </LoadableButton>
          }
        >
          <FormField
            name="source"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>下载来源</FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="选择下载来源" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="modelscope">ModelScope</SelectItem>
                    <SelectItem value="huggingface">HuggingFace</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            name="category"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>类别</FormLabel>
                <Select onValueChange={field.onChange} defaultValue={field.value}>
                  <FormControl>
                    <SelectTrigger>
                      <SelectValue placeholder="选择类别" />
                    </SelectTrigger>
                  </FormControl>
                  <SelectContent>
                    <SelectItem value="model">模型</SelectItem>
                    <SelectItem value="dataset">数据集</SelectItem>
                  </SelectContent>
                </Select>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            name="name"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>名称</FormLabel>
                <FormControl>
                  <Input placeholder={sourcePlaceholder} {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <FormField
            name="revision"
            control={form.control}
            render={({ field }) => (
              <FormItem>
                <FormLabel>版本（可选）</FormLabel>
                <FormControl>
                  <Input placeholder="例如: main, v1.0.0 或 commit ID" {...field} />
                </FormControl>
                <FormMessage />
              </FormItem>
            )}
          />

          <div className="bg-muted/50 text-muted-foreground rounded-md p-3 text-xs">
            <p className="mb-1 font-semibold">提示:</p>
            <ul className="ml-4 list-disc space-y-1">
              <li>模型统一下载到公共空间的 Models/ 目录,数据集下载到 Datasets/ 目录</li>
              <li>文件会保存在对应目录下的名称子目录中</li>
              <li>多个用户下载同一资源时会共享同一份文件</li>
              <li>下载过程可能需要较长时间,请耐心等待</li>
            </ul>
          </div>
        </SandwichLayout>
      </form>
    </Form>
  )
}
