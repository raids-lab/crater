import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  CalendarIcon,
  CheckCircle2Icon,
  CopyIcon,
  CpuIcon,
  CreditCardIcon,
  FileSlidersIcon,
  GaugeIcon,
  MessageSquareIcon,
  ServerIcon,
  Trash2Icon,
  UserRoundIcon,
} from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

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
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import KthenaStatusBadge from '@/components/badge/kthena-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import CardTitle from '@/components/label/card-title'
import UserLabel from '@/components/label/user-label'
import { DetailInfo } from '@/components/layout/detail-page'
import PageTitle from '@/components/layout/page-title'
import NotFound from '@/components/placeholder/not-found'

import {
  apiDeleteKthenaService,
  apiGetKthenaDiagnostics,
  apiGetKthenaService,
  apiRepairKthenaService,
} from '@/services/api/inference'
import { apiQueueList } from '@/services/api/queue'

import { getServingResources } from '@/hooks/inference/format'
import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { atomUserContext } from '@/utils/store'
import { showErrorToast } from '@/utils/toast'

import { REFETCH_INTERVAL } from '@/lib/constants'

import { DiagnosticCard, DiagnosticSummary, ResourceCard } from './-components/detail-parts'
import { InferenceMetrics } from './-components/inference-metrics'
import { InvokeWorkspace } from './-components/invoke-workspace'
import ResourceUsage from './-components/resource-usage'
import { ServingRoleSummary } from './-components/serving-role-summary'
import { ServingRuntime } from './-components/serving-runtime'

export const Route = createFileRoute('/portal/inference-services/$name')({
  component: KthenaServiceDetailPage,
  errorComponent: () => <NotFound />,
  loader: ({ params }) => ({ crumb: params.name }),
})

function KthenaServiceDetailPage() {
  const { t } = useTranslation()
  const navigate = Route.useNavigate()
  const { name } = Route.useParams()
  const queryClient = useQueryClient()
  const keys = useInferenceKeys()
  const account = useAtomValue(atomUserContext)
  const { data: accountName } = useQuery({
    queryKey: ['queues'],
    queryFn: apiQueueList,
    select: ({ data }) => data.find((queue) => queue.name === account?.queue)?.nickname,
  })
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: keys.detail(name),
    queryFn: () => apiGetKthenaService(name).then((res) => res.data),
    refetchInterval: REFETCH_INTERVAL,
  })

  const [diagnosticsFor, setDiagnosticsFor] = useState<string>()
  const [repairOpen, setRepairOpen] = useState(false)
  const diagnosticsQuery = useQuery({
    queryKey: [...keys.detail(name), 'diagnostics'],
    queryFn: () => apiGetKthenaDiagnostics(name).then((res) => res.data),
    enabled: !!data && diagnosticsFor === name,
  })
  const repair = useMutation({
    mutationFn: () => apiRepairKthenaService(name),
    onSuccess: () => {
      setRepairOpen(false)
      void refetch()
    },
    onError: showErrorToast,
  })
  const service = data
    ? { ...data, diagnostics: [...(data.diagnostics ?? []), ...(diagnosticsQuery.data ?? [])] }
    : undefined
  const { mutate: deleteService, isPending: isDeleting } = useMutation({
    mutationFn: apiDeleteKthenaService,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      toast.success(t('kthena.delete.success'))
      navigate({ to: '/portal/inference-services' })
    },
    onError: showErrorToast,
  })

  if (error)
    return (
      <div role="alert" className="space-y-4">
        <p>{error.message}</p>
        <Button onClick={() => void refetch()}>{t('kthena.actions.retry')}</Button>
      </div>
    )
  if (isLoading || !service) {
    return (
      <div className="flex flex-col gap-4">
        <PageTitle title={t('kthena.detail.title')} description={t('kthena.list.syncing')}>
          <Button variant="outline" asChild>
            <Link to="/portal/inference-services">
              <ArrowLeftIcon className="size-4" />
              {t('kthena.actions.back')}
            </Link>
          </Button>
        </PageTitle>
        <Card>
          <CardContent className="text-muted-foreground flex h-40 items-center justify-center">
            {t('kthena.loading')}
          </CardContent>
        </Card>
      </div>
    )
  }

  const workerResources = getServingResources(service)

  return (
    <div className="flex flex-col gap-6">
      <PageTitle title={service.name} description={t('kthena.detail.description')}>
        <Button variant="outline" asChild>
          <Link
            to="/portal/inference-services/new"
            search={{ clone: service.name, edit: undefined }}
          >
            <CopyIcon className="size-4" />
            {t('kthena.actions.clone')}
          </Link>
        </Button>
        <Button variant="outline" asChild>
          <Link to="/portal/inference-services">
            <ArrowLeftIcon className="size-4" />
            {t('kthena.actions.backToList')}
          </Link>
        </Button>
        <Button variant="destructive" disabled={isDeleting} onClick={() => setDeleteOpen(true)}>
          <Trash2Icon className="size-4" />
          {t('kthena.actions.delete')}
        </Button>
      </PageTitle>

      <DetailInfo
        items={[
          {
            icon: CreditCardIcon,
            title: t('jobs.headers.queue'),
            value: accountName || account?.queue || '—',
          },
          {
            icon: UserRoundIcon,
            title: t('kthena.table.owner'),
            value: service.userInfo?.username ? (
              <UserLabel info={service.userInfo} />
            ) : (
              service.owner || '—'
            ),
          },
          {
            icon: ActivityIcon,
            title: t('kthena.table.status'),
            value: <KthenaStatusBadge service={service} />,
          },
          {
            icon: CalendarIcon,
            title: t('kthena.table.createdAt'),
            value: <TimeDistance date={service.createdAt} />,
          },
          {
            icon: ServerIcon,
            title: t('kthena.topology.engine'),
            value: (
              <span className="text-foreground font-medium">{service.backendType || '—'}</span>
            ),
          },
          {
            icon: CheckCircle2Icon,
            title: t('kthena.detail.servedModel'),
            value: (
              <span className="text-foreground font-mono" title={service.servedModel}>
                {service.servedModel || '—'}
              </span>
            ),
          },
          {
            icon: CpuIcon,
            title: t('kthena.topology.summary'),
            value: Object.keys(workerResources).length ? (
              <ResourceBadges resources={workerResources} />
            ) : (
              '—'
            ),
          },
          { icon: ServerIcon, title: t('kthena.table.queue'), value: service.queue || '—' },
        ]}
      />

      <Tabs
        key={name}
        defaultValue={service.phase === 'Ready' ? 'invoke' : 'runtime'}
        className="min-w-0 gap-0"
      >
        <div className="relative mb-4 max-w-full overflow-x-auto">
          <TabsList className="tabs-list-underline mb-0 min-w-max gap-4 sm:gap-6">
            <TabsTrigger className="tabs-trigger-underline" value="invoke">
              <MessageSquareIcon className="size-4" />
              {t('kthena.detail.tabs.invoke')}
            </TabsTrigger>
            <TabsTrigger className="tabs-trigger-underline" value="runtime">
              <ServerIcon className="size-4" />
              {t('kthena.runtime.title')}
            </TabsTrigger>
            <TabsTrigger className="tabs-trigger-underline" value="configuration">
              <FileSlidersIcon className="size-4" />
              {t('kthena.detail.tabs.configuration')}
            </TabsTrigger>
            <TabsTrigger className="tabs-trigger-underline" value="usage">
              <GaugeIcon className="size-4" />
              {t('kthena.detail.tabs.usage')}
            </TabsTrigger>
          </TabsList>
        </div>

        <TabsContent value="configuration" className="mt-0 space-y-4">
          <Card>
            <CardHeader className="flex flex-wrap items-center justify-between gap-3">
              <CardTitle icon={ServerIcon}>{t('kthena.topology.step.roles')}</CardTitle>
              <Dialog>
                <DialogTrigger asChild>
                  <Button variant="outline" size="sm">
                    {t('kthena.detail.viewOrchestration')}
                  </Button>
                </DialogTrigger>
                <DialogContent className="max-h-[85vh] overflow-auto sm:max-w-3xl">
                  <DialogHeader>
                    <DialogTitle>{t('kthena.detail.tabs.resources')}</DialogTitle>
                    <DialogDescription>{t('kthena.detail.orchestrationHelp')}</DialogDescription>
                  </DialogHeader>
                  <div className="grid gap-3 sm:grid-cols-2">
                    {service.resources?.length ? (
                      service.resources.map((resource) => (
                        <ResourceCard
                          key={`${resource.kind}/${resource.name}`}
                          resource={resource}
                        />
                      ))
                    ) : (
                      <p className="text-muted-foreground text-sm">
                        {t('kthena.detail.noResources')}
                      </p>
                    )}
                  </div>
                </DialogContent>
              </Dialog>
            </CardHeader>
            <CardContent>
              <ServingRoleSummary roles={service.roles} />
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="usage" className="mt-0 space-y-4">
          <InferenceMetrics name={service.name} />
          <ResourceUsage runtimePods={service.runtimePods} />
        </TabsContent>

        <TabsContent value="runtime" className="mt-0 space-y-4">
          <ServingRuntime service={service} />
          <Card>
            <CardHeader className="flex flex-wrap items-center justify-between gap-3">
              <CardTitle icon={AlertTriangleIcon}>{t('kthena.detail.tabs.diagnostics')}</CardTitle>
              <div className="flex flex-wrap items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={diagnosticsQuery.isFetching}
                  onClick={() => {
                    setDiagnosticsFor(name)
                    if (diagnosticsFor === name) void diagnosticsQuery.refetch()
                  }}
                >
                  {t('kthena.detail.checkDiagnostics')}
                </Button>
                {data?.diagnostics?.some((item) => item.reason === 'RoutingResourcesMissing') && (
                  <Button size="sm" onClick={() => setRepairOpen(true)}>
                    {t('kthena.actions.repair')}
                  </Button>
                )}
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {diagnosticsQuery.error && <p role="alert">{diagnosticsQuery.error.message}</p>}
              <AlertDialog open={repairOpen} onOpenChange={setRepairOpen}>
                <AlertDialogContent>
                  <AlertDialogHeader>
                    <AlertDialogTitle>{t('kthena.actions.repair')}</AlertDialogTitle>
                    <AlertDialogDescription>
                      {t('kthena.detail.repairDescription', { name })}
                    </AlertDialogDescription>
                  </AlertDialogHeader>
                  <AlertDialogFooter>
                    <AlertDialogCancel>{t('kthena.actions.cancel')}</AlertDialogCancel>
                    <AlertDialogAction
                      disabled={repair.isPending}
                      onClick={(event) => {
                        event.preventDefault()
                        repair.mutate()
                      }}
                    >
                      {t('kthena.actions.confirm')}
                    </AlertDialogAction>
                  </AlertDialogFooter>
                </AlertDialogContent>
              </AlertDialog>

              <div className="grid gap-3">
                {service.diagnostics?.length ? (
                  <>
                    <DiagnosticSummary diagnostics={service.diagnostics} />
                    <div className="before:bg-border relative grid gap-3 pl-5 before:absolute before:top-1 before:bottom-1 before:left-2 before:w-px">
                      {service.diagnostics.map((diagnostic, index) => (
                        <DiagnosticCard
                          key={`${diagnostic.resource}-${diagnostic.reason}-${index}`}
                          diagnostic={diagnostic}
                        />
                      ))}
                    </div>
                  </>
                ) : (
                  <div className="text-muted-foreground text-sm">
                    {t(
                      diagnosticsQuery.isFetching
                        ? 'common.loading'
                        : diagnosticsQuery.isSuccess
                          ? 'kthena.detail.noDiagnostics'
                          : 'kthena.detail.diagnosticsPrompt'
                    )}
                  </div>
                )}
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="invoke" className="mt-0">
          <InvokeWorkspace
            key={`${service.namespace}/${service.name}/${keys.scope.join('/')}`}
            service={service}
          />
        </TabsContent>
      </Tabs>
      <AlertDialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.delete.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.delete.description', { name: service.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isDeleting}>
              {t('kthena.actions.cancel')}
            </AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={isDeleting}
              onClick={() => deleteService(service.name)}
            >
              {t('kthena.actions.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
