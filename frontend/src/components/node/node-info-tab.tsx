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
import { ActivityIcon, Cable, CpuIcon, HardDriveIcon, ServerIcon, Zap } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Cell, Label, Pie, PieChart, ResponsiveContainer } from 'recharts'

import { Badge } from '@/components/ui/badge'
import { Card, CardContent } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { Separator } from '@/components/ui/separator'

import { IClusterNodeDetail, IClusterNodeGPU } from '@/services/api/cluster'

import { convertKResourceToResource } from '@/utils/resource'

import { cn } from '@/lib/utils'

type NodeInfoTabProps = {
  nodeDetail?: IClusterNodeDetail
  gpuDetail?: IClusterNodeGPU
}

export const NodeInfoTab = ({ nodeDetail, gpuDetail }: NodeInfoTabProps) => {
  const { t } = useTranslation()

  if (!nodeDetail) return null

  // CPU 资源计算
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
  const cpuChartData = [
    { name: 'Used', value: cpuUsedValue || 0 },
    { name: 'Free', value: Math.max(0, (cpuCapacityValue || 0) - (cpuUsedValue || 0)) },
  ]

  // 内存资源计算
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

  // GPU 分配情况计算
  const gpuUtil = gpuDetail?.gpuUtil || {}
  const gpuTotalCount = gpuDetail?.gpuCount || 0
  const relateJobs = gpuDetail?.relateJobs
  const relateJobsMap: Record<string, string[]> = Array.isArray(relateJobs)
    ? {}
    : (relateJobs as unknown as Record<string, string[]>) || {}

  // 统计已使用的GPU数量
  const gpuUtilMap = new Map<string, boolean>()
  Object.entries(gpuUtil).forEach(([gpuId, util]) => {
    if (util > 0) gpuUtilMap.set(gpuId, true)
  })
  Object.entries(relateJobsMap).forEach(([gpuId, jobs]) => {
    if (jobs?.length > 0) {
      const hasRealJob = jobs.some((job) => job !== '__k8s_allocated__')
      const hasK8sAllocation = jobs.includes('__k8s_allocated__')
      if (hasRealJob || hasK8sAllocation) gpuUtilMap.set(gpuId, true)
    }
  })
  const gpuUsedCount = gpuUtilMap.size

  const CHART_COLORS = {
    used: 'var(--primary)',
    free: 'var(--muted)',
  }

  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <Card>
        <CardContent className="py-2.4 space-y-4 px-6">
          <h3 className="mb-4 flex items-center gap-2 text-base font-semibold">
            <ActivityIcon className="size-4" />
            {t('nodeDetail.info.basicInfo')}
          </h3>
          <div className="space-y-3">
            <div className="space-y-1.5">
              <div className="flex items-center justify-between text-xs">
                <div className="text-muted-foreground flex items-center gap-2">
                  <ServerIcon className="size-3.5" />
                  <span>{t('nodeDetail.info.memoryCapacity')}</span>
                </div>
                <span className="font-mono font-medium">
                  {memoryUsedGi !== undefined && memoryCapacityGi !== undefined
                    ? `${memoryUsedGi.toFixed(2)} / ${memoryCapacityGi.toFixed(2)} GB`
                    : memoryCapacityGi !== undefined
                      ? `- / ${memoryCapacityGi.toFixed(2)} GB`
                      : '-'}
                </span>
              </div>
              <Progress value={memoryPercentage} className="h-1.5" />
            </div>

            <Separator className="mt-6" />

            <div className="mt-6 grid grid-cols-2 gap-x-8">
              <div className="col-span-1">
                <div className="flex items-center">
                  <div className="text-muted-foreground flex w-32 shrink-0 items-center gap-2 text-sm">
                    <HardDriveIcon className="size-4" />
                    <span>{t('nodeDetail.info.diskCapacity')}</span>
                  </div>
                  <Badge variant="secondary" className="ml-1 font-mono text-[10px]">
                    {diskCapacityGi !== undefined ? `${diskCapacityGi.toFixed(2)} GB` : '-'}
                  </Badge>
                </div>
              </div>
              <div className="col-span-1">
                <div className="flex items-center">
                  <div className="text-muted-foreground flex w-32 shrink-0 items-center gap-2 text-sm">
                    <ServerIcon className="size-4" />
                    <span>{t('nodeDetail.info.kernelVersion')}</span>
                  </div>
                  <Badge variant="secondary" className="ml-1 font-mono text-[10px]">
                    {nodeDetail.kernelVersion || '-'}
                  </Badge>
                </div>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="py-2.4 flex flex-col px-6">
          <h3 className="mb-6 flex items-center gap-2 text-base font-semibold">
            <CpuIcon className="size-4" />
            {t('nodeDetail.info.cpuInfo')}
          </h3>
          <div className="flex items-center justify-between">
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
            <div className="flex flex-1 flex-col justify-center gap-4 pl-6">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-muted-foreground mb-1 text-[10px] tracking-wider uppercase">
                    {t('nodeDetail.info.architecture')}
                  </div>
                  <div className="text-lg font-bold">{nodeDetail.arch.toUpperCase()}</div>
                </div>
                <div>
                  <div className="text-muted-foreground mb-1 text-[10px] tracking-wider uppercase">
                    {t('nodeDetail.info.cpuCount')}
                  </div>
                  <div className="flex items-baseline gap-1">
                    <span className="text-lg font-bold">{cpuCapacity}</span>
                    <span className="text-muted-foreground text-xs">Cores</span>
                  </div>
                </div>
              </div>
              <div className="flex items-center border-t pt-4">
                <div className="text-muted-foreground flex w-32 shrink-0 items-center gap-2 text-sm">
                  <CpuIcon className="size-4" />
                  <span>{t('nodeDetail.info.cpuModel')}</span>
                </div>
                <Badge variant="secondary" className="ml-1 font-mono text-[10px]">
                  {'-'}
                </Badge>
              </div>
            </div>
          </div>
        </CardContent>
      </Card>

      {gpuDetail?.haveGPU && (
        <Card className="md:col-span-2">
          <CardContent className="py-2.4 px-6">
            <h3 className="mb-4 flex items-center gap-2 text-base font-semibold">
              <Zap className="size-4" />
              {t('nodeDetail.info.acceleratorInfo')}
            </h3>

            <div className="flex flex-col gap-8 lg:flex-row">
              <div className="flex-1 space-y-6">
                <div>
                  <div className="flex items-center gap-3 rounded-md bg-slate-950 px-4 py-3 text-white shadow-sm dark:bg-slate-800 dark:text-slate-100">
                    <div className="flex size-8 items-center justify-center rounded bg-white/10 dark:bg-white/20">
                      <Zap className="size-4 fill-yellow-400 text-yellow-400" />
                    </div>
                    <span className="text-lg font-semibold tracking-tight">
                      {gpuDetail.gpuProduct}
                    </span>
                  </div>
                </div>

                <div className="grid grid-cols-2 gap-x-8 gap-y-4">
                  {[
                    {
                      icon: Cable,
                      label: t('nodeDetail.gpu.driverVersion'),
                      value: gpuDetail.gpuDriver,
                    },
                    {
                      icon: ActivityIcon,
                      label: `CUDA ${t('nodeDetail.info.version')}`,
                      value: gpuDetail.cudaVersion,
                    },
                    {
                      icon: ServerIcon,
                      label: t('nodeDetail.gpu.memory'),
                      value: `${parseInt(gpuDetail.gpuMemory) / 1024} GB`,
                    },
                    {
                      icon: CpuIcon,
                      label: t('nodeDetail.gpu.architecture'),
                      value: gpuDetail.gpuArch,
                    },
                  ].map(({ icon: Icon, label, value }, idx) => (
                    <div key={idx} className="col-span-1">
                      <div className="flex items-center">
                        <div className="text-muted-foreground flex w-32 shrink-0 items-center gap-2 text-sm">
                          <Icon className="size-4" />
                          <span>{label}</span>
                        </div>
                        <Badge variant="secondary" className="ml-1 font-mono text-[10px]">
                          {value}
                        </Badge>
                      </div>
                    </div>
                  ))}
                </div>
              </div>

              <div className="flex-1 border-t pt-6 lg:border-t-0 lg:border-l lg:pt-0 lg:pl-8">
                <div className="mb-4 flex items-center justify-between">
                  <div className="text-muted-foreground text-xs font-bold tracking-wider uppercase">
                    {t('nodeDetail.gpu.allocationStatus')}
                  </div>
                  <div className="flex gap-2">
                    <Badge
                      variant="outline"
                      className="border-primary/20 bg-primary/10 text-primary h-5 px-2 text-[10px]"
                    >
                      {t('nodeDetail.gpu.allocatedCount', { count: gpuUsedCount })}
                    </Badge>
                    <Badge
                      variant="outline"
                      className="border-border bg-muted text-muted-foreground h-5 px-2 text-[10px]"
                    >
                      {t('nodeDetail.gpu.unallocatedCount', {
                        count: gpuTotalCount - gpuUsedCount,
                      })}
                    </Badge>
                  </div>
                </div>

                <div
                  className={cn(
                    'gap-3',
                    gpuTotalCount < 4
                      ? 'flex flex-wrap'
                      : 'grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4'
                  )}
                >
                  {Array.from({ length: gpuTotalCount }).map((_, i) => {
                    const gpuId = i.toString()
                    const jobs = relateJobsMap[gpuId] || []
                    const isAllocated =
                      jobs.some((job) => job !== '__k8s_allocated__') ||
                      jobs.includes('__k8s_allocated__')

                    return (
                      <div
                        key={gpuId}
                        className={cn(
                          'group bg-card relative flex flex-col items-center justify-between gap-3 rounded-lg border p-3 shadow-sm transition-all hover:shadow-md',
                          gpuTotalCount < 4 && 'w-[calc((100%-0.75rem*3)/4)]'
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
                        <div
                          className={`h-1 w-8 rounded-full transition-colors ${isAllocated ? 'bg-primary' : 'bg-muted-foreground/50'}`}
                        />
                      </div>
                    )
                  })}
                </div>
              </div>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

export default NodeInfoTab
