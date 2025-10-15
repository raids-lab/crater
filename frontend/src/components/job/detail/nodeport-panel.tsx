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
import { NamespacedName } from '@/components/codeblock/pod-container-dialog'
import FormLabelMust from '@/components/form/form-label-must'

import {
  PodNodeport,
  apiCreatePodNodeport,
  apiDeletePodNodeport,
  apiGetPodNodeports,
} from '@/services/api/tool'
import { PodNodeportMgr } from '@/services/api/tool'

import { ExternalAccessItem, ExternalAccessList } from './external-access-list'

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

  const onSubmitNodeport = (data: PodNodeport) => {
    if (namespacedName) {
      createNodeportMutation(data)
    }
  }

  const handleDeleteNodeport = (data: PodNodeport) => {
    if (namespacedName) {
      deleteNodeportMutation(data)
    }
  }

  const renderNodeportItem = (nodePort: PodNodeport) => {
    return (
      <ExternalAccessItem
        key={nodePort.name}
        name={nodePort.name}
        port={nodePort.nodePort}
        url={`${nodePort.address}:${nodePort.nodePort}`}
        fullURL={`http://${nodePort.address}:${nodePort.nodePort}`}
        isDeleting={isDeleting}
        handleDeleteItem={() => handleDeleteNodeport(nodePort)}
      />
    )
  }

  return (
    <>
      <ExternalAccessList
        items={nodeportList}
        renderItem={renderNodeportItem}
        docsButton={<DocsButton title={'帮助文档'} url={`toolbox/external-access/nodeport-rule`} />}
        addButton={
          <Button onClick={handleAddNodeport} disabled={isCreating}>
            <Plus className="size-4" />
            添加 NodePort 规则
          </Button>
        }
      />
      <Dialog open={isEditNodeportDialogOpen} onOpenChange={setIsEditNodeportDialogOpen}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加 NodePort 规则</DialogTitle>
          </DialogHeader>
          <Form {...nodeportForm}>
            <form
              onSubmit={(e) => {
                //@ts-expect-error // ignore
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
