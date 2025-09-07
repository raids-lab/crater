import { queryOptions } from '@tanstack/react-query'

import { apiJobGetDetail, apiJupyterTokenGet } from '@/services/api/vcjob'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const queryJupyterToken = (name: string) =>
  queryOptions({
    queryKey: ['ingress', 'jupyter', name],
    queryFn: () => apiJupyterTokenGet(name),
    select: ({ data }) => {
      return {
        ...data,
        urlWithToken: data.token ? `${data.fullURL}?token=${data.token}` : undefined,
      }
    },
    enabled: !!name,
  })

export const queryJobDetail = (jobName: string) =>
  queryOptions({
    queryKey: ['job', 'detail', jobName],
    queryFn: () => apiJobGetDetail(jobName),
    select: (res) => res.data,
    refetchInterval: REFETCH_INTERVAL,
  })
