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

import { CopyButton } from '@/components/button/copy-button'
import DocsButton from '@/components/button/docs-button'
import LoadableButton from '@/components/button/loadable-button'
import TooltipButton from '@/components/button/tooltip-button'
import { NamespacedName } from '@/components/codeblock/pod-container-dialog'
import FormLabelMust from '@/components/form/form-label-must'
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

import { apiCreatePodNodeport, apiDeletePodNodeport, apiGetPodNodeports } from '@/services/api/tool'
import { PodNodeportMgr } from '@/services/api/tool'

const nodeportFormSchema = z.object({
  name: z
    .string()
    .min(1)
    .max(20)
    .regex(/^[a-z]+$/, {
      message: '只能包含小写字母',
    }),
  containerPort: z.number().int().positive(),
})

interface NodeportPanelProps {
  namespacedName: NamespacedName
  userName: string
}

export function NodeportPanel({ namespacedName }: NodeportPanelProps) {
  const [isEditNodeportDialogOpen, setIsEditNodeportDialogOpen] = useState(false)

  const nodeportForm = useForm<PodNodeportMgr>({
    resolver: zodResolver(nodeportFormSchema),
    defaultValues: {
      name: '',
      containerPort: 0,
    },
  })

  const fetchNodeports = async () => {
    if (!namespacedName) return []
    const response = await apiGetPodNodeports(namespacedName.namespace, namespacedName.name)
    return response.data.nodeports
  }

  const { data: nodeportList = [], refetch: refetchNodeports } = useQuery({
    queryKey: ['fetchNodeports', namespacedName?.namespace, namespacedName?.name],
    queryFn: fetchNodeports,
    enabled: !!namespacedName,
  })

  const { mutate: createNodeportMutation, isPending: isCreating } = useMutation({
    mutationFn: (data: PodNodeportMgr) =>
      apiCreatePodNodeport(namespacedName!.namespace, namespacedName!.name, data),
    onSuccess: () => {
      void refetchNodeports()
      toast.success('添加成功')
      setIsEditNodeportDialogOpen(false)
    },
    onError: () => {
      toast.error('添加失败')
    },
  })

  const { mutate: deleteNodeportMutation, isPending: isDeleting } = useMutation({
    mutationFn: (data: PodNodeportMgr) =>
      apiDeletePodNodeport(namespacedName!.namespace, namespacedName!.name, data),
    onSuccess: () => {
      void refetchNodeports()
      toast.success('删除成功')
    },
    onError: () => {
      toast.error('删除失败')
    },
  })

  const handleAddNodeport = () => {
    nodeportForm.reset({ name: '', containerPort: 0 })
    setIsEditNodeportDialogOpen(true)
  }

  const onSubmitNodeport = (data: PodNodeportMgr) => {
    if (namespacedName) {
      createNodeportMutation(data)
    }
  }

  const handleDeleteNodeport = (data: PodNodeportMgr) => {
    if (namespacedName) {
      deleteNodeportMutation(data)
    }
  }

  return (
    <>
      <div className="space-y-4">
        <div className="space-y-2">
          {!nodeportList || nodeportList.length === 0 ? (
            <div className="text-muted-foreground text-center">
              <div className="flex flex-col items-center justify-center pt-8">
                <div className="bg-muted mb-4 rounded-full p-3">
                  <GridIcon className="h-6 w-6" />
                </div>
                <p className="select-none">暂无数据</p>
              </div>
            </div>
          ) : (
            nodeportList.map((nodeport) => (
              <div
                key={nodeport.name}
                className="bg-secondary flex items-center space-x-2 rounded p-3"
              >
                <div className="ml-2 flex grow flex-col items-start justify-start gap-0.5">
                  <p>{nodeport.name}</p>
                  <div className="text-muted-foreground flex flex-row text-xs">
                    {nodeport.containerPort} → {nodeport.address}:{nodeport.nodePort}
                  </div>
                </div>

                <TooltipButton
                  variant="ghost"
                  size="icon"
                  className="hover:text-primary"
                  onClick={() => {
                    const url = `http://${nodeport.address}:${nodeport.nodePort}`
                    window.open(url, '_blank')
                  }}
                  tooltipContent="访问链接"
                >
                  <ExternalLink className="size-4" />
                </TooltipButton>
                <CopyButton content={`${nodeport.address}:${nodeport.nodePort}`} />
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <TooltipButton
                      variant="ghost"
                      size="icon"
                      className="hover:text-destructive"
                      tooltipContent="删除"
                      disabled={isDeleting}
                    >
                      <Trash2 className="size-4" />
                    </TooltipButton>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>删除外部访问规则</AlertDialogTitle>
                      <AlertDialogDescription>
                        外部访问规则「{nodeport.name}」<br />
                        {nodeport.containerPort} → {nodeport.address}:{nodeport.nodePort}
                        <br />
                        将被删除，请谨慎操作。
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>取消</AlertDialogCancel>
                      <AlertDialogAction
                        variant="destructive"
                        onClick={() => handleDeleteNodeport(nodeport)}
                        disabled={isDeleting}
                      >
                        {isDeleting ? '删除中...' : '删除'}
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            ))
          )}
        </div>
        <div className="flex justify-end gap-2">
          <DocsButton title={'帮助文档'} url={`toolbox/external-access/nodeport-rule`} />
          <Button onClick={handleAddNodeport} disabled={isCreating}>
            <Plus className="mr-2 size-4" />
            添加 NodePort 规则
          </Button>
        </div>
      </div>
      <Dialog open={isEditNodeportDialogOpen} onOpenChange={setIsEditNodeportDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加 NodePort 规则</DialogTitle>
          </DialogHeader>
          <Form {...nodeportForm}>
            <form
              onSubmit={(e) => {
                void nodeportForm.handleSubmit(onSubmitNodeport)(e)
              }}
              className="space-y-4"
            >
              <FormField
                control={nodeportForm.control}
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
                    <FormDescription>请为 NodePort 规则命名，不超过20个字符。</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={nodeportForm.control}
                name="containerPort"
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
                    <FormDescription>请输入容器内需要使用的端口号。</FormDescription>
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
    </>
  )
}
