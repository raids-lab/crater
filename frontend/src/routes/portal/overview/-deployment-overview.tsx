import { useQuery } from '@tanstack/react-query'
import { Link } from '@tanstack/react-router'
import { type ColumnDef } from '@tanstack/react-table'
import { useAtomValue } from 'jotai'
import { ArrowUpRightIcon, BoxesIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Progress } from '@/components/ui/progress'

import { getKthenaStatusLabel } from '@/components/badge/kthena-status-badge'
import { PhaseBadge } from '@/components/badge/phase-badge'
import ResourceBadges from '@/components/badge/resource-badges'
import { TimeDistance } from '@/components/custom/time-distance'
import { DataTable } from '@/components/query-table'
import { DataTableColumnHeader } from '@/components/query-table/column-header'
import { type DataTableToolbarConfig } from '@/components/query-table/toolbar'

import {
  type KthenaDeploymentOverview,
  apiListKthenaDeploymentOverview,
} from '@/services/api/inference'
import { apiGetKthenaInferenceStatus } from '@/services/api/system-config'

import { atomUserContext, atomUserInfo } from '@/utils/store'

import { REFETCH_INTERVAL } from '@/lib/constants'

import { OverviewSection } from './-overview-section'

const deploymentStatuses = [
  'callable',
  'deploying',
  'routing-missing',
  'degraded',
  'queued',
  'suspended',
  'failed',
] as const

type DeploymentStatus = (typeof deploymentStatuses)[number]

function getDeploymentStatus(deployment: KthenaDeploymentOverview): DeploymentStatus {
  if (deployment.phase === 'Suspended') return 'suspended'
  if (deployment.phase === 'Prequeue') return 'queued'
  if (deployment.phase === 'Invalid' || deployment.phase === 'Failed') return 'failed'
  if (!deployment.routingConfigured) return 'routing-missing'
  if (deployment.phase === 'Degraded') return 'degraded'
  if ((deployment.readyGroups ?? 0) > 0) return 'callable'
  return 'deploying'
}

export function DeploymentOverview() {
  const { t } = useTranslation()
  const user = useAtomValue(atomUserInfo)
  const account = useAtomValue(atomUserContext)
  const { data: feature } = useQuery({
    queryKey: ['system-config', 'kthena-inference'],
    queryFn: () => apiGetKthenaInferenceStatus().then((res) => res.data),
    retry: false,
    staleTime: 30_000,
  })
  const deployments = useQuery({
    queryKey: ['kthena/inference-services', 'overview', user?.id, account?.space],
    queryFn: async ({ signal }) => {
      const { data } = await apiListKthenaDeploymentOverview(signal)
      return data.sort((left, right) => Date.parse(right.createdAt) - Date.parse(left.createdAt))
    },
    enabled: feature?.enabled === true,
    refetchInterval: REFETCH_INTERVAL,
  })

  const { columns, toolbarConfig } = useMemo(() => {
    const headers: Record<string, string> = {
      name: t('kthena.table.name'),
      model: t('kthena.table.model'),
      accountName: t('overview.deployments.account'),
      readyGroups: t('overview.deployments.readyGroups'),
      status: t('overview.deployments.serviceStatus'),
      resources: t('overview.deployments.requestedResources'),
      createdAt: t('kthena.table.createdAt'),
    }
    const statusLabel = (status: DeploymentStatus) => {
      if (status === 'callable') {
        return {
          label: t('overview.deployments.callable'),
          description: t('overview.deployments.callableHint'),
          color: 'bg-highlight-emerald/20 text-highlight-emerald',
        }
      }
      if (status === 'routing-missing') {
        return {
          label: t('overview.deployments.routingMissing'),
          description: t('overview.deployments.routingMissingHint'),
          color: 'bg-highlight-orange/20 text-highlight-orange',
        }
      }
      return getKthenaStatusLabel(status)
    }
    const columns: ColumnDef<KthenaDeploymentOverview>[] = [
      {
        accessorKey: 'name',
        cell: ({ row }) => {
          const { name, canOpenDetail } = row.original
          return canOpenDetail ? (
            <Link
              className="text-primary inline-flex items-center gap-1.5 font-medium hover:underline"
              to="/portal/inference-services/$name"
              params={{ name }}
            >
              {name}
              <ArrowUpRightIcon className="size-3.5 shrink-0" aria-hidden="true" />
            </Link>
          ) : (
            name
          )
        },
      },
      {
        accessorKey: 'model',
        cell: ({ row }) => (
          <span className="text-muted-foreground font-mono text-xs">
            {row.original.model || '—'}
          </span>
        ),
      },
      { accessorKey: 'accountName', cell: ({ row }) => row.original.accountName || '—' },
      {
        accessorKey: 'readyGroups',
        enableSorting: false,
        cell: ({ row }) => {
          const { readyGroups, desiredGroups } = row.original
          return (
            <div className="flex items-center gap-2.5 whitespace-nowrap">
              <span className="font-medium tabular-nums">
                {readyGroups ?? '—'} / {desiredGroups}
              </span>
              {readyGroups !== null && desiredGroups > 0 && (
                <Progress
                  aria-label={headers.readyGroups}
                  value={Math.min(100, (readyGroups / desiredGroups) * 100)}
                  className="h-1.5 w-14 shrink-0"
                />
              )}
            </div>
          )
        },
      },
      {
        id: 'status',
        accessorFn: getDeploymentStatus,
        cell: ({ row }) => (
          <PhaseBadge phase={getDeploymentStatus(row.original)} getPhaseLabel={statusLabel} />
        ),
        filterFn: (row, columnId, values: DeploymentStatus[]) =>
          values.includes(row.getValue(columnId)),
      },
      {
        accessorKey: 'resources',
        enableSorting: false,
        cell: ({ row }) => <ResourceBadges resources={row.original.resources ?? undefined} />,
      },
      {
        accessorKey: 'createdAt',
        sortingFn: 'datetime',
        cell: ({ row }) => <TimeDistance date={row.original.createdAt} />,
      },
    ]
    const toolbarConfig: DataTableToolbarConfig = {
      globalSearch: { enabled: true, placeholder: t('overview.deployments.search') },
      filterOptions: [
        {
          key: 'status',
          title: headers.status,
          option: deploymentStatuses.map((value) => ({ value, label: statusLabel(value).label })),
        },
      ],
      getHeader: (key) => headers[key] ?? key,
    }
    for (const definition of columns) {
      definition.header = ({ column }) => (
        <DataTableColumnHeader column={column} title={headers[column.id]} />
      )
    }
    return { columns, toolbarConfig }
  }, [t])

  if (!feature?.enabled) return null

  return (
    <OverviewSection
      title={t('overview.deployments.title')}
      description={t('overview.deployments.description')}
      icon={BoxesIcon}
      accent="bg-highlight-purple/15 text-highlight-purple"
      action={
        <Button variant="outline" size="sm" asChild>
          <Link to="/portal/inference-services">
            {t('overview.deployments.mine')}
            <ArrowUpRightIcon className="size-4" aria-hidden="true" />
          </Link>
        </Button>
      }
    >
      <DataTable
        storageKey="portal-overview-deployments"
        query={deployments}
        columns={columns}
        toolbarConfig={toolbarConfig}
        withI18n
      />
    </OverviewSection>
  )
}
