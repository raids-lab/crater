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
import { format } from 'date-fns'
import { type Locale, zhCN } from 'date-fns/locale'
import { type FC, type ReactNode } from 'react'

import { Badge } from '@/components/ui/badge'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import NodeStatusBadge, { getNodeStatusMeta } from '@/components/badge/node-status-badge'

import { type IClusterNodeTaint, type INodeBriefInfo, NodeStatus } from '@/services/api/cluster'

import { cn } from '@/lib/utils'

import { deriveNodeStatus } from './node-status-utils'

const formatNodeStatusTime = (time: string | undefined, locale?: Locale): string | undefined => {
  if (!time) return undefined
  const date = new Date(time)
  if (Number.isNaN(date.getTime())) return undefined
  return format(date, 'PPPp', { locale: locale || zhCN })
}

interface DetailLine {
  label: string
  value?: string
}

/** 只渲染非空行；全部为空时回退到 fallback 文案 */
const TooltipLines: FC<{ title?: string; lines: DetailLine[]; fallback?: string }> = ({
  title,
  lines,
  fallback,
}) => {
  const visible = lines.filter((l) => l.value)
  return (
    <div className="flex flex-col gap-0.5">
      {title && <div>{title}</div>}
      {visible.map((line) => (
        <div key={line.label}>
          {line.label}：{line.value}
        </div>
      ))}
      {visible.length === 0 && fallback && <div>{fallback}</div>}
    </div>
  )
}

const ConditionDetails: FC<{
  status: NodeStatus
  taint?: IClusterNodeTaint
  locale?: Locale
}> = ({ status, taint, locale }) => {
  const { label, description } = getNodeStatusMeta(status)
  return (
    <TooltipLines
      title={`${label}：${description}`}
      lines={[
        { label: '污点', value: taint?.key },
        { label: '时间', value: formatNodeStatusTime(taint?.timeAdded, locale) },
      ]}
    />
  )
}

/** 原生 PhaseBadge 样式的短词徽章，外包一层自定义 Tooltip */
const StatusPill: FC<{ status: NodeStatus; tooltip: ReactNode }> = ({ status, tooltip }) => (
  <Tooltip>
    <TooltipTrigger asChild>
      <span className="inline-flex cursor-help" tabIndex={0}>
        <NodeStatusBadge status={status} disableDefaultTooltip={true} />
      </span>
    </TooltipTrigger>
    <TooltipContent>{tooltip}</TooltipContent>
  </Tooltip>
)

const SecondaryStatusPill: FC<{ status: NodeStatus; tooltip: ReactNode }> = ({
  status,
  tooltip,
}) => {
  const meta = getNodeStatusMeta(status)
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <Badge
          variant="outline"
          className={cn(
            'cursor-help rounded-full border-none px-2 py-0 text-[11px] leading-4 font-medium',
            meta.color
          )}
          tabIndex={0}
        >
          {meta.label}
        </Badge>
      </TooltipTrigger>
      <TooltipContent>{tooltip}</TooltipContent>
    </Tooltip>
  )
}

export const NodeStatusCell: FC<{
  node: INodeBriefInfo
  locale?: Locale
}> = ({ node, locale }) => {
  const { health, pressures, unschedulable, exclusives } = deriveNodeStatus(node)
  const workloads = node.workloads ?? 0

  const accounts = exclusives.flatMap(({ account }) => (account ? [account] : []))
  const [occupied] = exclusives

  return (
    <div className="flex flex-col items-start gap-1">
      <div className="flex items-center gap-1">
        <StatusPill
          status={health.state}
          tooltip={
            health.state === NodeStatus.Ready ? (
              `${workloads} 个作业正在运行`
            ) : (
              <ConditionDetails status={health.state} taint={health.taint} locale={locale} />
            )
          }
        />

        {(health.state === NodeStatus.Ready || workloads > 0) && (
          <Tooltip>
            <TooltipTrigger
              className={cn(
                'bg-secondary text-secondary-foreground flex size-4 shrink-0 items-center justify-center rounded-full text-xs hover:cursor-help',
                { 'bg-primary text-primary-foreground': workloads > 0 }
              )}
            >
              {workloads}
            </TooltipTrigger>
            <TooltipContent>{workloads} 个作业正在运行</TooltipContent>
          </Tooltip>
        )}
      </div>

      {(pressures.length > 0 || unschedulable || occupied) && (
        <div className="flex flex-wrap items-center gap-1">
          {pressures.map(({ state, taint }) => (
            <SecondaryStatusPill
              key={state}
              status={state}
              tooltip={<ConditionDetails status={state} taint={taint} locale={locale} />}
            />
          ))}

          {unschedulable && (
            <SecondaryStatusPill
              status={NodeStatus.Unschedulable}
              tooltip={
                <TooltipLines
                  lines={[
                    { label: '操作者', value: unschedulable.operator },
                    { label: '原因', value: unschedulable.reason },
                    { label: '时间', value: formatNodeStatusTime(unschedulable.timeAdded, locale) },
                  ]}
                  fallback={getNodeStatusMeta(NodeStatus.Unschedulable).description}
                />
              }
            />
          )}

          {occupied && (
            <SecondaryStatusPill
              status={NodeStatus.Occupied}
              tooltip={
                <TooltipLines
                  lines={[
                    { label: '账户', value: accounts.join('、') },
                    { label: '操作者', value: occupied.operator },
                    { label: '原因', value: occupied.reason },
                    ...exclusives.map(({ account, timeAdded }) => ({
                      label: account && accounts.length > 1 ? `时间（${account}）` : '时间',
                      value: formatNodeStatusTime(timeAdded, locale),
                    })),
                  ]}
                  fallback={getNodeStatusMeta(NodeStatus.Occupied).description}
                />
              }
            />
          )}
        </div>
      )}
    </div>
  )
}
