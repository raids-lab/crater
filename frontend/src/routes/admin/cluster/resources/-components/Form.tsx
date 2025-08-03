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
// i18n-processed-v1.1.0
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { PlusCircleIcon, XIcon } from 'lucide-react'
import * as React from 'react'
import { FC } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

import FormLabelMust from '@/components/form/FormLabelMust'

import {
  Resource,
  apiAdminResourceNetworkAdd,
  apiAdminResourceNetworkDelete,
  apiAdminResourceNetworksList,
  apiAdminResourceUpdate,
  apiResourceList,
} from '@/services/api/resource'

interface UpdateTaskFormProps extends React.HTMLAttributes<HTMLDivElement> {
  current: Resource
  open: boolean
  onOpenChange: (open: boolean) => void
}

const formSchema = z.object({
  id: z.number().int(),
  label: z.string().min(1, { message: 'updateResourceForm.labelError' }),
})

type FormSchema = z.infer<typeof formSchema>

export function UpdateResourceForm({ open, onOpenChange, current }: UpdateTaskFormProps) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      id: current.ID,
      label: current.label,
    },
  })

  const closeDialog = (event: { preventDefault: () => void }) => {
    event.preventDefault()
    onOpenChange(false)
  }

  const { mutate: updateLabelPack } = useMutation({
    mutationFn: (values: FormSchema) => apiAdminResourceUpdate(values.id, values.label),
    onSuccess: async (_, { label }) => {
      await queryClient.invalidateQueries({
        queryKey: ['resource', 'list'],
      })
      toast.success(`Label ${label} ${t('updateResourceForm.successMessage')}`)
      onOpenChange(false)
    },
  })

  // 2. Define a submit handler.
  const onUpdateSubmit = (values: FormSchema) => {
    updateLabelPack(values)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('updateResourceForm.title')}</DialogTitle>
          <DialogDescription>
            {t('updateResourceForm.description', { name: current.label })}
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onUpdateSubmit)} className="grid gap-4">
            <FormField
              control={form.control}
              name="label"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>
                    {t('updateResourceForm.label')}
                    <FormLabelMust />
                  </FormLabel>
                  <Input {...field} />
                  <FormDescription>{t('updateResourceForm.formDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter className="grid grid-cols-2">
              <Button onClick={closeDialog} variant={'secondary'}>
                {t('updateResourceForm.cancelButton')}
              </Button>
              <Button type="submit">{t('updateResourceForm.submitButton')}</Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

interface UpdateResourceTypeFormProps {
  current: Resource
  open: boolean
  onOpenChange: (open: boolean) => void
}

const resourceTypeFormSchema = z.object({
  type: z.enum(['default', 'gpu', 'rdma']),
})

export const UpdateResourceTypeForm: FC<UpdateResourceTypeFormProps> = ({
  current,
  open,
  onOpenChange,
}) => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const form = useForm<z.infer<typeof resourceTypeFormSchema>>({
    resolver: zodResolver(resourceTypeFormSchema),
    defaultValues: {
      type: current.type || 'default',
    },
  })

  const { mutate: updateType, isPending } = useMutation({
    mutationFn: (values: z.infer<typeof resourceTypeFormSchema>) => {
      return apiAdminResourceUpdate(current.ID, current.label, values.type)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['resource', 'list'] })
      await queryClient.invalidateQueries({
        queryKey: ['resource', 'networks', current.ID],
      })
      toast.success(t('updateResourceTypeForm.successMessage'))
      onOpenChange(false)
    },
  })

  function onSubmit(values: z.infer<typeof resourceTypeFormSchema>) {
    if (values.type === 'default') {
      return
    }
    updateType(values)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{t('updateResourceTypeForm.title')}</DialogTitle>
        </DialogHeader>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField
              control={form.control}
              name="type"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('updateResourceTypeForm.label')}</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue={field.value}>
                    <FormControl>
                      <SelectTrigger>
                        <SelectValue placeholder={t('updateResourceTypeForm.placeholder')} />
                      </SelectTrigger>
                    </FormControl>
                    <SelectContent>
                      <SelectItem value="default">
                        {t('updateResourceTypeForm.type.default')}
                      </SelectItem>
                      <SelectItem value="gpu">{t('updateResourceTypeForm.type.gpu')}</SelectItem>
                      <SelectItem value="rdma">{t('updateResourceTypeForm.type.rdma')}</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormDescription>{t('updateResourceTypeForm.formDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <div className="flex justify-end">
              <Button type="submit" disabled={isPending}>
                {t('updateResourceTypeForm.submitButton')}
              </Button>
            </div>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  )
}

interface NetworkAssociationFormProps {
  gpuResource: Resource
  open: boolean
  onOpenChange: (open: boolean) => void
}

export const NetworkAssociationForm: FC<NetworkAssociationFormProps> = ({
  gpuResource,
  open,
  onOpenChange,
}) => {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  // Fetch all RDMA resources
  const rdmaResourcesQuery = useQuery({
    queryKey: ['resource', 'list', 'rdma'],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data.filter((r) => r.type === 'rdma')
    },
  })

  // Fetch current network associations
  const currentNetworksQuery = useQuery({
    queryKey: ['resource', 'networks', gpuResource.ID],
    queryFn: () => apiAdminResourceNetworksList(gpuResource.ID),
    select: (res) => res.data,
  })

  const { mutate: addNetworkAssociation, isPending } = useMutation({
    mutationFn: (rdmaId: number) => {
      return apiAdminResourceNetworkAdd(gpuResource.ID, rdmaId)
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['resource', 'networks', gpuResource.ID],
      })
      toast.success(t('networkAssociationForm.addSuccess'))
    },
  })

  const { mutate: removeNetworkAssociation } = useMutation({
    mutationFn: (rdmaId: number) => apiAdminResourceNetworkDelete(gpuResource.ID, rdmaId),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['resource', 'networks', gpuResource.ID],
      })
      toast.success(t('networkAssociationForm.removeSuccess'))
    },
  })

  const currentNetworkIds = currentNetworksQuery.data?.map((n) => n.ID) || []

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>{t('networkAssociationForm.title')}</DialogTitle>
        </DialogHeader>
        <div className="space-y-4">
          <div>
            <h3 className="mb-2 text-sm font-medium">
              {t('networkAssociationForm.gpuResourceLabel')}
            </h3>
            <Badge className="font-mono" variant="secondary">
              {gpuResource.name}
            </Badge>
          </div>

          <div>
            <h3 className="mb-2 text-sm font-medium">
              {t('networkAssociationForm.currentNetworksLabel')}
            </h3>
            {currentNetworksQuery.isLoading ? (
              <div className="text-muted-foreground text-sm">
                {t('networkAssociationForm.loadingMessage')}
              </div>
            ) : currentNetworkIds.length === 0 ? (
              <div className="text-muted-foreground text-sm">
                {t('networkAssociationForm.noNetworksMessage')}
              </div>
            ) : (
              <div className="flex flex-wrap gap-2">
                {currentNetworksQuery.data?.map((network) => (
                  <Badge key={network.ID} variant="outline" className="font-mono">
                    {network.name}
                  </Badge>
                ))}
              </div>
            )}
          </div>

          <div>
            <h3 className="mb-2 text-sm font-medium">
              {t('networkAssociationForm.availableResourcesLabel')}
            </h3>
            {rdmaResourcesQuery.isLoading ? (
              <div className="text-muted-foreground text-sm">
                {t('networkAssociationForm.loadingMessage')}
              </div>
            ) : rdmaResourcesQuery.data?.length === 0 ? (
              <div className="text-muted-foreground text-sm">
                {t('networkAssociationForm.noResourcesMessage')}
              </div>
            ) : (
              <ScrollArea className="h-56 rounded-md border">
                <div className="space-y-2 p-4">
                  {rdmaResourcesQuery.data?.map((rdma) => {
                    const isAssociated = currentNetworkIds.includes(rdma.ID)

                    return (
                      <div
                        key={rdma.ID}
                        className="hover:bg-muted flex items-center justify-between rounded p-2"
                      >
                        <div className="flex items-center gap-2">
                          <Badge variant="outline" className="font-mono">
                            {rdma.name}
                          </Badge>
                          {rdma.label && (
                            <span className="text-muted-foreground text-xs">{rdma.label}</span>
                          )}
                        </div>

                        {isAssociated ? (
                          <Button
                            size="sm"
                            variant="destructive"
                            disabled={isPending}
                            onClick={() => removeNetworkAssociation(rdma.ID)}
                          >
                            <XIcon className="size-4" />
                            {t('networkAssociationForm.disconnectButton')}
                          </Button>
                        ) : (
                          <Button
                            size="sm"
                            variant="outline"
                            disabled={isPending}
                            onClick={() => addNetworkAssociation(rdma.ID)}
                          >
                            <PlusCircleIcon className="size-4" />
                            {t('networkAssociationForm.connectButton')}
                          </Button>
                        )}
                      </div>
                    )
                  })}
                </div>
              </ScrollArea>
            )}
          </div>

          <div className="flex justify-end pt-2">
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              {t('networkAssociationForm.doneButton')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
