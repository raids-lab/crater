import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import {
  ActivityIcon,
  AlertTriangleIcon,
  ArrowLeftIcon,
  BracesIcon,
  CalendarIcon,
  CheckCircle2Icon,
  CopyIcon,
  CpuIcon,
  GaugeIcon,
  MessageSquareIcon,
  NetworkIcon,
  ServerIcon,
  TerminalIcon,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import KthenaStatusBadge from '@/components/badge/kthena-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import CardTitle from '@/components/label/card-title'
import UserLabel from '@/components/label/user-label'
import PageTitle from '@/components/layout/page-title'
import NotFound from '@/components/placeholder/not-found'

import {
  apiDeleteKthenaService,
  apiGetKthenaDiagnostics,
  apiGetKthenaService,
  apiRepairKthenaService,
} from '@/services/api/inference'

import { getServingResources } from '@/hooks/inference/format'
import { useInferenceKeys } from '@/hooks/inference/use-inference-keys'

import { showErrorToast } from '@/utils/toast'

import { REFETCH_INTERVAL } from '@/lib/constants'

import {
  CopyValue,
  DiagnosticCard,
  DiagnosticSummary,
  KthenaDetailMeta,
  OverviewField,
  OverviewSection,
  ResourceCard,
  RuntimePanel,
  displayAPIBaseURL,
} from './-components/detail-parts'
import { InferenceMetrics } from './-components/inference-metrics'
import { InvokeWorkspace } from './-components/invoke-workspace'
import ResourceUsage from './-components/resource-usage'
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
  const [deleteOpen, setDeleteOpen] = useState(false)
  const { data, isLoading, error, refetch } = useQuery({
    queryKey: keys.detail(name),
    queryFn: () => apiGetKthenaService(name).then((res) => res.data),
    refetchInterval: REFETCH_INTERVAL,
  })

  const [tab, setTab] = useState('')
  const [repairOpen, setRepairOpen] = useState(false)
  const diagnosticsQuery = useQuery({
    queryKey: [...keys.detail(name), 'diagnostics'],
    queryFn: () => apiGetKthenaDiagnostics(name).then((res) => res.data),
    enabled: !!data && (tab === 'diagnostics' || (!tab && data.phase !== 'Ready')),
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
  const primaryPod = service.runtimePods?.find((pod) => pod.ready) ?? service.runtimePods?.[0]

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

      <div className="text-muted-foreground grid grid-cols-1 gap-3 text-sm sm:grid-cols-2 md:grid-cols-4">
        <KthenaDetailMeta icon={ActivityIcon} label={t('kthena.table.status')}>
          <KthenaStatusBadge service={service} />
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={ServerIcon} label={t('kthena.table.backend')}>
          <span className="text-foreground font-medium">{service.backendType || '-'}</span>
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CheckCircle2Icon} label={t('kthena.detail.servedModel')}>
          <span className="text-foreground truncate font-mono text-sm" title={service.servedModel}>
            {service.servedModel || '-'}
          </span>
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CpuIcon} label={t('kthena.detail.runtimeResources')}>
          {Object.keys(workerResources).length ? (
            <ResourceBadges resources={workerResources} />
          ) : (
            <span>-</span>
          )}
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={UserRoundIcon} label={t('kthena.table.owner')}>
          {service.userInfo?.username ? (
            <UserLabel info={service.userInfo} />
          ) : (
            <span className="truncate">{service.owner || '-'}</span>
          )}
        </KthenaDetailMeta>
        <KthenaDetailMeta icon={CalendarIcon} label={t('kthena.table.createdAt')}>
          <TimeDistance date={service.createdAt} />
        </KthenaDetailMeta>
      </div>

      <Tabs
        onValueChange={setTab}
        defaultValue={service.phase === 'Ready' ? 'invoke' : 'diagnostics'}
        className="gap-0"
      >
        <TabsList className="tabs-list-underline">
          <TabsTrigger className="tabs-trigger-underline" value="invoke">
            <MessageSquareIcon className="size-4" />
            {t('kthena.detail.tabs.invoke')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="overview">
            <NetworkIcon className="size-4" />
            {t('kthena.detail.tabs.overview')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="resources">
            <BracesIcon className="size-4" />
            {t('kthena.detail.tabs.resources')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="usage">
            <GaugeIcon className="size-4" />
            {t('kthena.detail.tabs.usage')}
          </TabsTrigger>
          <TabsTrigger className="tabs-trigger-underline" value="diagnostics">
            <AlertTriangleIcon className="size-4" />
            {t('kthena.detail.tabs.diagnostics')}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="overview" className="mt-0">
          <div className="grid items-start gap-4 xl:grid-cols-[minmax(0,1fr)_22rem]">
            <div className="bg-card overflow-hidden rounded-xl border">
              <OverviewSection icon={TerminalIcon} title={t('kthena.detail.invokeInfo')}>
                <OverviewField label={t('kthena.detail.apiBase')}>
                  <CopyValue value={displayAPIBaseURL(service)} />
                </OverviewField>
              </OverviewSection>
              <OverviewSection
                icon={NetworkIcon}
                title={t('kthena.detail.overviewTitle')}
                className="border-t"
              >
                <div className="grid gap-2 sm:grid-cols-2">
                  <OverviewField label={t('kthena.detail.routeModelName')}>
                    <CopyValue value={service.access?.modelName || service.name} />
                  </OverviewField>
                  <OverviewField label={t('kthena.detail.servedModel')}>
                    <CopyValue value={service.servedModel || '-'} />
                  </OverviewField>
                  <OverviewField label={t('kthena.detail.routeResource')}>
                    <CopyValue
                      value={service.access?.routeName || t('kthena.detail.waitingModelRoute')}
                    />
                  </OverviewField>
                  <OverviewField label="Router">
                    <CopyValue value={service.access?.routerService || '-'} />
                  </OverviewField>
                </div>
              </OverviewSection>
            </div>
            <RuntimePanel service={service} primaryPod={primaryPod} resources={workerResources} />
            <ServingRuntime service={service} />
          </div>
        </TabsContent>

        <TabsContent value="resources" className="mt-0">
          <Card>
            <CardHeader>
              <CardTitle icon={BracesIcon}>{t('kthena.detail.tabs.resources')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3 md:grid-cols-2">
              {service.resources?.length ? (
                service.resources.map((resource) => (
                  <ResourceCard key={`${resource.kind}/${resource.name}`} resource={resource} />
                ))
              ) : (
                <div className="text-muted-foreground text-sm">
                  {t('kthena.detail.noResources')}
                </div>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="usage" className="mt-0">
          <InferenceMetrics name={service.name} />
          <ResourceUsage runtimePods={service.runtimePods} />
        </TabsContent>

        <TabsContent value="diagnostics" className="mt-0">
          {diagnosticsQuery.error && <p role="alert">{diagnosticsQuery.error.message}</p>}
          <Button variant="outline" onClick={() => void diagnosticsQuery.refetch()}>
            {t('kthena.actions.retry')}
          </Button>
          {data?.diagnostics?.some((item) => item.reason === 'RoutingResourcesMissing') && (
            <Button onClick={() => setRepairOpen(true)}>{t('kthena.actions.repair')}</Button>
          )}
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

          <Card>
            <CardHeader>
              <CardTitle icon={AlertTriangleIcon}>{t('kthena.detail.diagnosticsTitle')}</CardTitle>
            </CardHeader>
            <CardContent className="grid gap-3">
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
                  {t('kthena.detail.noDiagnostics')}
                </div>
              )}
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
