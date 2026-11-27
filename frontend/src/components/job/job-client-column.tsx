/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { ColumnDef } from '@tanstack/react-table'
import { t } from 'i18next'
import { useTranslation } from 'react-i18next'

import { Badge } from '@/components/ui/badge'

import SimpleTooltip from '@/components/label/simple-tooltip'
import { DataTableColumnHeader } from '@/components/query-table/column-header'

import { IJobInfo, JobClientType } from '@/services/api/vcjob'

import { cn } from '@/lib/utils'

const clientBadgeClass: Record<JobClientType, string> = {
  cli: 'border-transparent bg-highlight-sky/10 text-highlight-sky',
  web: 'border-transparent bg-highlight-green/10 text-highlight-green',
  unknown: 'border-transparent bg-muted text-muted-foreground',
}

interface JobClientLabelProps {
  clientType?: JobClientType
  clientVersion?: string
  withVersion?: boolean
}

export function JobClientLabel({
  clientType,
  clientVersion,
  withVersion = false,
}: JobClientLabelProps) {
  const { t: translate } = useTranslation()
  const type = clientType ?? 'unknown'

  if (type === 'cli') {
    return (
      <span>
        {translate('jobs.client.cli')}
        {withVersion && clientVersion ? ` ${clientVersion}` : ''}
      </span>
    )
  }
  if (type === 'web') {
    return <span>{translate('jobs.client.web')}</span>
  }
  return <span>{translate('jobs.client.unknown')}</span>
}

interface JobClientCellProps {
  clientType?: JobClientType
  clientVersion?: string
}

export function JobClientCell({ clientType, clientVersion }: JobClientCellProps) {
  const { t: translate } = useTranslation()
  const type = clientType ?? 'unknown'
  const label = (
    <Badge variant="outline" className={cn('font-normal', clientBadgeClass[type])}>
      <JobClientLabel clientType={type} />
    </Badge>
  )

  if (type !== 'cli') {
    return label
  }

  const tooltip = clientVersion
    ? translate('jobs.client.cliVersionTooltip', { version: clientVersion })
    : translate('jobs.client.cliTooltip')
  return <SimpleTooltip tooltip={tooltip}>{label}</SimpleTooltip>
}

export const jobClientColumn = (): ColumnDef<IJobInfo> => ({
  accessorKey: 'clientType',
  enableSorting: false,
  header: ({ column }) => <DataTableColumnHeader column={column} title={t('jobs.clientType')} />,
  cell: ({ row }) => (
    <JobClientCell
      clientType={row.original.clientType}
      clientVersion={row.original.clientVersion}
    />
  ),
})
