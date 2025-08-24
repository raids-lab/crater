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
import { PlusIcon, Trash2Icon } from 'lucide-react'
import { useEffect } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { Badge } from '@/components/ui/badge'
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
import { ScrollArea } from '@/components/ui/scroll-area'
import { Separator } from '@/components/ui/separator'

import LoadableButton from '@/components/button/loadable-button'
import SandwichSheet, { SandwichLayout, SandwichSheetProps } from '@/components/sheet/SandwichSheet'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui-custom/alert-dialog'

import {
  CudaBaseImage,
  apiAddCudaBaseImage,
  apiDeleteCudaBaseImage,
  apiGetCudaBaseImages,
  cudaImageLabelRegex,
  cudaImageValueRegex,
} from '@/services/api/imagepack'

interface CudaBaseImageSheetContentProps {
  closeSheet: () => void
}

function CudaBaseImageSheetContent({ closeSheet }: CudaBaseImageSheetContentProps) {
  const queryClient = useQueryClient()

  const { data: cudaBaseImages = [], isLoading } = useQuery({
    queryKey: ['cudaBaseImages'],
    queryFn: apiGetCudaBaseImages,
    select: (res) => res.data.cudaBaseImages,
  })

  const cudaBaseImageSchema = z.object({
    label: z.string().min(1, '镜像展示标签不能为空'),
    imageLabel: z
      .string()
      .min(1, '镜像版本标签不能为空')
      .regex(
        cudaImageLabelRegex,
        '镜像版本标签格式不正确, 只能包含字母、数字、点、下划线、连字符, 长度1-128字符, 不能以点或连字符开头/结尾'
      )
      .refine(
        (value) => !cudaBaseImages.some((image) => image.imageLabel === value),
        '该镜像版本标签已存在，请使用不同的标签'
      ),
    value: z
      .string()
      .min(1, '镜像完整链接不能为空')
      .regex(
        cudaImageValueRegex,
        '镜像链接格式不正确, 请输入完整的镜像地址, 如: registry.example.com/namespace/image:tag'
      ),
  })

  type CudaBaseImageFormValues = z.infer<typeof cudaBaseImageSchema>

  const form = useForm<CudaBaseImageFormValues>({
    resolver: zodResolver(cudaBaseImageSchema),
    defaultValues: {
      label: '',
      imageLabel: '',
      value: '',
    },
  })

  // 当 cudaBaseImages 数据变化时，重新触发 imageLabel 字段的验证
  useEffect(() => {
    if (form.getValues('imageLabel')) {
      form.trigger('imageLabel')
    }
  }, [cudaBaseImages, form])

  const { mutate: addCudaBaseImage, isPending: isAdding } = useMutation({
    mutationFn: apiAddCudaBaseImage,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cudaBaseImages'] })
      form.reset()
      toast.success('CUDA Base镜像添加成功')
      closeSheet()
    },
    onError: () => {
      toast.error('添加失败')
    },
  })

  const { mutate: deleteCudaBaseImage, isPending: isDeleting } = useMutation({
    mutationFn: apiDeleteCudaBaseImage,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['cudaBaseImages'] })
      toast.success('CUDA Base镜像删除成功')
    },
    onError: () => {
      toast.error('删除失败')
    },
  })

  const onSubmit = (values: CudaBaseImageFormValues) => {
    addCudaBaseImage(values)
  }

  const handleDelete = (image: CudaBaseImage) => {
    deleteCudaBaseImage(image.id)
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SandwichLayout
          footer={
            <>
              <LoadableButton
                type="submit"
                isLoading={isAdding}
                isLoadingText="添加中..."
                className="w-full"
              >
                <PlusIcon className="mr-2 h-4 w-4" />
                添加CUDA Base镜像
              </LoadableButton>
            </>
          }
        >
          {/* 镜像列表 */}
          <div>
            <h3 className="mb-4 text-lg font-medium">当前CUDA Base镜像</h3>
            <ScrollArea className="h-[400px] rounded-md border p-4">
              {isLoading ? (
                <div className="flex h-20 items-center justify-center">
                  <div className="text-muted-foreground">加载中...</div>
                </div>
              ) : cudaBaseImages.length === 0 ? (
                <div className="flex h-20 items-center justify-center">
                  <div className="text-muted-foreground">暂无CUDA Base镜像</div>
                </div>
              ) : (
                <div className="space-y-3">
                  {cudaBaseImages.map((image) => (
                    <div
                      key={image.id}
                      className="flex items-center justify-between rounded-md border p-4"
                    >
                      <div className="flex-1 space-y-3">
                        <div className="space-y-2">
                          <div className="flex items-center gap-2">
                            <span className="text-muted-foreground text-xs font-medium">
                              显示名称：
                            </span>
                            <Badge variant="outline" className="px-2 py-1 font-medium">
                              {image.label}
                            </Badge>
                          </div>
                          <div className="flex items-center gap-2">
                            <span className="text-muted-foreground text-xs font-medium">
                              版本标签：
                            </span>
                            <Badge variant="secondary" className="px-2 py-1 font-medium">
                              {image.imageLabel}
                            </Badge>
                          </div>
                        </div>
                        <div className="flex items-start gap-2">
                          <span className="text-muted-foreground mt-0.5 shrink-0 text-xs font-medium">
                            镜像地址：
                          </span>
                          <div className="text-muted-foreground font-mono text-xs break-all">
                            {image.value}
                          </div>
                        </div>
                      </div>
                      <AlertDialog>
                        <AlertDialogTrigger asChild>
                          <Button variant="ghost" size="sm">
                            <Trash2Icon className="text-destructive h-4 w-4" />
                          </Button>
                        </AlertDialogTrigger>
                        <AlertDialogContent>
                          <AlertDialogHeader>
                            <AlertDialogTitle>删除CUDA Base镜像</AlertDialogTitle>
                            <AlertDialogDescription>
                              确定要删除镜像「{image.label}」吗？此操作无法撤销。
                            </AlertDialogDescription>
                          </AlertDialogHeader>
                          <AlertDialogFooter>
                            <AlertDialogCancel>取消</AlertDialogCancel>
                            <AlertDialogAction
                              variant="destructive"
                              onClick={() => handleDelete(image)}
                              disabled={isDeleting}
                            >
                              删除
                            </AlertDialogAction>
                          </AlertDialogFooter>
                        </AlertDialogContent>
                      </AlertDialog>
                    </div>
                  ))}
                </div>
              )}
            </ScrollArea>
          </div>

          <Separator />

          {/* 添加新镜像表单 */}
          <div>
            <h3 className="mb-4 text-lg font-medium">添加新的CUDA Base镜像</h3>
            <div className="space-y-4">
              <FormField
                control={form.control}
                name="label"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>镜像展示标签</FormLabel>
                    <FormControl>
                      <Input placeholder="例如: CUDA 12.8.1" {...field} />
                    </FormControl>
                    <FormDescription>用于在界面上显示的镜像标签</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="imageLabel"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>镜像版本标签</FormLabel>
                    <FormControl>
                      <Input placeholder="例如: cu12.8.1" {...field} />
                    </FormControl>
                    <FormDescription>
                      用于代码中标识镜像的版本标签，必须唯一且不能与现有标签重复
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="value"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>镜像完整链接</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="例如: harbor.raids-lab.cn/nvidia/cuda:12.8.1-cudnn-devel-ubuntu22.04"
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>完整的镜像地址</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </div>
        </SandwichLayout>
      </form>
    </Form>
  )
}

interface CudaBaseImageSheetProps extends SandwichSheetProps {
  closeSheet: () => void
}

export function CudaBaseImageSheet({ closeSheet, ...props }: CudaBaseImageSheetProps) {
  return (
    <SandwichSheet {...props}>
      <CudaBaseImageSheetContent closeSheet={closeSheet} />
    </SandwichSheet>
  )
}
