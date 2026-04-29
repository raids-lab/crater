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
import { format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { useAtomValue } from 'jotai'
import { CalendarIcon, CirclePlusIcon, XIcon } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
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
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import LoadableButton from '@/components/button/loadable-button'
import SelectBox from '@/components/custom/select-box'
import { BillingPeriodFields } from '@/components/form/billing-period-fields'
import FormExportButton from '@/components/form/form-export-button'
import FormImportButton from '@/components/form/form-import-button'
import FormLabelMust from '@/components/form/form-label-must'
import { MetadataFormAccount } from '@/components/form/types'
import { SandwichLayout } from '@/components/sheet/sandwich-sheet'

import { IAccount, apiAccountCreate, apiAccountUpdate } from '@/services/api/account'
import { apiAdminUserList } from '@/services/api/admin/user'
import {
  apiAdminGetAccountBillingConfig,
  apiAdminUpdateAccountBillingConfig,
} from '@/services/api/billing'
import {
  apiAdminCreateQueueQuota,
  apiAdminDeleteQueueQuota,
  apiAdminGetQueueQuotas,
  apiAdminUpdateQueueQuota,
} from '@/services/api/queue-quota'
import { apiResourceList } from '@/services/api/resource'
import { apiAdminGetBillingStatus } from '@/services/api/system-config'

import { convertFormToQueueQuota, convertFormToQuota, convertQuotaToForm } from '@/utils/quota'
import { globalSettings } from '@/utils/store'

import { cn } from '@/lib/utils'

import { AccountFormSchema, formSchema } from './account-form'

interface AccountFormProps {
  onOpenChange: (open: boolean) => void
  account?: IAccount | null
}

export const AccountForm = ({ onOpenChange, account }: AccountFormProps) => {
  const { scheduler } = useAtomValue(globalSettings)
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [cachedFormName, setCachedFormName] = useState<string | null>(null)

  const { data: userList } = useQuery({
    queryKey: ['admin', 'userlist'],
    queryFn: apiAdminUserList,
    select: (res) =>
      res.data.map((user) => ({
        value: user.id.toString(),
        label: user.attributes.nickname || user.name,
        labelNote: user.name,
      })),
  })

  const resourceDimensionQuery = useQuery({
    queryKey: ['resources', 'list'],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data
        .map((item) => item.name)
        .filter(
          (name) =>
            name != 'ephemeral-storage' &&
            name != 'hugepages-1Gi' &&
            name != 'hugepages-2Mi' &&
            name != 'pods'
        )
        .sort(
          // cpu > memory > xx/xx > pods
          (a, b) => {
            if (a === 'cpu') {
              return -1
            }
            if (b === 'cpu') {
              return 1
            }
            if (a === 'memory') {
              return -1
            }
            if (b === 'memory') {
              return 1
            }
            return a.localeCompare(b)
          }
        )
    },
  })
  const resourceDimension = resourceDimensionQuery.data

  const { data: billingStatus } = useQuery({
    queryKey: ['admin', 'system-config', 'billing-status'],
    queryFn: () => apiAdminGetBillingStatus().then((res) => res.data),
  })
  const billingEnabled = billingStatus?.featureEnabled ?? false
  const amountOverrideEnabled = billingStatus?.accountIssueAmountOverrideEnabled ?? false
  const periodOverrideEnabled = billingStatus?.accountIssuePeriodOverrideEnabled ?? false
  const accountBillingConfigQuery = useQuery({
    queryKey: ['admin', 'accounts', account?.id, 'billing-config'],
    queryFn: () => apiAdminGetAccountBillingConfig(account?.id ?? 0).then((res) => res.data),
    enabled: billingEnabled && Boolean(account?.id),
  })
  const accountBillingConfig = accountBillingConfigQuery.data
  const hydratedFormKeyRef = useRef<string | null>(null)
  const isPublicAccount = account?.name === 'default'
  const publicQueueQuotaQuery = useQuery({
    queryKey: ['admin', 'queue-quotas'],
    queryFn: () => apiAdminGetQueueQuotas().then((res) => res.data),
    select: (data) => (data.quotas || []).find((quota) => quota.name === 'default'),
    enabled: Boolean(isPublicAccount && account?.id),
  })
  const publicQueueQuota = publicQueueQuotaQuery.data

  const form = useForm<AccountFormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: account?.nickname || '',
      resources: resourceDimension?.map((name) => ({ name })) || [
        { name: 'cpu' },
        { name: 'memory' },
      ],
      expiredAt: account?.expiredAt ? new Date(account.expiredAt) : undefined,
      admins: [],
      billingIssueAmount: accountBillingConfig?.issueAmount,
      billingIssuePeriodMinutes: accountBillingConfig?.issuePeriodMinutes,
      ...(account ? { id: account.id } : {}),
    },
  })

  const hydratedValues = useMemo<AccountFormSchema>(
    () => ({
      name: account?.nickname || '',
      resources:
        account && resourceDimension
          ? convertQuotaToForm(
              account.quota,
              resourceDimension,
              isPublicAccount ? publicQueueQuota?.quota : undefined
            )
          : resourceDimension?.map((name) => ({ name })) || [{ name: 'cpu' }, { name: 'memory' }],
      expiredAt: account?.expiredAt ? new Date(account.expiredAt) : undefined,
      admins: [],
      billingIssueAmount: accountBillingConfig?.issueAmount,
      billingIssuePeriodMinutes: accountBillingConfig?.issuePeriodMinutes,
      ...(account ? { id: account.id } : {}),
    }),
    [account, accountBillingConfig, isPublicAccount, publicQueueQuota, resourceDimension]
  )
  const needsBillingHydration = billingEnabled && Boolean(account?.id)
  const needsQueueQuotaHydration = isPublicAccount && Boolean(account?.id)
  const hydrationReady =
    resourceDimensionQuery.isFetched &&
    (!needsBillingHydration || accountBillingConfigQuery.isFetched) &&
    (!needsQueueQuotaHydration || publicQueueQuotaQuery.isFetched)
  const hydrationKey = hydrationReady
    ? `${account?.id ?? 'create'}:${resourceDimensionQuery.dataUpdatedAt}:${accountBillingConfigQuery.dataUpdatedAt}:${publicQueueQuotaQuery.dataUpdatedAt}`
    : null

  useEffect(() => {
    if (!hydrationKey) {
      return
    }
    if (hydratedFormKeyRef.current === hydrationKey) {
      return
    }

    form.reset(hydratedValues, {
      keepDirtyValues: true,
    })
    hydratedFormKeyRef.current = hydrationKey
  }, [form, hydratedValues, hydrationKey])

  const currentValues = form.watch()

  const {
    fields: resourcesFields,
    append: resourcesAppend,
    remove: resourcesRemove,
  } = useFieldArray<AccountFormSchema>({
    name: 'resources',
    control: form.control,
  })

  const buildBillingConfigPayload = (values: AccountFormSchema) => {
    const payload: { issueAmount?: number; issuePeriodMinutes?: number } = {}
    if (billingEnabled && amountOverrideEnabled && values.billingIssueAmount !== undefined) {
      payload.issueAmount = values.billingIssueAmount
    }
    if (billingEnabled && periodOverrideEnabled && values.billingIssuePeriodMinutes !== undefined) {
      payload.issuePeriodMinutes = values.billingIssuePeriodMinutes
    }
    return payload
  }

  const saveAccountBillingConfig = async (
    accountId: number,
    values: AccountFormSchema
  ): Promise<string | null> => {
    const billingPayload = buildBillingConfigPayload(values)
    if (Object.keys(billingPayload).length === 0) {
      return null
    }

    try {
      await apiAdminUpdateAccountBillingConfig(accountId, billingPayload)
      return null
    } catch (error) {
      if (error instanceof Error && error.message) {
        return error.message
      }
      return 'unknown error'
    }
  }

  const savePublicQueueQuota = async (values: AccountFormSchema): Promise<string | null> => {
    if (!isPublicAccount) {
      return null
    }

    const quota = convertFormToQueueQuota(values.resources)
    const hasQuota = Object.keys(quota).length > 0

    try {
      const existingQuota = publicQueueQuotaQuery.isFetched
        ? publicQueueQuota
        : (await apiAdminGetQueueQuotas()).data.quotas?.find((quota) => quota.name === 'default')

      if (existingQuota?.id) {
        if (!hasQuota) {
          await apiAdminDeleteQueueQuota(existingQuota.id)
          return null
        }
        await apiAdminUpdateQueueQuota(existingQuota.id, {
          name: 'default',
          quota,
        })
        return null
      }
      if (hasQuota) {
        await apiAdminCreateQueueQuota({
          name: 'default',
          quota,
        })
      }
      return null
    } catch (error) {
      if (error instanceof Error && error.message) {
        return error.message
      }
      return 'unknown error'
    }
  }

  const { mutate: createAccount, isPending: isCreatePending } = useMutation({
    mutationFn: async (values: AccountFormSchema) => {
      const resp = await apiAccountCreate({
        name: values.name,
        quota: convertFormToQuota(values.resources),
        expiredAt: values.expiredAt,
        admins: values.admins?.map((id) => parseInt(id)),
        withoutVolcano: scheduler !== 'volcano',
      })
      const billingConfigError =
        resp.data.id && resp.data.id > 0
          ? await saveAccountBillingConfig(resp.data.id, values)
          : null
      return {
        billingConfigError,
        resp,
      }
    },
    onSuccess: async ({ billingConfigError }, { name }) => {
      await queryClient.invalidateQueries({
        queryKey: ['admin', 'accounts'],
      })
      if (billingConfigError) {
        toast.warning(t('toast.accountCreatedPartialFailure', { name, error: billingConfigError }))
      } else {
        toast.success(t('toast.accountCreated', { name }))
      }
      onOpenChange(false)
    },
  })

  const { mutate: updateAccount, isPending: isUpdatePending } = useMutation({
    mutationFn: async (values: AccountFormSchema) => {
      const resp = await apiAccountUpdate(values.id ?? 0, {
        name: values.name,
        quota: convertFormToQuota(values.resources),
        expiredAt: values.expiredAt,
        admins: values.admins?.map((id) => parseInt(id)),
        withoutVolcano: scheduler !== 'volcano',
      })
      const queueQuotaError = await savePublicQueueQuota(values)
      const billingConfigError =
        (values.id ?? 0) > 0 ? await saveAccountBillingConfig(values.id ?? 0, values) : null
      return {
        billingConfigError,
        queueQuotaError,
        resp,
      }
    },
    onSuccess: async ({ billingConfigError, queueQuotaError }, { name }) => {
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: ['admin', 'accounts'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['admin', 'queue-quotas'],
        }),
        queryClient.invalidateQueries({
          queryKey: ['context', 'job-resource-summary'],
        }),
      ])
      if (account?.id) {
        await queryClient.invalidateQueries({
          queryKey: ['admin', 'accounts', account?.id, 'billing-config'],
        })
      }
      if (queueQuotaError || billingConfigError) {
        toast.warning(
          t('toast.accountUpdatedPartialFailure', {
            name,
            error: queueQuotaError || billingConfigError,
          })
        )
      } else {
        toast.success(t('toast.accountUpdated', { name }))
      }
      onOpenChange(false)
    },
  })

  const onSubmit = (values: z.infer<typeof formSchema>) => {
    if (values.id) {
      updateAccount(values)
    } else {
      createAccount(values)
    }
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)} className="grid gap-4">
        <SandwichLayout
          footer={
            <>
              <FormImportButton
                form={form}
                metadata={MetadataFormAccount}
                beforeImport={(data) => {
                  setCachedFormName(data.name)
                }}
                afterImport={() => {
                  if (cachedFormName) {
                    form.setValue('name', cachedFormName)
                    form.setValue('expiredAt', undefined)
                  }
                  setCachedFormName(null)
                }}
              />
              <FormExportButton form={form} metadata={MetadataFormAccount} />
              <LoadableButton
                isLoading={isCreatePending || isUpdatePending}
                isLoadingText={
                  form.getValues('id')
                    ? t('accountForm.updateButton')
                    : t('accountForm.createButton')
                }
                type="submit"
              >
                <CirclePlusIcon className="size-4" />
                {form.getValues('id')
                  ? t('accountForm.updateButton')
                  : t('accountForm.createButton')}
              </LoadableButton>
            </>
          }
        >
          <div className="flex flex-row items-start justify-between gap-4">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem className="col-span-1 grow">
                  <FormLabel>
                    {t('accountForm.nameLabel')}
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <Input autoComplete="off" {...field} className="w-full" autoFocus={true} />
                  </FormControl>
                  <FormDescription>{t('accountForm.nameDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="expiredAt"
              render={({ field }) => (
                <FormItem className="col-span-1 flex flex-col">
                  <FormLabel>{t('accountForm.expiredAtLabel')}</FormLabel>
                  <Popover>
                    <PopoverTrigger asChild>
                      <FormControl>
                        <Button
                          variant={'outline'}
                          className={cn(
                            'w-[240px] pl-3 text-left font-normal',
                            !field.value && 'text-muted-foreground'
                          )}
                        >
                          {field.value ? (
                            format(field.value, 'PPP', {
                              locale: zhCN,
                            })
                          ) : (
                            <span>{t('accountForm.expiredAtPlaceholder')}</span>
                          )}
                          <CalendarIcon className="ml-auto size-4 opacity-50" />
                        </Button>
                      </FormControl>
                    </PopoverTrigger>
                    <PopoverContent className="w-auto p-0" align="start">
                      <Calendar
                        mode="single"
                        locale={zhCN}
                        selected={field.value}
                        onSelect={field.onChange}
                        disabled={(date) => date < new Date()}
                        initialFocus
                      />
                    </PopoverContent>
                  </Popover>
                  <FormDescription>{t('accountForm.expiredAtDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          {!form.getValues('id') && (
            <FormField
              control={form.control}
              name="admins"
              render={() => (
                <FormItem>
                  <FormLabel>
                    {t('accountForm.adminsLabel')}
                    <FormLabelMust />
                  </FormLabel>
                  <FormControl>
                    <SelectBox
                      className="my-0 h-8"
                      options={userList ?? []}
                      inputPlaceholder={t('accountForm.adminsSearchPlaceholder')}
                      placeholder={t('accountForm.adminsPlaceholder')}
                      value={currentValues.admins}
                      onChange={(value) => form.setValue('admins', value)}
                    />
                  </FormControl>
                  <FormDescription>{t('accountForm.adminsDescription')}</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {billingEnabled && (
            <div className="grid gap-4 md:grid-cols-2">
              <FormField
                control={form.control}
                name="billingIssueAmount"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('accountForm.billingIssueAmountLabel', {
                        defaultValue: '周期发放额度',
                      })}
                    </FormLabel>
                    <FormControl>
                      <Input
                        type="number"
                        min={0}
                        disabled={!amountOverrideEnabled}
                        value={field.value ?? ''}
                        onChange={(e) =>
                          field.onChange(e.target.value === '' ? undefined : Number(e.target.value))
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      {amountOverrideEnabled
                        ? t('accountForm.billingIssueAmountDescription', {
                            defaultValue: '每个发放周期发放的免费点数额度。',
                          })
                        : t('accountForm.billingIssueAmountDisabledDescription', {
                            defaultValue: '当前由系统默认发放额度控制。',
                          })}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="billingIssuePeriodMinutes"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>
                      {t('accountForm.billingIssuePeriodMinutesLabel', {
                        defaultValue: '发放周期（分钟）',
                      })}
                    </FormLabel>
                    <FormControl>
                      <BillingPeriodFields
                        totalMinutes={field.value ?? 0}
                        disabled={!periodOverrideEnabled}
                        onChange={(value) => {
                          field.onChange(value.totalMinutes)
                        }}
                      />
                    </FormControl>
                    <FormDescription>
                      {periodOverrideEnabled
                        ? t('accountForm.billingIssuePeriodMinutesDescription', {
                            defaultValue: '填 0 表示关闭该账户的周期发放。',
                          })
                        : t('accountForm.billingIssuePeriodMinutesDisabledDescription', {
                            defaultValue: '当前由系统默认发放周期控制。',
                          })}
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          )}
          <div className="space-y-2">
            {resourcesFields.length > 0 && <FormLabel>{t('accountForm.quotaLabel')}</FormLabel>}
            {resourcesFields.map(({ id }, index) => (
              <div key={id} className="flex flex-row gap-2">
                <FormField
                  control={form.control}
                  name={`resources.${index}.name`}
                  render={({ field }) => (
                    <FormItem className="w-fit">
                      <FormControl>
                        <Input
                          {...field}
                          placeholder={t('accountForm.resourcePlaceholder')}
                          className="font-mono"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`resources.${index}.guaranteed`}
                  render={() => (
                    <FormItem>
                      <FormControl>
                        <Input
                          type="number"
                          placeholder={t('accountForm.guaranteedPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.guaranteed`, {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`resources.${index}.deserved`}
                  render={() => (
                    <FormItem>
                      <FormControl>
                        <Input
                          type="string"
                          placeholder={t('accountForm.deservedPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.deserved`, {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name={`resources.${index}.capability`}
                  render={() => (
                    <FormItem>
                      <FormControl>
                        <Input
                          type="string"
                          placeholder={t('accountForm.capabilityPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.capability`, {
                            valueAsNumber: true,
                          })}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                {isPublicAccount && (
                  <FormField
                    control={form.control}
                    name={`resources.${index}.queueLimit`}
                    render={() => (
                      <FormItem>
                        <FormControl>
                          <Input
                            type="string"
                            placeholder={t('accountForm.queueLimitPlaceholder')}
                            className="font-mono"
                            {...form.register(`resources.${index}.queueLimit`, {
                              valueAsNumber: true,
                            })}
                          />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}
                <div>
                  <Button
                    size="icon"
                    type="button"
                    variant="outline"
                    onClick={() => resourcesRemove(index)}
                  >
                    <XIcon className="size-4" />
                  </Button>
                </div>
              </div>
            ))}
            {resourcesFields.length > 0 && (
              <FormDescription>{t('accountForm.quotaDescription')}</FormDescription>
            )}

            <Button
              type="button"
              variant="secondary"
              onClick={() =>
                resourcesAppend({
                  name: '',
                })
              }
            >
              <CirclePlusIcon className="size-4" />
              {t('accountForm.addQuotaButton')}
            </Button>
          </div>
        </SandwichLayout>
      </form>
    </Form>
  )
}
