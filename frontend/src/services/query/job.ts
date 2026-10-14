import { queryOptions } from '@tanstack/react-query'

import {
  apiAIJobGetDetail,
  apiJobGetDetail,
  apiJupyterTokenGet,
  apiWebIDETokenGet,
} from '@/services/api/vcjob'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const queryJupyterToken = (name: string) =>
  queryOptions({
    queryKey: ['ingress', 'jupyter', name],
    queryFn: () => apiJupyterTokenGet(name),
    select: (data) => data.data,
    enabled: !!name,
  })

export const queryWebIDEToken = (name: string) =>
  queryOptions({
    queryKey: ['ingress', 'webide', name],
    queryFn: () => apiWebIDETokenGet(name),
    select: (data) => data.data,
    enabled: !!name,
  })

export const queryJobDetail = (jobName: string, id?: number) =>
  queryOptions({
    queryKey: ['job', 'detail', id ?? jobName],
    queryFn: () => (id === undefined ? apiJobGetDetail(jobName) : apiAIJobGetDetail(id)),
    select: (res) => res.data,
    refetchInterval: REFETCH_INTERVAL,
  })
