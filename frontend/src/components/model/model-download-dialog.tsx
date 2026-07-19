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
import type { TFunction } from 'i18next'
import { BoxIcon, DatabaseIcon } from 'lucide-react'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
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

const createFormSchema = (t: TFunction) =>
  z.object({
    name: z
      .string()
      .min(1, { message: t('modelDownload.dialog.validation.nameRequired') })
      .regex(/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/, {
        message: t('modelDownload.dialog.validation.nameFormat'),
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
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [pendingRequest, setPendingRequest] = useState<CreateModelDownloadReq | null>(null)
  const formSchema = createFormSchema(t)

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
  const categoryLabel = t(
    defaultCategory === 'dataset'
      ? 'modelDownload.dialog.category.dataset'
      : 'modelDownload.dialog.category.model'
  )

  const sourcePlaceholder =
    source === 'modelscope'
      ? t('modelDownload.dialog.namePlaceholder.modelscope')
      : t('modelDownload.dialog.namePlaceholder.huggingface')
  const revisionPlaceholder =
    source === 'modelscope'
      ? t('modelDownload.dialog.revisionPlaceholder.modelscope')
      : t('modelDownload.dialog.revisionPlaceholder.huggingface')

  const tokenHint =
    source === 'modelscope'
      ? t('modelDownload.dialog.tokenHint.modelscope')
      : t('modelDownload.dialog.tokenHint.huggingface')

  const { mutate, status } = useMutation({
    mutationFn: (data: CreateModelDownloadReq) => apiCreateModelDownload(data),
    onSuccess: (response, variables) => {
      const defaultMessage =
        variables.category === 'dataset'
          ? t('modelDownload.dialog.success.dataset')
          : t('modelDownload.dialog.success.model')
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

  const pendingCategoryLabel = t(
    pendingRequest?.category === 'dataset'
      ? 'modelDownload.dialog.category.dataset'
      : 'modelDownload.dialog.category.model'
  )
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
    ? t('modelDownload.dialog.quota.disabled')
    : effectiveLimitConfig.exempt
      ? t('modelDownload.dialog.quota.exempt')
      : t('modelDownload.dialog.quota.limited', {
          maxConcurrent: effectiveLimitConfig.maxConcurrent,
          windowHours: effectiveLimitConfig.windowHours,
          maxSuccessfulDownloads: effectiveLimitConfig.maxSuccessfulDownloads,
        })

  return (
    <>
      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <SandwichLayout
            footer={
              <LoadableButton
                type="submit"
                isLoading={status === 'pending'}
                isLoadingText={t('modelDownload.dialog.submitting')}
              >
                {t('modelDownload.dialog.startDownload')}
              </LoadableButton>
            }
          >
            <FormField
              name="source"
              control={form.control}
              render={({ field }) => (
                <FormItem>
                  <FormLabel>{t('modelDownload.dialog.sourceLabel')}</FormLabel>
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
              <FormLabel>{t('modelDownload.dialog.categoryLabel')}</FormLabel>
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
                  <FormLabel>
                    {t('modelDownload.dialog.nameLabel', { category: categoryLabel })}
                  </FormLabel>
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
                  <FormLabel>{t('modelDownload.dialog.revisionLabel')}</FormLabel>
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
                  <FormLabel>{t('modelDownload.dialog.tokenLabel')}</FormLabel>
                  <FormControl>
                    <Input
                      type="password"
                      placeholder={t('modelDownload.dialog.tokenPlaceholder')}
                      autoComplete="new-password"
                      data-1p-ignore
                      data-lpignore="true"
                      data-form-type="other"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>
                    {t('modelDownload.dialog.tokenDescription', { sourceHint: tokenHint })}
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <div className="bg-muted/50 text-muted-foreground rounded-md p-3 text-xs">
              <p className="mb-1 font-semibold">{t('modelDownload.dialog.tips.title')}</p>
              <ul className="ml-4 list-disc space-y-1">
                <li>{t('modelDownload.dialog.tips.publicDirectories')}</li>
                <li>{t('modelDownload.dialog.tips.nameSubdirectory')}</li>
                <li>{t('modelDownload.dialog.tips.sharedResource')}</li>
                <li>{quotaHint}</li>
                <li>{t('modelDownload.dialog.tips.privateToken')}</li>
                <li>{t('modelDownload.dialog.tips.patience')}</li>
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
            <AlertDialogTitle>
              {t('modelDownload.dialog.confirmTitle', { category: pendingCategoryLabel })}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t('modelDownload.dialog.confirmDescription', { quota: quotaHint })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          {pendingRequest && (
            <dl className="bg-muted/50 grid grid-cols-[5rem_minmax(0,1fr)] gap-x-3 gap-y-2 rounded-md p-4 text-sm">
              <dt className="text-muted-foreground">{t('modelDownload.dialog.categoryLabel')}</dt>
              <dd>{pendingCategoryLabel}</dd>
              <dt className="text-muted-foreground">{t('modelDownload.dialog.sourceLabel')}</dt>
              <dd>{pendingSourceLabel}</dd>
              <dt className="text-muted-foreground">{t('modelDownload.dialog.details.name')}</dt>
              <dd className="font-mono break-all">{pendingRequest.name}</dd>
              <dt className="text-muted-foreground">
                {t('modelDownload.dialog.details.revision')}
              </dt>
              <dd className="font-mono break-all">
                {pendingRequest.revision || t('modelDownload.dialog.defaultRevision')}
              </dd>
              <dt className="text-muted-foreground">{t('modelDownload.dialog.details.path')}</dt>
              <dd className="font-mono break-all">{pendingPath}</dd>
              <dt className="text-muted-foreground">{t('modelDownload.dialog.details.token')}</dt>
              <dd>
                {pendingRequest.token
                  ? t('modelDownload.dialog.tokenProvided')
                  : t('modelDownload.dialog.tokenMissing')}
              </dd>
            </dl>
          )}
          <AlertDialogFooter>
            <AlertDialogCancel disabled={status === 'pending'}>
              {t('modelDownload.dialog.backToEdit')}
            </AlertDialogCancel>
            <AlertDialogAction
              disabled={status === 'pending' || pendingRequest === null}
              onClick={(event) => {
                event.preventDefault()
                if (pendingRequest) {
                  mutate(pendingRequest)
                }
              }}
            >
              {status === 'pending'
                ? t('modelDownload.dialog.submitting')
                : t('modelDownload.dialog.confirmDownload')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
