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
import {
  ActivityIcon,
  Cable,
  CpuIcon,
  HardDriveIcon,
  LucideIcon,
  MemoryStick,
  ServerIcon,
  Zap,
} from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Cell, Label, Pie, PieChart, ResponsiveContainer } from 'recharts'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'

import AcceleratorBadge from '@/components/badge/accelerator-badge'

import { IClusterNodeDetail, IClusterNodeGPU } from '@/services/api/cluster'

import { convertKResourceToResource } from '@/utils/resource'

import { cn } from '@/lib/utils'

type NodeInfoTabProps = {
  nodeDetail?: IClusterNodeDetail
  gpuDetail?: IClusterNodeGPU
}

function NodeInfoField({
  icon: Icon,
  label,
  value,
}: {
  icon: LucideIcon
  label: string
  value: string
}) {
  return (
    <div className="min-w-0 space-y-2">
      <dt className="text-muted-foreground flex items-start gap-2 text-sm">
        <Icon className="mt-0.5 size-4 shrink-0" />
        <span className="min-w-0 [overflow-wrap:anywhere]">{label}</span>
      </dt>
      <dd>
        <Badge
          variant="secondary"
          className="max-w-full min-w-0 font-mono text-xs [overflow-wrap:anywhere] whitespace-normal"
        >
          {value}
        </Badge>
      </dd>
    </div>
  )
}

export const NodeInfoTab = ({ nodeDetail, gpuDetail }: NodeInfoTabProps) => {
  const { t } = useTranslation()
  const [isDataLoaded, setIsDataLoaded] = useState(false)

  useEffect(() => {
    if (nodeDetail?.used?.cpu !== undefined && nodeDetail?.capacity?.cpu !== undefined) {
      const timer = setTimeout(() => setIsDataLoaded(true), 50)
      return () => clearTimeout(timer)
    } else {
      setIsDataLoaded(false)
    }
  }, [nodeDetail?.used?.cpu, nodeDetail?.capacity?.cpu])

  if (!nodeDetail) return null

  const cpuUsedValue = nodeDetail.used?.cpu
    ? convertKResourceToResource('cpu', nodeDetail.used.cpu)
    : undefined
  const cpuCapacityValue = nodeDetail.capacity?.cpu
    ? convertKResourceToResource('cpu', nodeDetail.capacity.cpu)
    : undefined
  const cpuCapacity = cpuCapacityValue !== undefined ? cpuCapacityValue.toFixed(2) : '-'
  const cpuPercentage =
    cpuUsedValue !== undefined && cpuCapacityValue !== undefined && cpuCapacityValue > 0
      ? Math.round((cpuUsedValue / cpuCapacityValue) * 100)
      : 0

  const cpuChartData = isDataLoaded
    ? [
        { name: 'Used', value: cpuUsedValue || 0 },
        { name: 'Free', value: Math.max(0, (cpuCapacityValue || 0) - (cpuUsedValue || 0)) },
      ]
    : [
        { name: 'Used', value: 0 },
        { name: 'Free', value: cpuCapacityValue || 0 },
      ]

  const memoryUsedGi = nodeDetail.used?.memory
    ? convertKResourceToResource('memory', nodeDetail.used.memory)
    : undefined
  const memoryCapacityGi = nodeDetail.capacity?.memory
    ? convertKResourceToResource('memory', nodeDetail.capacity.memory)
    : undefined
  const memoryPercentage =
    memoryUsedGi !== undefined && memoryCapacityGi !== undefined && memoryCapacityGi > 0
      ? (memoryUsedGi / memoryCapacityGi) * 100
      : 0

  // 磁盘容量
  const diskCapacityGi = nodeDetail.capacity?.['ephemeral-storage']
    ? convertKResourceToResource('memory', nodeDetail.capacity['ephemeral-storage'])
    : undefined

  // GPU ID 到作业名称列表的映射
  const relateJobsMap: Record<string, string[]> = gpuDetail?.relateJobs || {}

  const CHART_COLORS = {
    used: 'var(--primary)',
    free: 'var(--muted)',
  }

  return (
    <div className="@container/node-info w-full min-w-0">
      <div className="grid min-w-0 grid-cols-1 gap-4 @min-[56rem]/node-info:grid-cols-2">
        <Card className="min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <ActivityIcon className="size-4 shrink-0" />
              {t('nodeDetail.info.basicInfo')}
            </CardTitle>
          </CardHeader>
          <CardContent className="min-w-0 space-y-4">
            <div className="space-y-3">
              <div className="space-y-1.5">
                <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 text-xs">
                  <div className="text-muted-foreground flex items-center gap-2">
                    <MemoryStick className="size-3.5 shrink-0" />
                    <span>{t('nodeDetail.info.memoryCapacity')}</span>
                  </div>
                  <div className="flex min-w-0 flex-wrap items-center gap-2">
                    <span className="font-mono font-medium [overflow-wrap:anywhere]">
                      {memoryUsedGi !== undefined && memoryCapacityGi !== undefined
                        ? `${memoryUsedGi.toFixed(2)} / ${memoryCapacityGi.toFixed(2)} GB`
                        : memoryCapacityGi !== undefined
                          ? `- / ${memoryCapacityGi.toFixed(2)} GB`
                          : '-'}
                    </span>
                    {memoryPercentage > 0 && (
                      <span className="text-muted-foreground font-mono text-[10px]">
                        ({memoryPercentage.toFixed(1)}%)
                      </span>
                    )}
                  </div>
                </div>
                <Progress value={memoryPercentage} className="h-1.5" />
              </div>

              <Separator className="mt-6" />

              <dl className="mt-6 grid grid-cols-[repeat(auto-fit,minmax(min(100%,12rem),1fr))] gap-4">
                <NodeInfoField
                  icon={HardDriveIcon}
                  label={t('nodeDetail.info.diskCapacity')}
                  value={diskCapacityGi !== undefined ? `${diskCapacityGi.toFixed(2)} GB` : '-'}
                />
                <NodeInfoField
                  icon={ServerIcon}
                  label={t('nodeDetail.info.kernelVersion')}
                  value={nodeDetail.kernelVersion || '-'}
                />
              </dl>
            </div>
          </CardContent>
        </Card>

        <Card className="@container/cpu min-w-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <CpuIcon className="size-4 shrink-0" />
              {t('nodeDetail.info.cpuInfo')}
            </CardTitle>
          </CardHeader>
          <CardContent className="min-w-0">
            <div className="flex min-w-0 flex-col items-center gap-6 @min-[28rem]/cpu:flex-row">
              <div className="flex shrink-0 flex-col items-center gap-2">
                <div className="relative h-[100px] w-[100px]">
                  <ResponsiveContainer width="100%" height="100%">
                    <PieChart>
                      <Pie
                        data={cpuChartData}
                        cx="50%"
                        cy="50%"
                        innerRadius={30}
                        outerRadius={45}
                        startAngle={90}
                        endAngle={-270}
                        dataKey="value"
                        stroke="none"
                        isAnimationActive={isDataLoaded}
                        animationBegin={0}
                        animationDuration={800}
                      >
                        <Cell fill={CHART_COLORS.used} />
                        <Cell fill={CHART_COLORS.free} />
                        <Label
                          content={({ viewBox }) => {
                            if (viewBox && 'cx' in viewBox && 'cy' in viewBox) {
                              return (
                                <text
                                  x={viewBox.cx}
                                  y={viewBox.cy}
                                  textAnchor="middle"
                                  dominantBaseline="middle"
                                >
                                  <tspan
                                    x={viewBox.cx}
                                    y={viewBox.cy}
                                    className="fill-foreground text-lg font-bold"
                                  >
                                    {cpuPercentage}%
                                  </tspan>
                                </text>
                              )
                            }
                          }}
                        />
                      </Pie>
                    </PieChart>
                  </ResponsiveContainer>
                </div>
                {cpuUsedValue !== undefined && cpuCapacityValue !== undefined && (
                  <div className="text-muted-foreground font-mono text-[10px]">
                    {cpuUsedValue.toFixed(2)} / {cpuCapacityValue.toFixed(2)} Cores
                  </div>
                )}
              </div>
              <div className="flex w-full min-w-0 flex-1 flex-col justify-center gap-4">
                <div className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,7rem),1fr))] gap-4">
                  <div className="min-w-0">
                    <div className="text-muted-foreground mb-1 text-[10px] tracking-wider uppercase">
                      {t('nodeDetail.info.architecture')}
                    </div>
                    <div className="text-lg font-bold [overflow-wrap:anywhere]">
                      {nodeDetail.arch.toUpperCase()}
                    </div>
                  </div>
                  <div className="min-w-0">
                    <div className="text-muted-foreground mb-1 text-[10px] tracking-wider uppercase">
                      {t('nodeDetail.info.cpuCount')}
                    </div>
                    <div className="flex flex-wrap items-baseline gap-1">
                      <span className="text-lg font-bold [overflow-wrap:anywhere]">
                        {cpuCapacity}
                      </span>
                      <span className="text-muted-foreground text-xs">Cores</span>
                    </div>
                  </div>
                </div>
                <dl className="min-w-0 border-t pt-4">
                  <NodeInfoField
                    icon={CpuIcon}
                    label={t('nodeDetail.info.cpuModel')}
                    value={nodeDetail.cpuModel || '-'}
                  />
                </dl>
              </div>
            </div>
          </CardContent>
        </Card>

        {gpuDetail?.haveGPU && (
          <div className="min-w-0 space-y-4 @min-[56rem]/node-info:col-span-2">
            {gpuDetail.gpuDevices?.map((device, deviceIdx) => {
              let gpuStartIdx = 0
              for (let i = 0; i < deviceIdx; i++) {
                gpuStartIdx += gpuDetail.gpuDevices?.[i]?.count || 0
              }
              const gpuEndIdx = gpuStartIdx + device.count

              let deviceUsedCount = 0
              for (let i = gpuStartIdx; i < gpuEndIdx; i++) {
                const gpuId = i.toString()
                const jobs = relateJobsMap[gpuId] || []
                const isAllocated = jobs.length > 0
                if (isAllocated) {
                  deviceUsedCount++
                }
              }
              const deviceUnusedCount = device.count - deviceUsedCount

              return (
                <Card key={deviceIdx} className="@container/gpu min-w-0">
                  <CardHeader>
                    <CardTitle className="flex items-start gap-2 text-base">
                      <Zap className="size-4 shrink-0" />
                      <span className="min-w-0 [overflow-wrap:anywhere]">
                        {t('nodeDetail.info.acceleratorInfo')} - {device.label.toUpperCase()}
                      </span>
                    </CardTitle>
                  </CardHeader>
                  <CardContent className="min-w-0">
                    <div className="grid min-w-0 grid-cols-1 gap-6 @min-[56rem]/gpu:grid-cols-2">
                      <div className="min-w-0 space-y-6">
                        <div>
                          <div className="bg-muted text-foreground flex min-w-0 flex-wrap items-center gap-3 rounded-md px-4 py-3">
                            <AcceleratorBadge acceleratorString={device.resourceName} />
                            <span className="min-w-0 text-base font-semibold tracking-tight [overflow-wrap:anywhere]">
                              {device.product?.toUpperCase()}
                            </span>
                          </div>
                        </div>

                        <dl className="grid grid-cols-[repeat(auto-fit,minmax(min(100%,12rem),1fr))] gap-4">
                          {[
                            {
                              icon: Cable,
                              label: t('nodeDetail.gpu.driverVersion'),
                              value: device.driver || '-',
                            },
                            {
                              icon: ActivityIcon,
                              label: device.vendorDomain?.includes('nvidia')
                                ? `CUDA ${t('nodeDetail.info.version')}`
                                : device.vendorDomain?.includes('amd')
                                  ? `ROCm ${t('nodeDetail.info.version')}`
                                  : t('nodeDetail.info.version'),
                              value: device.runtimeVersion || '-',
                            },
                            {
                              icon: MemoryStick,
                              label: t('nodeDetail.gpu.memory'),
                              value: device.memory ? `${parseInt(device.memory) / 1024} GB` : '-',
                            },
                            {
                              icon: CpuIcon,
                              label: t('nodeDetail.gpu.architecture'),
                              value: device.arch || '-',
                            },
                          ].map(({ icon: Icon, label, value }, idx) => (
                            <NodeInfoField key={idx} icon={Icon} label={label} value={value} />
                          ))}
                        </dl>
                      </div>

                      <div className="min-w-0 border-t pt-6 @min-[56rem]/gpu:border-t-0 @min-[56rem]/gpu:border-l @min-[56rem]/gpu:pt-0 @min-[56rem]/gpu:pl-6">
                        <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                          <div className="text-muted-foreground text-xs font-bold tracking-wider uppercase">
                            {t('nodeDetail.gpu.allocationStatus')}
                          </div>
                          <div className="flex flex-wrap gap-2">
                            <Badge
                              variant="outline"
                              className="border-primary/20 bg-primary/10 text-primary h-5 px-2 text-[10px]"
                            >
                              {t('nodeDetail.gpu.allocatedCount', { count: deviceUsedCount })}
                            </Badge>
                            <Badge
                              variant="outline"
                              className="border-border bg-muted text-muted-foreground h-5 px-2 text-[10px]"
                            >
                              {t('nodeDetail.gpu.unallocatedCount', {
                                count: deviceUnusedCount,
                              })}
                            </Badge>
                          </div>
                        </div>

                        <div className="grid grid-cols-[repeat(auto-fill,minmax(min(100%,5rem),1fr))] gap-3">
                          {Array.from({ length: device.count }).map((_, i) => {
                            const gpuId = (gpuStartIdx + i).toString()
                            const jobs = relateJobsMap[gpuId] || []
                            const isAllocated = jobs.length > 0
                            const displayJobs = jobs.filter((job) => !job.startsWith('__'))
                            const hasDisplayJobs = displayJobs.length > 0

                            const gpuCard = (
                              <div
                                key={gpuId}
                                className={cn(
                                  'group bg-card relative flex flex-col items-center justify-center gap-2 rounded-lg border p-3 shadow-sm transition-all hover:shadow-md',
                                  isAllocated && 'border-primary/30'
                                )}
                              >
                                <div
                                  className={`rounded-full p-2 ${isAllocated ? 'bg-primary/10' : 'bg-muted'}`}
                                >
                                  <Zap
                                    className={`size-4 ${isAllocated ? 'text-primary fill-primary' : 'text-muted-foreground'}`}
                                  />
                                </div>
                                <div className="text-muted-foreground font-mono text-xs font-medium">
                                  GPU-{gpuId}
                                </div>
                              </div>
                            )

                            if (hasDisplayJobs) {
                              return (
                                <Tooltip key={gpuId}>
                                  <TooltipTrigger asChild>{gpuCard}</TooltipTrigger>
                                  <TooltipContent side="top" className="max-w-xs">
                                    <div className="space-y-0.5">
                                      {displayJobs.map((job, idx) => (
                                        <div key={idx} className="text-xs">
                                          {job}
                                        </div>
                                      ))}
                                    </div>
                                  </TooltipContent>
                                </Tooltip>
                              )
                            }

                            return gpuCard
                          })}
                        </div>
                      </div>
                    </div>
                  </CardContent>
                </Card>
              )
            })}
          </div>
        )}
      </div>
    </div>
  )
}

export default NodeInfoTab
