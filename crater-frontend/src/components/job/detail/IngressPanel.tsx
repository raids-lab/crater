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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { ExternalLink, Plus, Trash2 } from 'lucide-react'
import { GridIcon } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import * as z from 'zod'

import { Button } from '@/components/ui/button'
import { Dialog, DialogContent, DialogHeader, DialogTitle } from '@/components/ui/dialog'
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

import DocsButton from '@/components/button/docs-button'
import LoadableButton from '@/components/button/loadable-button'
import TooltipButton from '@/components/button/tooltip-button'
import { PodNamespacedName } from '@/components/codeblock/PodContainerDialog'
import FormLabelMust from '@/components/form/FormLabelMust'
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

import { apiCreatePodIngress, apiDeletePodIngress, apiGetPodIngresses } from '@/services/api/tool'
import { PodIngressMgr } from '@/services/api/tool'

import { configUrlHostAtom } from '@/utils/store/config'

const ingressFormSchema = z.object({
  name: z
    .string()
    .min(1)
    .max(20)
    .regex(/^[a-z]+$/, {
      message: '只能包含小写字母',
    }),
  port: z.number().int().positive(),
})

interface IngressPanelProps {
  namespacedName: PodNamespacedName
  jobName: string
}

export const IngressPanel = ({ namespacedName }: IngressPanelProps) => {
  const [isEditIngressDialogOpen, setIsEditIngressDialogOpen] = useState(false)
  const host = useAtomValue(configUrlHostAtom)

  const ingressForm = useForm<PodIngressMgr>({
    resolver: zodResolver(ingressFormSchema),
    defaultValues: {
      name: '',
      port: 0,
    },
  })

  const { namespace, name } = namespacedName || {}
  const {
    data: ingressList,
    isLoading,
    refetch: refetchIngresses,
  } = useQuery({
    queryKey: ['ingresses', namespace, name],
    queryFn: async () => {
      if (!namespace || !name) return []
      const response = await apiGetPodIngresses(namespace, name)
      return response.data.ingresses
    },
    enabled: !!namespace && !!name,
  })

  const { mutate: createIngressMutation, isPending: isCreating } = useMutation({
    mutationFn: (data: PodIngressMgr) =>
      apiCreatePodIngress(namespacedName.namespace, namespacedName.name, data),
    onSuccess: () => {
      void refetchIngresses()
      toast.success('添加成功')
      setIsEditIngressDialogOpen(false)
    },
    onError: () => {
      toast.error('添加失败')
    },
  })

  const { mutate: deleteIngressMutation, isPending: isDeleting } = useMutation({
    mutationFn: (data: PodIngressMgr) =>
      apiDeletePodIngress(namespacedName.namespace, namespacedName.name, data),
    onSuccess: () => {
      void refetchIngresses()
      toast.success('删除成功')
    },
    onError: () => {
      toast.error('删除失败')
    },
  })

  const handleAddIngress = () => {
    ingressForm.reset({ name: '', port: 0 })
    setIsEditIngressDialogOpen(true)
  }

  const onSubmitIngress = (data: PodIngressMgr) => {
    if (namespacedName) {
      createIngressMutation(data)
    }
  }

  const handleDeleteIngress = (data: PodIngressMgr) => {
    if (namespacedName) {
      deleteIngressMutation(data)
    }
  }

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        {isLoading ? (
          <div className="text-muted-foreground py-6 text-center">加载中...</div>
        ) : !ingressList || ingressList.length === 0 ? (
          <div className="text-muted-foreground text-center">
            <div className="flex flex-col items-center justify-center pt-8">
              <div className="bg-muted mb-4 rounded-full p-3">
                <GridIcon className="h-6 w-6" />
              </div>
              <p className="select-none">暂无数据</p>
            </div>
          </div>
        ) : (
          ingressList.map((ingress) => (
            <div
              key={ingress.name}
              className="bg-secondary flex items-center space-x-2 rounded-md p-3"
            >
              <div className="ml-2 flex grow flex-col items-start justify-start gap-0.5">
                <p>{ingress.name}</p>
                <div className="text-muted-foreground flex flex-row text-xs">
                  {ingress.port} → {ingress.prefix}
                </div>
              </div>
              <TooltipButton
                variant="ghost"
                size="icon"
                className="hover:text-primary"
                onClick={() => {
                  window.open(ingress.prefix, '_blank')
                }}
                tooltipContent="访问链接"
              >
                <ExternalLink className="size-4" />
              </TooltipButton>
              <AlertDialog>
                <AlertDialogTrigger asChild>
                  <TooltipButton
                    variant="ghost"
                    size="icon"
                    className="hover:text-destructive"
                    tooltipContent="删除"
                    disabled={ingress.port === 8888 || isDeleting}
                  >
                    <Trash2 className="size-4" />
                  </TooltipButton>
                </AlertDialogTrigger>
                {ingress.port !== 8888 && (
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除外部访问规则</AlertDialogTitle>
                      <AlertDialogDescription>
                        外部访问规则「{ingress.name}」<br />
                        {ingress.port} → {host}
                        {ingress.prefix}
                        <br />
                        将被删除，请谨慎操作。
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={() => handleDeleteIngress(ingress)}
                        disabled={isDeleting}
                      >
                        {isDeleting ? '删除中...' : '删除'}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                )}
              </AlertDialog>
            </div>
          ))
        )}
      </div>

      <div className="flex justify-end gap-2">
        <DocsButton title={'帮助文档'} url={`toolbox/external-access/ingress-rule`} />
        <Button onClick={handleAddIngress} disabled={isCreating}>
          <Plus className="mr-2 size-4" />
          添加 Ingress 规则
        </Button>
      </div>

      <Dialog open={isEditIngressDialogOpen} onOpenChange={setIsEditIngressDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加 Ingress 规则</DialogTitle>
          </DialogHeader>
          <Form {...ingressForm}>
            <form
              onSubmit={(e) => {
                void ingressForm.handleSubmit(onSubmitIngress)(e)
              }}
              className="space-y-4"
            >
              <FormField
                control={ingressForm.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      规则名称
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>只能包含小写字母，不超过20个字符。</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={ingressForm.control}
                name="port"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      容器端口
                      <FormLabelMust />
                    </FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        type="text"
                        onChange={(e) => {
                          const value = e.target.value
                          if (value === '') {
                            field.onChange(null)
                          } else {
                            const parsed = parseInt(value, 10)
                            if (!isNaN(parsed)) {
                              field.onChange(parsed)
                            }
                          }
                        }}
                        value={field.value ?? ''}
                      />
                    </FormControl>
                    <FormDescription>请输入容器内需要转发的端口号。</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <LoadableButton isLoading={isCreating} isLoadingText="保存中..." type="submit">
                保存
              </LoadableButton>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
