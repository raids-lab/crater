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
// Modified code
import { useTranslation } from 'react-i18next'
import { Card, CardDescription, CardFooter, CardHeader } from '@/components/ui/card'
import { Progress } from '@/components/ui/progress'
import { ResourceResp } from '@/services/api/context'
import { useQuery } from '@tanstack/react-query'
import { CpuIcon, MemoryStickIcon, GpuIcon } from 'lucide-react'
import { REFETCH_INTERVAL } from '@/config/task'
import { useMemo } from 'react'
import { cn } from '@/lib/utils'
import { apiAccountQuotaGet } from '@/services/api/account'

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
  const { t } = useTranslation()
  const allocated = resource?.allocated?.amount ?? 0
  const quota = resource?.deserved?.amount ?? resource?.capability?.amount ?? 1
  const [progress, overflow] = useMemo(() => {
    const progress = (allocated / quota) * 100
    const overflow = progress > 100
    return [overflow ? 100 : progress, overflow]
  }, [allocated, quota])
  return (
    <Card className="flex flex-col items-stretch justify-between">
      <CardHeader className="flex flex-row items-start justify-between space-y-0 pb-3">
        <CardDescription className="flex flex-row items-center justify-start uppercase">
          <div className="bg-primary/10 mr-1.5 flex h-7 w-7 items-center justify-center rounded-full">
            <props.icon className="text-primary size-4" />
          </div>
          {resource?.label}
        </CardDescription>
        <p
          className={cn('text-muted-foreground mx-0.5 font-mono text-xl font-bold', {
            'text-orange-500': overflow,
          })}
        >
          {t('quotaCard.used')}
          <span
            className={cn('text-primary mx-0.5 font-mono text-xl font-bold', {
              'text-orange-500': overflow,
            })}
          >
            {showAmount(allocated, resource?.label)}
          </span>
          {showUnit(resource?.label)}
        </p>
      </CardHeader>
      <CardFooter className="flex flex-col gap-2">
        <Progress
          value={progress}
          aria-label={resource?.label}
          className={cn({ 'bg-orange-500/20 *:bg-orange-400': overflow })}
        />
        <div className="flex w-full flex-row-reverse items-center justify-between text-xs">
          {resource?.capability?.amount !== undefined && (
            <p className="text-orange-500">
              {t('quotaCard.limit')}
              {showAmount(resource?.capability?.amount, resource?.label)}
            </p>
          )}
          {resource?.deserved?.amount !== undefined && (
            <p className="text-teal-500">
              {t('quotaCard.entitled')}
              {showAmount(resource?.deserved?.amount, resource?.label)}
            </p>
          )}
          {resource?.allocated?.amount !== undefined && (
            <p className="text-sky-500">
              {t('quotaCard.usedLabel')}
              {showAmount(resource?.allocated?.amount, resource?.label)}
            </p>
          )}
          {resource?.guarantee?.amount !== undefined && resource?.guarantee?.amount > 0 && (
            <p className="text-slate-500">
              {t('quotaCard.guaranteed')}
              {showAmount(resource?.guarantee?.amount, resource?.label)}
            </p>
          )}
        </div>
      </CardFooter>
    </Card>
  )
}

const Quota = ({ accountID }: { accountID: number }) => {
  const { data: quota } = useQuery({
    queryKey: ['admin', 'accounts', accountID, 'quota'],
    queryFn: () => apiAccountQuotaGet(accountID),
    select: (res) => res.data.data,
    enabled: !!accountID,
    refetchInterval: REFETCH_INTERVAL,
  })

  return (
    <>
      <QuotaCard resource={quota?.cpu} icon={CpuIcon} />
      <QuotaCard resource={quota?.memory} icon={MemoryStickIcon} />
      {quota?.gpus?.map((gpu, i) => <QuotaCard key={i} resource={gpu} icon={GpuIcon} />)}
    </>
  )
}

export default Quota
