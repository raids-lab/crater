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
import { BoxIcon, DatabaseIcon } from 'lucide-react'
import { useState } from 'react'
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
import { SandwichLayout } from '@/components/sheet/sandwich-sheet'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui-custom/alert-dialog'

import { CreateModelDownloadReq, apiCreateModelDownload } from '@/services/api/modeldownload'
import { apiGetModelDownloadLimitConfig } from '@/services/api/system-config'

import { showErrorToast } from '@/utils/toast'

import { cn } from '@/lib/utils'

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
  token: z.string().optional(),
})

interface ModelDownloadDialogProps {
  closeSheet: () => void
  defaultCategory?: 'model' | 'dataset'
}

export function ModelDownloadDialog({
  closeSheet,
  defaultCategory = 'model',
}: ModelDownloadDialogProps) {
  const queryClient = useQueryClient()
  const [pendingRequest, setPendingRequest] = useState<CreateModelDownloadReq | null>(null)

  const { data: limitConfig } = useQuery({
    queryKey: ['system-config', 'model-download-limit'],
    queryFn: () => apiGetModelDownloadLimitConfig().then((res) => res.data),
  })

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: '',
      revision: '',
      source: 'modelscope',
      category: defaultCategory,
      token: '',
    },
  })

  const source = form.watch('source')
  const categoryLabel = defaultCategory === 'dataset' ? '数据集' : '模型'

  const sourcePlaceholder =
    source === 'modelscope'
      ? '例如: qwen/Qwen2.5-Coder-7B-Instruct'
      : '例如: meta-llama/Llama-2-7b-hf'
  const revisionPlaceholder =
    source === 'modelscope'
      ? '例如: master、标签或 commit ID；留空使用默认分支'
      : '例如: main、标签或 commit ID；留空使用默认分支'

  const tokenHint =
    source === 'modelscope'
      ? '从 ModelScope 个人中心获取 SDK Token'
      : '从 HuggingFace Settings → Access Tokens 获取'

  const { mutate, status } = useMutation({
    mutationFn: (data: CreateModelDownloadReq) => apiCreateModelDownload(data),
    onSuccess: (response, variables) => {
      // 如果后端返回了消息（如"资源已存在"），显示该消息；否则根据类别显示默认消息
      const defaultMessage =
        variables.category === 'dataset' ? '已提交数据集下载任务' : '已提交模型下载任务'
      const message = response.msg || defaultMessage
      toast.success(message)
      queryClient.invalidateQueries({ queryKey: ['model-downloads'] })
      setPendingRequest(null)
      closeSheet()
      form.reset()
    },
    onError: showErrorToast,
  })

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    setPendingRequest({
      name: values.name,
      revision: values.revision || undefined,
      source: values.source,
      category: values.category,
      token: values.token?.trim() || undefined,
    })
  }

  const pendingCategoryLabel = pendingRequest?.category === 'dataset' ? '数据集' : '模型'
  const pendingSourceLabel = pendingRequest?.source === 'modelscope' ? 'ModelScope' : 'HuggingFace'
  const pendingPath = pendingRequest
    ? `public/${pendingRequest.category === 'dataset' ? 'Datasets' : 'Models'}/${pendingRequest.name}`
    : ''

  const effectiveLimitConfig = limitConfig ?? {
    enabled: true,
    maxConcurrent: 5,
    windowHours: 2,
    maxSuccessfulDownloads: 5,
    exempt: false,
  }
  const quotaHint = !effectiveLimitConfig.enabled
    ? '当前未启用下载额度限制。'
    : effectiveLimitConfig.exempt
      ? '当前账号在下载额度白名单内，不受同时下载任务数和滚动窗口成功下载数限制。'
      : `当前账号同时最多可有 ${effectiveLimitConfig.maxConcurrent} 个等待中或下载中的任务；滚动 ${effectiveLimitConfig.windowHours} 小时最多成功下载 ${effectiveLimitConfig.maxSuccessfulDownloads} 个。下载中的任务会临时预占窗口名额，失败或暂停后释放；白名单用户不受限制。`

  return (
    <>
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
                  <FormControl>
                    <div className="bg-muted/60 grid grid-cols-2 gap-1 rounded-lg p-1">
                      {(
                        [
                          { value: 'modelscope', label: 'ModelScope' },
                          { value: 'huggingface', label: 'HuggingFace' },
                        ] as const
                      ).map((opt) => (
                        <button
                          key={opt.value}
                          type="button"
                          onClick={() => field.onChange(opt.value)}
                          className={cn(
                            'rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                            field.value === opt.value
                              ? 'bg-background text-foreground shadow-sm'
                              : 'text-muted-foreground hover:text-foreground'
                          )}
                        >
                          {opt.label}
                        </button>
                      ))}
                    </div>
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="space-y-2">
              <FormLabel>类别</FormLabel>
              <div className="flex items-center gap-2">
                <span className="bg-primary/10 text-primary inline-flex items-center rounded-md px-2.5 py-1 text-sm font-medium">
                  {defaultCategory === 'dataset' ? (
                    <DatabaseIcon className="mr-1.5 size-3.5" />
                  ) : (
                    <BoxIcon className="mr-1.5 size-3.5" />
                  )}
                  {categoryLabel}
                </span>
              </div>
            </div>

            <FormField
              name="name"
              control={form.control}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{categoryLabel}名称</FormLabel>
                  <FormControl>
                    <Input placeholder={sourcePlaceholder} className="font-mono" {...field} />
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
                    <Input
                      placeholder={revisionPlaceholder}
                      autoComplete="off"
                      data-1p-ignore
                      data-lpignore="true"
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              name="token"
              control={form.control}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>访问令牌（可选）</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder="用于下载受限/私有仓库"
                      autoComplete="new-password"
                      data-1p-ignore
                      data-lpignore="true"
                      data-form-type="other"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>{tokenHint}，仅用于本次下载，不会被保存。</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="bg-muted/50 text-muted-foreground rounded-md p-3 text-xs">
              <p className="mb-1 font-semibold">提示:</p>
              <ul className="ml-4 list-disc space-y-1">
                <li>模型统一下载到公共空间的 Models/ 目录,数据集下载到 Datasets/ 目录</li>
                <li>文件会保存在对应目录下的名称子目录中</li>
                <li>多个用户下载同一资源时会共享同一份文件，不会重复创建下载任务</li>
                <li>{quotaHint}</li>
                <li>受限或私有仓库（如部分 Llama / Gemma）需填写访问令牌</li>
                <li>下载过程可能需要较长时间,请耐心等待</li>
              </ul>
            </div>
          </SandwichLayout>
        </form>
      </Form>

      <AlertDialog
        open={pendingRequest !== null}
        onOpenChange={(open) => !open && status !== 'pending' && setPendingRequest(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>确认提交{pendingCategoryLabel}下载任务</AlertDialogTitle>
            <AlertDialogDescription>
              请确认以下信息。提交后平台会在公共存储中创建或复用对应资源。{quotaHint}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingRequest && (
            <dl className="bg-muted/50 grid grid-cols-[5rem_minmax(0,1fr)] gap-x-3 gap-y-2 rounded-md p-4 text-sm">
              <dt className="text-muted-foreground">类别</dt>
              <dd>{pendingCategoryLabel}</dd>
              <dt className="text-muted-foreground">来源</dt>
              <dd>{pendingSourceLabel}</dd>
              <dt className="text-muted-foreground">名称</dt>
              <dd className="font-mono break-all">{pendingRequest.name}</dd>
              <dt className="text-muted-foreground">版本</dt>
              <dd className="font-mono break-all">{pendingRequest.revision || '默认分支'}</dd>
              <dt className="text-muted-foreground">保存路径</dt>
              <dd className="font-mono break-all">{pendingPath}</dd>
              <dt className="text-muted-foreground">访问令牌</dt>
              <dd>{pendingRequest.token ? '已填写，仅用于本次下载' : '未填写'}</dd>
            </dl>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={status === 'pending'}>返回修改</AlertDialogCancel>
            <AlertDialogAction
              disabled={status === 'pending' || pendingRequest === null}
              onClick={(event) => {
                event.preventDefault()
                if (pendingRequest) {
                  mutate(pendingRequest)
                }
              }}
            >
              {status === 'pending' ? '提交中...' : '确认下载'}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
