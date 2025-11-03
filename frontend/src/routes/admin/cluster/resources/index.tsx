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
import { useMutation, useQuery } from '@tanstack/react-query'
import { useQueryClient } from '@tanstack/react-query'
import { createFileRoute } from '@tanstack/react-router'
import { ColumnDef } from '@tanstack/react-table'
import { EllipsisVerticalIcon as DotsHorizontalIcon } from 'lucide-react'
import { BoxIcon, CpuIcon, NetworkIcon, RefreshCcwIcon, TagIcon, Trash2Icon } from 'lucide-react'
import { type FC, useMemo, useState } from 'react'
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
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'

import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  Resource,
  apiAdminResourceDelete,
  apiAdminResourceSync,
  apiAdminResourceVGPUList,
  apiResourceList,
  apiResourceNetworks,
} from '@/services/api/resource'

import { formatBytes } from '@/utils/formatter'

import { UpdateResourceForm } from './-components/form'
import {
  NetworkAssociationForm,
  UpdateResourceTypeForm,
  VGPUAssociationForm,
} from './-components/form'

export const Route = createFileRoute('/admin/cluster/resources/')({
  component: Resources,
})

// New component for Networks cell
const NetworksCell: FC<{ resourceId: number; resourceType?: string }> = ({
  resourceId,
  resourceType,
}) => {
  const networksQuery = useQuery({
    queryKey: ['resource', 'networks', resourceId],
    queryFn: () => apiResourceNetworks(resourceId),
    enabled: resourceType === 'gpu',
    select: (res) => res.data,
  })

  if (resourceType !== 'gpu' || !networksQuery.data?.length) {
    return <></>
  }

  return (
    <div className="flex flex-wrap gap-1">
      {networksQuery.data.map((network) => (
        <Badge key={network.ID} variant="outline" className="font-mono">
          {network.name}
        </Badge>
      ))}
    </div>
  )
}

// New component for VGPU associations cell
const VGPUCell: FC<{ resourceId: number; resourceType?: string }> = ({
  resourceId,
  resourceType,
}) => {
  const vgpuQuery = useQuery({
    queryKey: ['resource', 'vgpu', resourceId],
    queryFn: () => apiAdminResourceVGPUList(resourceId),
    enabled: resourceType === 'gpu',
    select: (res) => res.data,
  })

  if (resourceType !== 'gpu' || !vgpuQuery.data?.length) {
    return <></>
  }

  return (
    <div className="flex flex-wrap gap-1">
      {vgpuQuery.data.map((vgpu) => (
        <Badge key={vgpu.ID} variant="outline" className="font-mono">
          {vgpu.vgpuResource?.name || `VGPU-${vgpu.ID}`}
          {vgpu.description && (
            <span className="ml-1 text-xs opacity-75">({vgpu.description})</span>
          )}
        </Badge>
      ))}
    </div>
  )
}

// New component for Actions cell
const ActionsCell: FC<{ resource: Resource }> = ({ resource }) => {
  const { t } = useTranslation()
  const [openLabelSheet, setOpenLabelSheet] = useState(false)
  const [openTypeSheet, setOpenTypeSheet] = useState(false)
  const [openNetworkSheet, setOpenNetworkSheet] = useState(false)
  const [openVGPUSheet, setOpenVGPUSheet] = useState(false)
  const queryClient = useQueryClient()

  const { mutate: deleteResource } = useMutation({
    mutationFn: () => apiAdminResourceDelete(resource.ID),
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: ['resource', 'list'],
      })
      toast.success(t('resources.delete.success'))
    },
  })

  return (
    <>
      <AlertDialog>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" className="h-8 w-8 p-0">
              <span className="sr-only">{t('resources.actions.menu')}</span>
              <DotsHorizontalIcon className="size-4" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            <DropdownMenuLabel className="text-muted-foreground text-xs">
              {t('resources.actions.title')}
            </DropdownMenuLabel>
            <DropdownMenuItem onClick={() => setOpenLabelSheet(true)}>
              <TagIcon />
              {t('resources.actions.editLabels')}
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => setOpenTypeSheet(true)}>
              <BoxIcon />
              {t('resources.actions.editType')}
            </DropdownMenuItem>
            {resource.type === 'gpu' && (
              <DropdownMenuItem onClick={() => setOpenNetworkSheet(true)}>
                <NetworkIcon />
                {t('resources.actions.associateNetworks')}
              </DropdownMenuItem>
            )}
            {resource.type === 'gpu' && (
              <DropdownMenuItem onClick={() => setOpenVGPUSheet(true)}>
                <CpuIcon />
                {t('resources.actions.associateVGPU')}
              </DropdownMenuItem>
            )}
            <AlertDialogTrigger asChild>
              <DropdownMenuItem>
                <Trash2Icon className="text-destructive" />
                {t('resources.actions.delete')}
              </DropdownMenuItem>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>{t('resources.delete.title')}</AlertDialogTitle>
                <AlertDialogDescription>
                  {t('resources.delete.confirm', { name: resource.name })}
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => deleteResource()}
                  className="bg-destructive text-destructive-foreground"
                >
                  {t('resources.delete.confirmAction')}
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </DropdownMenuContent>
        </DropdownMenu>
      </AlertDialog>
      <UpdateResourceForm
        current={resource}
        open={openLabelSheet}
        onOpenChange={setOpenLabelSheet}
      />
      <UpdateResourceTypeForm
        current={resource}
        open={openTypeSheet}
        onOpenChange={setOpenTypeSheet}
      />
      {resource.type === 'gpu' && (
        <NetworkAssociationForm
          gpuResource={resource}
          open={openNetworkSheet}
          onOpenChange={setOpenNetworkSheet}
        />
      )}
      {resource.type === 'gpu' && (
        <VGPUAssociationForm
          gpuResource={resource}
          open={openVGPUSheet}
          onOpenChange={setOpenVGPUSheet}
        />
      )}
    </>
  )
}

function Resources() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()

  const toolbarConfig: DataTableToolbarConfig = useMemo(() => {
    return {
      filterInput: {
        placeholder: t('resources.filter.placeholder'),
        key: 'name',
      },
      filterOptions: [],
      getHeader: (x) => x,
    }
  }, [t])

  const columns: ColumnDef<Resource>[] = useMemo(() => {
    return [
      {
        accessorKey: 'name',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.name')} />
        ),
        cell: ({ row }) => {
          return (
            <div>
              <p className="font-mono">{row.original.label}</p>
              <p className="text-muted-foreground font-mono text-xs font-normal">
                {row.getValue<string>('name')}
              </p>
            </div>
          )
        },
      },
      {
        accessorKey: 'type',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.type')} />
        ),
        cell: ({ row }) => {
          const type = row.getValue<string>('type')
          if (!type) {
            return <></>
          }
          return (
            <Badge variant="secondary">
              {type === 'gpu'
                ? t('resources.type.gpu')
                : type === 'rdma'
                  ? t('resources.type.rdma')
                  : type === 'vgpu'
                    ? t('resources.type.vgpu')
                    : type}
            </Badge>
          )
        },
      },
      {
        id: 'networks',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.networks')} />
        ),
        cell: ({ row }) => (
          <NetworksCell resourceId={row.original.ID} resourceType={row.original.type} />
        ),
      },
      {
        id: 'vgpu',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.vgpu')} />
        ),
        cell: ({ row }) => (
          <VGPUCell resourceId={row.original.ID} resourceType={row.original.type} />
        ),
      },
      {
        accessorKey: 'amount',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.total')} />
        ),
        cell: ({ row }) => {
          const amount = row.getValue<number>('amount')
          if (amount == 0) {
            return
          }
          if (amount > 1024 * 1024) {
            return (
              <Badge className="font-mono" variant="secondary">
                {formatBytes(amount)}
              </Badge>
            )
          }
          return (
            <Badge className="font-mono" variant="secondary">
              {row.getValue('amount')}
            </Badge>
          )
        },
      },
      {
        accessorKey: 'amountSingleMax',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.maxPerNode')} />
        ),
        cell: ({ row }) => {
          const amount = row.getValue<number>('amountSingleMax')
          if (amount == 0) {
            return
          }
          if (amount > 1024 * 1024) {
            return (
              <Badge className="font-mono" variant="secondary">
                {formatBytes(amount)}
              </Badge>
            )
          }
          return (
            <Badge className="font-mono" variant="secondary">
              {row.getValue('amountSingleMax')}
            </Badge>
          )
        },
      },
      {
        accessorKey: 'format',
        header: ({ column }) => (
          <DataTableColumnHeader column={column} title={t('resources.columns.format')} />
        ),
        cell: ({ row }) => (
          <Badge className="font-mono font-normal" variant="outline">
            {row.getValue('format')}
          </Badge>
        ),
      },
      {
        id: 'actions',
        enableHiding: false,
        cell: ({ row }) => <ActionsCell resource={row.original} />,
      },
    ]
  }, [t])

  const query = useQuery({
    queryKey: ['resource', 'list'],
    queryFn: () => apiResourceList(false),
    select: (res) => {
      return res.data.sort((a, b) => a.name.localeCompare(b.name))
    },
  })

  const { mutate: syncNvidiaLabel } = useMutation({
    mutationFn: apiAdminResourceSync,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['resource', 'list'] })
      toast.success(t('resources.sync.success'))
    },
  })

  return (
    <DataTable
      query={query}
      columns={columns}
      toolbarConfig={toolbarConfig}
      info={{
        title: t('resources.info.title'),
        description: t('resources.info.description'),
      }}
      storageKey="admin_resource_list"
    >
      <AlertDialog>
        <AlertDialogTrigger asChild>
          <Button>
            <RefreshCcwIcon />
            {t('resources.sync.button')}
          </Button>
        </AlertDialogTrigger>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('resources.sync.title')}</AlertDialogTitle>
            <AlertDialogDescription>{t('resources.sync.confirm')}</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t('common.cancel')}</AlertDialogCancel>
            <AlertDialogAction onClick={() => syncNvidiaLabel()}>
              {t('common.confirm')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </DataTable>
  )
}

export default Resources
