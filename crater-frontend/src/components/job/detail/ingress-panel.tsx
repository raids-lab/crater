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
import { Plus } from 'lucide-react'
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
import { PodNamespacedName } from '@/components/codeblock/pod-container-dialog'
import FormLabelMust from '@/components/form/form-label-must'

import {
  PodIngress,
  apiCreatePodIngress,
  apiDeletePodIngress,
  apiGetPodIngresses,
} from '@/services/api/tool'
import { PodIngressMgr } from '@/services/api/tool'

import { ExternalAccessItem, ExternalAccessList } from './external-access-list'

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

  const onSubmitIngress = (data: PodIngress) => {
    if (namespacedName) {
      createIngressMutation(data)
    }
  }

  const handleDeleteIngress = (data: PodIngress) => {
    if (namespacedName) {
      deleteIngressMutation(data)
    }
  }

  const renderIngressItem = (ingress: PodIngress) => {
    return (
      <ExternalAccessItem
        key={ingress.name}
        name={ingress.name}
        port={ingress.port}
        url={ingress.prefix}
        isDeleting={isDeleting}
        handleDeleteItem={() => handleDeleteIngress(ingress)}
      />
    )
  }

  return (
    <>
      <ExternalAccessList
        items={ingressList || []}
        renderItem={renderIngressItem}
        isLoading={isLoading}
        docsButton={<DocsButton title={'帮助文档'} url={`toolbox/external-access/ingress-rule`} />}
        addButton={
          <Button onClick={handleAddIngress} disabled={isCreating}>
            <Plus className="size-4" />
            添加 Ingress 规则
          </Button>
        }
      />

      <Dialog open={isEditIngressDialogOpen} onOpenChange={setIsEditIngressDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加 Ingress 规则</DialogTitle>
          </DialogHeader>
          <Form {...ingressForm}>
            <form
              onSubmit={(e) => {
                //@ts-expect-error // ignore
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
    </>
  )
}
