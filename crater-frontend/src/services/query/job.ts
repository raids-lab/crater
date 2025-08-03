import { REFETCH_INTERVAL } from '@/config/task'
import { queryOptions } from '@tanstack/react-query'

import { apiJobGetDetail, apiJupyterTokenGet } from '@/services/api/vcjob'

export const queryJupyterToken = (name: string) =>
  queryOptions({
    queryKey: ['ingress', 'jupyter', name],
    queryFn: () => apiJupyterTokenGet(name ?? '0'),
    retry: 3,
    retryDelay: 2000,
    select: (res) => res.data,
    enabled: !!name,
  })

export const queryJobDetail = (jobName: string) =>
  queryOptions({
    queryKey: ['job', 'detail', jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data,
    refetchInterval: REFETCH_INTERVAL,
  })
