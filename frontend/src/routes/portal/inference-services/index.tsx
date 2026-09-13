import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link, createFileRoute } from '@tanstack/react-router'
import { type ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import {
  ArrowUpRightIcon,
  CopyIcon,
  EllipsisVerticalIcon,
  RocketIcon,
  Trash2Icon,
} from 'lucide-react'
import { useMemo, useState } from 'react'
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
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import KthenaStatusBadge, {
  getKthenaDisplayState,
  getKthenaStatusLabel,
} from '@/components/badge/kthena-status-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import TooltipCopy from '@/components/label/tooltop-copy'
import UserLabel from '@/components/label/user-label'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { type DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  KthenaService,
  apiDeleteKthenaService,
  apiListKthenaServices,
} from '@/services/api/inference'

import { showErrorToast } from '@/utils/toast'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const Route = createFileRoute('/portal/inference-services/')({
  component: KthenaServicesPage,
  loader: () => ({ crumb: t('kthena.title') }),
})

function KthenaServicesPage() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const [serviceToDelete, setServiceToDelete] = useState<KthenaService | null>(null)
  const servicesQuery = useQuery({
    queryKey: ['kthena/inference-services'],
    queryFn: async () => {
      const services = (await apiListKthenaServices()).data
      return [...services].sort(
        (left, right) =>
          new Date(right.createdAt || 0).getTime() - new Date(left.createdAt || 0).getTime()
      )
    },
    refetchInterval: REFETCH_INTERVAL,
  })

  const { mutate: deleteService, isPending } = useMutation({
    mutationFn: apiDeleteKthenaService,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['kthena/inference-services'] })
      setServiceToDelete(null)
      toast.success(t('kthena.delete.success'))
    },
    onError: showErrorToast,
  })

  const columns = useMemo<ColumnDef<KthenaService>[]>(
    () => [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.name')} />
        ),
        cell: ({ row }) => <KthenaServiceName service={row.original} />,
      },
      {
        accessorFn: (service) => service.userInfo?.username || service.owner || '',
        id: 'owner',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.owner')} />
        ),
        cell: ({ row }) => <KthenaServiceOwner service={row.original} />,
      },
      {
        accessorKey: 'servedModel',
        id: 'model',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.model')} />
        ),
        cell: ({ row }) => {
          const service = row.original
          return (
            <TooltipCopy
              name={service.servedModel || service.modelURI}
              copyMessage={t('kthena.copy.model')}
              className="max-w-64 truncate font-mono text-xs"
            />
          )
        },
      },
      {
        accessorKey: 'backendType',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.backend')} />
        ),
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.backendType}</span>,
      },
      {
        accessorKey: 'replicas',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.replicas')} />
        ),
        cell: ({ row }) => <span className="font-mono text-xs">{row.original.replicas}</span>,
      },
      {
        id: 'resources',
        accessorFn: getWorkerResources,
        enableSorting: false,
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.detail.runtimeResources')} />
        ),
        cell: ({ row }) => <WorkerResourceBadges service={row.original} />,
      },
      {
        accessorKey: 'workerImage',
        id: 'image',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.image')} />
        ),
        cell: ({ row }) => (
          <TooltipCopy
            name={row.original.workerImage}
            copyMessage={t('kthena.copy.image')}
            className="max-w-72 truncate font-mono text-xs"
          />
        ),
      },
      {
        accessorFn: getKthenaDisplayState,
        id: 'status',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.status')} />
        ),
        cell: ({ row }) => <KthenaStatusBadge service={row.original} />,
        filterFn: (row, id, value) => (value as string[]).includes(row.getValue<string>(id)),
      },
      {
        accessorKey: 'createdAt',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('kthena.table.createdAt')} />
        ),
        cell: ({ row }) => <TimeDistance date={row.original.createdAt} />,
        sortingFn: 'datetime',
      },
      {
        id: 'actions',
        enableSorting: false,
        enableHiding: false,
        cell: ({ row }) => (
          <KthenaServiceActions
            service={row.original}
            deleting={isPending}
            onDelete={() => setServiceToDelete(row.original)}
          />
        ),
      },
    ],
    [isPending, t]
  )

  const toolbarConfig = useMemo<DataTableToolbarConfig>(
    () => ({
      filterInput: {
        key: 'name',
        placeholder: t('kthena.list.searchPlaceholder'),
      },
      filterOptions: [
        {
          key: 'status',
          title: t('kthena.table.status'),
          option: [
            { value: 'submitted', label: getKthenaStatusLabel('submitted').label },
            { value: 'scheduling', label: getKthenaStatusLabel('scheduling').label },
            { value: 'deploying', label: getKthenaStatusLabel('deploying').label },
            { value: 'running', label: getKthenaStatusLabel('running').label },
            { value: 'degraded', label: getKthenaStatusLabel('degraded').label },
            { value: 'failed', label: getKthenaStatusLabel('failed').label },
          ],
        },
      ],
      getHeader: (key) => {
        switch (key) {
          case 'name':
            return t('kthena.table.name')
          case 'owner':
            return t('kthena.table.owner')
          case 'model':
            return t('kthena.table.model')
          case 'backendType':
            return t('kthena.table.backend')
          case 'replicas':
            return t('kthena.table.replicas')
          case 'resources':
            return t('kthena.detail.runtimeResources')
          case 'image':
            return t('kthena.table.image')
          case 'status':
            return t('kthena.table.status')
          case 'createdAt':
            return t('kthena.table.createdAt')
          case 'actions':
            return t('kthena.table.actions')
          default:
            return key
        }
      },
    }),
    [t]
  )

  return (
    <>
      <DataTable
        info={{ title: t('kthena.title'), description: t('kthena.list.description') }}
        storageKey="portal_kthena_inference_services"
        query={servicesQuery}
        columns={columns}
        toolbarConfig={toolbarConfig}
        initialColumnVisibility={{ image: false }}
        withI18n
      >
        <Button asChild>
          <Link to="/portal/inference-services/new" search={{ clone: undefined }}>
            <RocketIcon className="size-4" />
            {t('kthena.actions.create')}
          </Link>
        </Button>
      </DataTable>
      <AlertDialog
        open={serviceToDelete !== null}
        onOpenChange={(open) => !open && setServiceToDelete(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('kthena.delete.title')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('kthena.delete.description', { name: serviceToDelete?.name })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={isPending}>{t('kthena.actions.cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
              disabled={isPending || serviceToDelete === null}
              onClick={() => serviceToDelete && deleteService(serviceToDelete.name)}
            >
              {t('kthena.actions.delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  )
}

function KthenaServiceName({ service }: { service: KthenaService }) {
  return (
    <div className="flex min-w-0 flex-col gap-0.5">
      <Link
        to="/portal/inference-services/$name"
        params={{ name: service.name }}
        className="hover:text-primary truncate font-medium"
      >
        {service.name}
      </Link>
      <span className="text-muted-foreground truncate text-xs">{service.namespace}</span>
    </div>
  )
}

function KthenaServiceOwner({ service }: { service: KthenaService }) {
  const userInfo = service.userInfo?.username
    ? service.userInfo
    : service.owner
      ? { username: service.owner, nickname: '' }
      : null
  return userInfo ? <UserLabel info={userInfo} /> : <span className="text-muted-foreground">-</span>
}

function KthenaServiceActions({
  service,
  deleting,
  onDelete,
}: {
  service: KthenaService
  deleting: boolean
  onDelete: () => void
}) {
  const { t } = useTranslation()
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="icon" disabled={deleting} title={t('kthena.actions.more')}>
          <EllipsisVerticalIcon className="size-4" />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        <DropdownMenuLabel className="text-muted-foreground text-xs">
          {service.name}
        </DropdownMenuLabel>
        <DropdownMenuItem asChild>
          <Link to="/portal/inference-services/$name" params={{ name: service.name }}>
            <ArrowUpRightIcon className="size-4" />
            {t('kthena.actions.view')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link to="/portal/inference-services/new" search={{ clone: service.name }}>
            <CopyIcon className="size-4" />
            {t('kthena.actions.clone')}
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem variant="destructive" onClick={onDelete}>
          <Trash2Icon className="size-4" />
          {t('kthena.actions.deleteDeployment')}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  )
}

function WorkerResourceBadges({ service }: { service: KthenaService }) {
  const resources = getWorkerResources(service)
  if (Object.keys(resources).length === 0) {
    return <span className="text-muted-foreground text-xs">-</span>
  }
  return <ResourceBadges resources={resources} />
}

function getWorkerResources(service: KthenaService) {
  const resources: Record<string, string> = {}
  if (service.workerCPU) resources.cpu = service.workerCPU
  if (service.workerMemory) resources.memory = service.workerMemory
  if (service.workerGPU && service.workerGPU !== '0') {
    resources[service.workerGPUModel || 'gpu'] = service.workerGPU
  }
  return resources
}
