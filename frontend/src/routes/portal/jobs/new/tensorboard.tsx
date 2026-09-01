import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { type TFunction, t as i18nT } from 'i18next'
import { useAtomValue } from 'jotai'
import {
  AlertCircleIcon,
  CheckCircle2Icon,
  CircleHelpIcon,
  ClockIcon,
  FolderOpenIcon,
  HardDriveIcon,
  LayoutGridIcon,
  LoaderCircleIcon,
  PlayIcon,
} from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { z } from 'zod'

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader } from '@/components/ui/card'
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
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'

import SelectBox from '@/components/custom/select-box'
import FormExportButton from '@/components/form/form-export-button'
import FormImportButton from '@/components/form/form-import-button'
import FormLabelMust from '@/components/form/form-label-must'
import { TemplateInfo } from '@/components/form/template-info'
import { MetadataFormTensorboard } from '@/components/form/types'
import { PublishConfigForm } from '@/components/job/publish'
import CardTitle from '@/components/label/card-title'
import PageTitle from '@/components/layout/page-title'

import {
  type CreateTensorboardReq,
  MAX_TENSORBOARD_SOURCE_JOBS,
  apiTensorboardCreate,
  apiTensorboardSourceConfig,
} from '@/services/api/tensorboard'
import { apiJobSelfList } from '@/services/api/vcjob'

import { atomUserInfo } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

type TensorboardSearch = {
  fromTemplate?: number
  sourceJob?: string
}

const validateTensorboardSearch = (search: Record<string, unknown>): TensorboardSearch => ({
  fromTemplate: Number(search.fromTemplate) || undefined,
  sourceJob: typeof search.sourceJob === 'string' ? search.sourceJob : undefined,
})

export const Route = createFileRoute('/portal/jobs/new/tensorboard')({
  validateSearch: validateTensorboardSearch,
  component: RouteComponent,
  loader: () => {
    return {
      crumb: i18nT('tensorboard.create.crumb'),
    }
  },
})

const createFormSchema = (t: TFunction) =>
  z
    .object({
      sourceJobNames: z
        .array(z.string())
        .min(1, t('tensorboard.validation.sourceRequired'))
        .max(
          MAX_TENSORBOARD_SOURCE_JOBS,
          t('tensorboard.validation.maxSourceJobs', { count: MAX_TENSORBOARD_SOURCE_JOBS })
        ),
      logDir: z.string(),
      sourceLogDirs: z.record(z.string()).default({}),
      ttlHours: z
        .number()
        .min(1, t('tensorboard.validation.minTTL'))
        .max(168, t('tensorboard.validation.maxTTL')),
    })
    .superRefine((data, ctx) => {
      if (data.sourceJobNames.length === 1) {
        const logDir = data.logDir.trim()
        if (logDir === '') {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['logDir'],
            message: t('tensorboard.validation.logDirRequired'),
          })
        } else if (!logDir.startsWith('/')) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['logDir'],
            message: t('tensorboard.jobForm.absolutePath'),
          })
        }
      }
      if (data.sourceJobNames.length > 1) {
        const missingJobs = data.sourceJobNames.filter(
          (jobName) => (data.sourceLogDirs[jobName] ?? '').trim() === ''
        )
        if (missingJobs.length > 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['sourceLogDirs'],
            message: t('tensorboard.validation.missingLogDirs', { jobs: missingJobs.join(', ') }),
          })
        }
        const invalidJobs = data.sourceJobNames.filter((jobName) => {
          const logDir = (data.sourceLogDirs[jobName] ?? '').trim()
          return logDir !== '' && !logDir.startsWith('/')
        })
        if (invalidJobs.length > 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['sourceLogDirs'],
            message: t('tensorboard.validation.invalidLogDirs', { jobs: invalidJobs.join(', ') }),
          })
        }
      }
    })

type FormSchema = z.infer<ReturnType<typeof createFormSchema>>

const dataProcessor = (data: FormSchema) => {
  return {
    ...data,
    sourceJobNames: data.sourceJobNames ?? [],
    logDir: data.logDir ?? '',
    sourceLogDirs: data.sourceLogDirs ?? {},
    ttlHours: Number(data.ttlHours) || 24,
  }
}

const FormHelpTooltip = ({ content }: { content: string }) => (
  <TooltipProvider delayDuration={100}>
    <Tooltip>
      <TooltipTrigger asChild>
        <CircleHelpIcon className="text-muted-foreground size-4 cursor-help" />
      </TooltipTrigger>
      <TooltipContent className="max-w-sm">{content}</TooltipContent>
    </Tooltip>
  </TooltipProvider>
)

type SourceConfigState = {
  status: 'loading' | 'loaded' | 'error'
  declaredLogDir: string
}

function RouteComponent() {
  const { t } = useTranslation()
  const navigate = Route.useNavigate()
  const queryClient = useQueryClient()
  const searchParams = Route.useSearch()
  const currentUser = useAtomValue(atomUserInfo)

  const { data: jobList } = useQuery({
    queryKey: ['job', 'self', 'tensorboard-source'],
    queryFn: () =>
      apiJobSelfList({
        page: 1,
        page_size: 200,
        filters: {},
      }),
    select: (res) => res.data.items,
  })

  const formSchema = useMemo(() => createFormSchema(t), [t])

  const form = useForm<FormSchema>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      sourceJobNames: [],
      logDir: '',
      sourceLogDirs: {},
      ttlHours: 24,
    },
  })
  const selectedSourceJobNames = form.watch('sourceJobNames')
  const selectedSourceLogDirs = form.watch('sourceLogDirs')
  const isMultiSource = selectedSourceJobNames.length > 1
  const [sourceConfigStates, setSourceConfigStates] = useState<Record<string, SourceConfigState>>(
    {}
  )
  const [pendingCreateRequest, setPendingCreateRequest] = useState<CreateTensorboardReq | null>(
    null
  )
  const isResolvingSourceConfigs = selectedSourceJobNames.some(
    (jobName) => sourceConfigStates[jobName]?.status === 'loading'
  )
  const hasMissingMultiLogDirs =
    isMultiSource &&
    selectedSourceJobNames.some((jobName) => (selectedSourceLogDirs[jobName] ?? '').trim() === '')

  const { mutate: loadSourceConfig } = useMutation({
    mutationFn: apiTensorboardSourceConfig,
    onMutate: (jobName) => {
      setSourceConfigStates((current) => ({
        ...current,
        [jobName]: {
          status: 'loading',
          declaredLogDir: current[jobName]?.declaredLogDir ?? '',
        },
      }))
    },
    onSuccess: (res, jobName) => {
      const configuredLogDir = (res.data.logDir ?? '').trim()
      setSourceConfigStates((current) => ({
        ...current,
        [jobName]: { status: 'loaded', declaredLogDir: configuredLogDir },
      }))
      const sourceLogDirs = form.getValues('sourceLogDirs')
      if ((sourceLogDirs[jobName] ?? '').trim() === '') {
        form.setValue(
          'sourceLogDirs',
          { ...sourceLogDirs, [jobName]: configuredLogDir },
          {
            shouldValidate: true,
          }
        )
      }
      const selectedJobs = form.getValues('sourceJobNames')
      if (
        selectedJobs.length === 1 &&
        selectedJobs[0] === jobName &&
        form.getValues('logDir').trim() === ''
      ) {
        form.setValue('logDir', configuredLogDir, { shouldValidate: true })
      }
    },
    onError: (err, jobName) => {
      setSourceConfigStates((current) => ({
        ...current,
        [jobName]: { status: 'error', declaredLogDir: '' },
      }))
      showErrorToast(err)
    },
  })

  useEffect(() => {
    const sourceJob = searchParams.sourceJob
    if (sourceJob) {
      form.setValue('sourceJobNames', [sourceJob])
      form.setValue('logDir', '')
    }
  }, [form, searchParams.sourceJob])

  useEffect(() => {
    selectedSourceJobNames.forEach((jobName) => {
      if (sourceConfigStates[jobName] === undefined) {
        loadSourceConfig(jobName)
      }
    })
  }, [loadSourceConfig, selectedSourceJobNames, sourceConfigStates])

  const { mutate: createTensorboard, isPending } = useMutation({
    mutationFn: apiTensorboardCreate,
    onSuccess: () => {
      setPendingCreateRequest(null)
      toast.success(t('tensorboard.create.success'))
      queryClient.invalidateQueries({ queryKey: ['tensorboards'] })
      if (currentUser?.name) {
        navigate({
          to: '/portal/users/$name',
          params: { name: currentUser.name },
          search: { tab: 'tensorboard' },
        })
      } else {
        navigate({ to: '/portal/jobs/tensorboard' })
      }
    },
    onError: (err) => {
      showErrorToast(err)
    },
  })

  const onSubmit = (data: FormSchema) => {
    if (isPending) {
      toast.info(t('tensorboard.create.duplicateSubmit'))
      return
    }
    const singleSource = data.sourceJobNames.length === 1
    setPendingCreateRequest({
      sourceJobs: data.sourceJobNames.map((jobName) => ({
        jobName,
        logDir: singleSource ? data.logDir.trim() : (data.sourceLogDirs[jobName] ?? '').trim(),
      })),
      // Keep the top-level value for compatibility with exported legacy configurations.
      logDir: singleSource ? data.logDir.trim() : '',
      ttlHours: data.ttlHours,
    })
  }

  return (
    <>
      <Form {...form}>
        <form
          onSubmit={form.handleSubmit(onSubmit)}
          className="grid items-start gap-4 md:gap-6 lg:grid-cols-3"
        >
          <PageTitle
            title={t('tensorboard.create.title')}
            description={t('tensorboard.create.description')}
            className="lg:col-span-3"
            tipContent={t('tensorboard.create.version', {
              version: MetadataFormTensorboard.version,
            })}
          >
            <div className="flex w-full flex-wrap items-center justify-start gap-2 sm:w-fit sm:flex-nowrap sm:justify-end sm:gap-3">
              <FormImportButton
                metadata={MetadataFormTensorboard}
                dataProcessor={dataProcessor}
                form={form}
              />
              <FormExportButton metadata={MetadataFormTensorboard} form={form} />
              <PublishConfigForm
                config={MetadataFormTensorboard}
                configform={form}
                fromTemplate={searchParams.fromTemplate}
              />
              <Button
                type="submit"
                disabled={isPending || isResolvingSourceConfigs || hasMissingMultiLogDirs}
                className="gap-2"
              >
                <PlayIcon className="h-4 w-4" />
                {isPending ? t('tensorboard.create.creating') : t('tensorboard.create.submit')}
              </Button>
            </div>
          </PageTitle>

          <div className="flex flex-col gap-4 md:gap-6 lg:col-span-2">
            <Card className="rounded-md border shadow-sm">
              <CardHeader className="bg-muted/50 p-4 pb-2">
                <CardTitle icon={LayoutGridIcon}>{t('tensorboard.create.basicSettings')}</CardTitle>
              </CardHeader>
              <CardContent className="space-y-4 p-4">
                <FormField
                  control={form.control}
                  name="sourceJobNames"
                  render={({ field }) => (
                    <FormItem>
                      <div className="flex items-center gap-1">
                        <FormLabel>
                          <HardDriveIcon className="mr-2 inline-block h-4 w-4" />
                          {t('tensorboard.create.sourceJobs')}
                          <FormLabelMust />
                        </FormLabel>
                        <FormHelpTooltip
                          content={t('tensorboard.create.sourceJobsDescription', {
                            count: MAX_TENSORBOARD_SOURCE_JOBS,
                          })}
                        />
                      </div>
                      <FormDescription>
                        {t('tensorboard.create.sourceJobsDescription', {
                          count: MAX_TENSORBOARD_SOURCE_JOBS,
                        })}
                      </FormDescription>
                      <FormControl>
                        <SelectBox
                          withDialogOverlay={false}
                          options={(jobList ?? []).map((job) => ({
                            value: job.jobName,
                            label: job.jobName,
                            labelNote: job.status,
                          }))}
                          value={field.value}
                          onChange={(values) => {
                            if (values.length > MAX_TENSORBOARD_SOURCE_JOBS) {
                              toast.warning(
                                t('tensorboard.validation.maxSourceJobs', {
                                  count: MAX_TENSORBOARD_SOURCE_JOBS,
                                })
                              )
                              return
                            }
                            const sourceLogDirs = { ...form.getValues('sourceLogDirs') }
                            if (field.value.length === 1) {
                              sourceLogDirs[field.value[0]] = form.getValues('logDir')
                              form.setValue('sourceLogDirs', sourceLogDirs)
                            }
                            field.onChange(values)
                            if (values.length === 1) {
                              const jobName = values[0]
                              const knownLogDir =
                                sourceLogDirs[jobName] ??
                                sourceConfigStates[jobName]?.declaredLogDir ??
                                ''
                              form.setValue('logDir', knownLogDir, { shouldValidate: true })
                            } else {
                              form.setValue('logDir', '')
                            }
                          }}
                          placeholder={t('tensorboard.create.sourceJobsPlaceholder')}
                          inputPlaceholder={t('tensorboard.create.searchJobs')}
                          emptyPlaceholder={t('tensorboard.create.noJobs')}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                {isMultiSource ? (
                  <FormField
                    control={form.control}
                    name="sourceLogDirs"
                    render={({ field }) => (
                      <FormItem>
                        <div className="flex items-center gap-1">
                          <FormLabel>
                            <FolderOpenIcon className="mr-2 inline-block h-4 w-4" />
                            {t('tensorboard.create.sourceLogDirs')}
                            <FormLabelMust />
                          </FormLabel>
                          <FormHelpTooltip
                            content={t('tensorboard.create.sourceLogDirsDescription')}
                          />
                        </div>
                        <FormDescription>
                          {t('tensorboard.create.sourceLogDirsDescription')}
                        </FormDescription>
                        <div className="space-y-3">
                          {selectedSourceJobNames.map((jobName) => {
                            const value = field.value[jobName] ?? ''
                            const state = sourceConfigStates[jobName]
                            const hasValue = value.trim() !== ''
                            return (
                              <div key={jobName} className="space-y-2 rounded-md border p-3">
                                <div className="flex min-w-0 items-center justify-between gap-3">
                                  <span
                                    className="min-w-0 truncate font-mono text-sm"
                                    title={jobName}
                                  >
                                    {jobName}
                                  </span>
                                  {state?.status === 'loading' ? (
                                    <span className="text-muted-foreground flex shrink-0 items-center gap-1 text-xs">
                                      <LoaderCircleIcon className="h-3.5 w-3.5 animate-spin" />
                                      {t('tensorboard.create.loadingSourceConfig')}
                                    </span>
                                  ) : hasValue ? (
                                    <span className="flex shrink-0 items-center gap-1 text-xs text-emerald-600 dark:text-emerald-400">
                                      <CheckCircle2Icon className="h-3.5 w-3.5" />
                                      {t('tensorboard.create.logDirReady')}
                                    </span>
                                  ) : (
                                    <span className="flex shrink-0 items-center gap-1 text-xs text-amber-600 dark:text-amber-400">
                                      <AlertCircleIcon className="h-3.5 w-3.5" />
                                      {state?.status === 'error'
                                        ? t('tensorboard.create.sourceConfigFailed')
                                        : t('tensorboard.create.logDirMissing')}
                                    </span>
                                  )}
                                </div>
                                <Input
                                  value={value}
                                  placeholder={t('tensorboard.create.logDirExample')}
                                  onChange={(event) =>
                                    field.onChange({
                                      ...field.value,
                                      [jobName]: event.target.value,
                                    })
                                  }
                                />
                              </div>
                            )
                          })}
                        </div>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                ) : (
                  <FormField
                    control={form.control}
                    name="logDir"
                    render={({ field }) => (
                      <FormItem>
                        <div className="flex items-center gap-1">
                          <FormLabel>
                            <FolderOpenIcon className="mr-2 inline-block h-4 w-4" />
                            {t('tensorboard.create.logDir')} <FormLabelMust />
                          </FormLabel>
                          <FormHelpTooltip content={t('tensorboard.create.logDirDescription')} />
                        </div>
                        <FormDescription>
                          {t('tensorboard.create.logDirDescription')}
                        </FormDescription>
                        <FormControl>
                          <Input placeholder="/workspace/logs" {...field} />
                        </FormControl>
                        {selectedSourceJobNames.length === 1 &&
                          sourceConfigStates[selectedSourceJobNames[0]]?.status === 'error' && (
                            <p className="text-xs text-amber-600 dark:text-amber-400">
                              {t('tensorboard.create.sourceConfigFailedDescription')}
                            </p>
                          )}
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                )}

                <FormField
                  control={form.control}
                  name="ttlHours"
                  render={({ field }) => (
                    <FormItem>
                      <div className="flex items-center gap-1">
                        <FormLabel>
                          <ClockIcon className="mr-2 inline-block h-4 w-4" />
                          {t('tensorboard.create.ttl')} <FormLabelMust />
                        </FormLabel>
                        <FormHelpTooltip content={t('tensorboard.create.ttlDescription')} />
                      </div>
                      <FormDescription>{t('tensorboard.create.ttlDescription')}</FormDescription>
                      <FormControl>
                        <Input
                          type="number"
                          placeholder="24"
                          {...field}
                          onChange={(e) => field.onChange(parseInt(e.target.value, 10) || 0)}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </CardContent>
            </Card>

            <TemplateInfo
              form={form}
              metadata={MetadataFormTensorboard}
              searchParams={searchParams}
              dataProcessor={dataProcessor}
              defaultMarkdown={t('tensorboard.create.helpMarkdown')}
            />
          </div>

          <div className="flex flex-col gap-4 md:gap-6" />
        </form>
      </Form>

      <AlertDialog
        open={pendingCreateRequest !== null}
        onOpenChange={(open) => {
          if (!open && !isPending) {
            setPendingCreateRequest(null)
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('tensorboard.create.confirmTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('tensorboard.create.confirmDescription', {
                count: pendingCreateRequest?.sourceJobs?.length ?? 0,
                ttl: pendingCreateRequest?.ttlHours ?? 0,
              })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isPending}>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              disabled={isPending || pendingCreateRequest === null}
              onClick={(event) => {
                event.preventDefault()
                if (pendingCreateRequest !== null && !isPending) {
                  createTensorboard(pendingCreateRequest)
                }
              }}
            >
              {isPending ? t('tensorboard.create.creating') : t('tensorboard.create.confirmAction')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}
