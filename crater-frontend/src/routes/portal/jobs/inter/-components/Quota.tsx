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
import { REFETCH_INTERVAL } from '@/config/task'
import { useQuery } from '@tanstack/react-query'
import { useAtomValue } from 'jotai'
import { CpuIcon, GpuIcon, MemoryStickIcon, NetworkIcon } from 'lucide-react'
import { useMemo } from 'react'

import { Card, CardContent, CardDescription, CardHeader } from '@/components/ui/card'

import { ProgressBar } from '@/components/ui-custom/colorful-progress'

import { ResourceResp, apiContextQuota } from '@/services/api/context'

import { atomUserContext } from '@/utils/store'

import { cn } from '@/lib/utils'

const showAmount = (allocated: number, label?: string) => {
  if (label === 'mem') {
    return (allocated / 1024 / 1024 / 1024).toFixed(0)
  }
  return allocated
}

const showUnit = (label?: string) => {
  if (label === 'mem') {
    return 'GB'
  } else if (label === 'cpu') {
    return '核'
  } else if (label?.startsWith('rdma')) {
    return '网卡'
  }
  return '卡'
}

const QuotaCard = ({
  resource,
  ...props
}: {
  icon: React.ComponentType<React.SVGProps<SVGSVGElement>>
  resource?: ResourceResp
}) => {
  const allocated = resource?.allocated?.amount ?? 0
  const deserved = resource?.deserved?.amount ?? 0
  const quota = resource?.deserved?.amount ?? resource?.capability?.amount ?? 1
  const [progress, overflow] = useMemo(() => {
    const progress = (allocated / quota) * 100
    const overflow = progress > 100
    return [overflow ? 100 : progress, overflow]
  }, [allocated, quota])
  return (
    <Card className="flex flex-col items-stretch justify-between gap-3">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-0">
        <CardDescription className="flex flex-row items-center justify-start uppercase">
          <div className="bg-primary/10 mr-1.5 flex size-7 items-center justify-center rounded-full">
            <props.icon className="text-primary size-4" />
          </div>
          <span className="font-semibold">{resource?.label}</span>
        </CardDescription>
        <div className="text-muted-foreground font-sans text-xs">
          已用
          <span
            className={cn('text-primary mx-0.5 font-mono text-2xl font-bold', {
              'text-orange-500': overflow,
            })}
          >
            {showAmount(allocated, resource?.label)}
            {deserved > 0 && (
              <span className="text-base">/{showAmount(deserved, resource?.label)}</span>
            )}
          </span>
          {showUnit(resource?.label)}
        </div>
      </CardHeader>
      <CardContent>
        <ProgressBar width={progress} aria-label={resource?.label} />
      </CardContent>
    </Card>
  )
}

const Quota = () => {
  const context = useAtomValue(atomUserContext)
  const { data: quota } = useQuery({
    queryKey: ['context', 'quota'],
    queryFn: apiContextQuota,
    select: (res) => res.data,
    enabled: context?.queue !== '',
    refetchInterval: REFETCH_INTERVAL,
  })

  return (
    <div
      className={cn('grid grid-cols-2 gap-4', {
        'md:grid-cols-3': quota?.gpus?.length === 1,
        'md:grid-cols-4': quota?.gpus?.length !== 1,
      })}
    >
      <QuotaCard resource={quota?.cpu} icon={CpuIcon} />
      <QuotaCard resource={quota?.memory} icon={MemoryStickIcon} />
      {quota?.gpus?.map((gpu, i) =>
        gpu.label.startsWith('rdma') ? (
          <QuotaCard key={i} resource={gpu} icon={NetworkIcon} />
        ) : (
          <QuotaCard key={i} resource={gpu} icon={GpuIcon} />
        )
      )}
    </div>
  )
}

export default Quota
