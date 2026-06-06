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
import { CalendarIcon, Check, ChevronsUpDown, CirclePlusIcon, XIcon } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
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

import { AccountFormSchema, createAccountFormSchema } from './account-form'

interface AccountFormProps {
  onOpenChange: (open: boolean) => void
  account?: IAccount | null
}

const parseOptionalQuotaNumber = (value: unknown) => {
  if (value === '' || value === null || value === undefined) {
    return undefined
  }
  return Number(value)
}

interface ResourceNamePickerProps {
  options: { name: string; label: string }[]
  current: string
  onSelect: (value: string) => void
}

const ResourceNamePicker = ({ options, current, onSelect }: ResourceNamePickerProps) => {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const formTitle = t('accountForm.resourceName')
  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon"
          tabIndex={-1}
          className="text-muted-foreground hover:text-foreground absolute top-0 right-0 h-full px-2 hover:bg-transparent"
        >
          <ChevronsUpDown className="size-4 opacity-60" />
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-64 p-0" align="end" onOpenAutoFocus={(e) => e.preventDefault()}>
        <Command>
          <CommandInput placeholder={t('combobox.search', { formTitle })} className="h-9" />
          <CommandList>
            <CommandEmpty>{t('combobox.noResults', { formTitle })}</CommandEmpty>
            <CommandGroup>
              {options.map((opt) => (
                <CommandItem
                  key={opt.name}
                  value={`${opt.label} ${opt.name}`}
                  onSelect={() => {
                    onSelect(opt.name)
                    setOpen(false)
                  }}
                >
                  <div className="flex w-full min-w-0 flex-col">
                    <span className="truncate">{opt.label}</span>
                    {opt.label !== opt.name && (
                      <span className="text-muted-foreground truncate font-mono text-xs">
                        {opt.name}
                      </span>
                    )}
                  </div>
                  <Check
                    className={cn(
                      'ml-auto size-4 shrink-0',
                      opt.name === current ? 'opacity-100' : 'opacity-0'
                    )}
                  />
                </CommandItem>
              ))}
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  )
}

export const AccountForm = ({ onOpenChange, account }: AccountFormProps) => {
  const { scheduler } = useAtomValue(globalSettings)
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [cachedFormName, setCachedFormName] = useState<string | null>(null)
  const formSchema = useMemo(() => createAccountFormSchema(t), [t])

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
        .filter(
          (item) =>
            item.name != 'ephemeral-storage' &&
            item.name != 'hugepages-1Gi' &&
            item.name != 'hugepages-2Mi' &&
            item.name != 'pods'
        )
        .map((item) => ({ name: item.name, label: item.label || item.name }))
        .sort(
          // cpu > memory > xx/xx > pods
          (a, b) => {
            if (a.name === 'cpu') {
              return -1
            }
            if (b.name === 'cpu') {
              return 1
            }
            if (a.name === 'memory') {
              return -1
            }
            if (b.name === 'memory') {
              return 1
            }
            return a.name.localeCompare(b.name)
          }
        )
    },
  })
  const resourceDimension = resourceDimensionQuery.data
  const resourceNames = useMemo(() => resourceDimension?.map((r) => r.name), [resourceDimension])

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
      resources: resourceNames?.map((name) => ({ name })) || [{ name: 'cpu' }, { name: 'memory' }],
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
        account && resourceNames
          ? convertQuotaToForm(
              account.quota,
              resourceNames,
              isPublicAccount ? publicQueueQuota?.quota : undefined
            )
          : resourceNames?.map((name) => ({ name })) || [{ name: 'cpu' }, { name: 'memory' }],
      expiredAt: account?.expiredAt ? new Date(account.expiredAt) : undefined,
      admins: [],
      billingIssueAmount: accountBillingConfig?.issueAmount,
      billingIssuePeriodMinutes: accountBillingConfig?.issuePeriodMinutes,
      ...(account ? { id: account.id } : {}),
    }),
    [account, accountBillingConfig, isPublicAccount, publicQueueQuota, resourceNames]
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

  const onSubmit = (values: AccountFormSchema) => {
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
                    <Input
                      autoComplete="off"
                      placeholder={t('accountForm.namePlaceholder')}
                      maxLength={16}
                      {...field}
                      className="w-full"
                      autoFocus={true}
                    />
                  </FormControl>
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
                  <FormMessage />
                </FormItem>
              )}
            />
          )}

          {billingEnabled && (amountOverrideEnabled || periodOverrideEnabled) && (
            <div className="grid gap-4 md:grid-cols-2">
              {amountOverrideEnabled && (
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
                          value={field.value ?? ''}
                          onChange={(e) =>
                            field.onChange(
                              e.target.value === '' ? undefined : Number(e.target.value)
                            )
                          }
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
              {periodOverrideEnabled && (
                <FormField
                  control={form.control}
                  name="billingIssuePeriodMinutes"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>
                        {t('accountForm.billingIssuePeriodMinutesLabel', {
                          defaultValue: '发放周期（分钟，设为 0 关闭）',
                        })}
                      </FormLabel>
                      <FormControl>
                        <BillingPeriodFields
                          totalMinutes={field.value ?? 0}
                          onChange={(value) => {
                            field.onChange(value.totalMinutes)
                          }}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              )}
            </div>
          )}
          <div className="space-y-2">
            {resourcesFields.length > 0 && <FormLabel>{t('accountForm.quotaLabel')}</FormLabel>}
            {resourcesFields.map(({ id }, index) => (
              <div key={id} className="flex flex-row gap-2">
                <FormField
                  control={form.control}
                  name={`resources.${index}.name`}
                  render={({ field }) => {
                    const usedNames = new Set(
                      (currentValues.resources ?? [])
                        .map((r, i) => (i !== index ? r?.name : ''))
                        .filter((n): n is string => !!n)
                    )
                    const pickerOptions = (resourceDimension ?? []).filter(
                      (r) => !usedNames.has(r.name)
                    )
                    return (
                      <FormItem className="w-56">
                        <FormControl>
                          <div className="relative">
                            <Input
                              {...field}
                              value={field.value ?? ''}
                              placeholder={t('accountForm.resourcePlaceholder')}
                              className="pr-9 font-mono"
                            />
                            <ResourceNamePicker
                              options={pickerOptions}
                              current={field.value ?? ''}
                              onSelect={field.onChange}
                            />
                          </div>
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )
                  }}
                />
                <FormField
                  control={form.control}
                  name={`resources.${index}.guaranteed`}
                  render={() => (
                    <FormItem>
                      <FormControl>
                        <Input
                          type="number"
                          min={0}
                          step="any"
                          placeholder={t('accountForm.guaranteedPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.guaranteed`, {
                            setValueAs: parseOptionalQuotaNumber,
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
                          type="number"
                          min={0}
                          step="any"
                          placeholder={t('accountForm.deservedPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.deserved`, {
                            setValueAs: parseOptionalQuotaNumber,
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
                          type="number"
                          min={0}
                          step="any"
                          placeholder={t('accountForm.capabilityPlaceholder')}
                          className="font-mono"
                          {...form.register(`resources.${index}.capability`, {
                            setValueAs: parseOptionalQuotaNumber,
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
                            type="number"
                            min={0}
                            step="any"
                            placeholder={t('accountForm.queueLimitPlaceholder')}
                            className="font-mono"
                            {...form.register(`resources.${index}.queueLimit`, {
                              setValueAs: parseOptionalQuotaNumber,
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
              onClick={() => {
                const used = new Set(
                  (currentValues.resources ?? []).map((r) => r?.name).filter(Boolean)
                )
                const next = resourceDimension?.find((r) => !used.has(r.name))
                resourcesAppend({
                  name: next?.name ?? '',
                })
              }}
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
