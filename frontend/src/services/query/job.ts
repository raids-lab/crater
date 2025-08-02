import { queryOptions } from '@tanstack/react-query'

import { apiJupyterTokenGet } from '@/services/api/vcjob'

export const queryJupyterToken = (name: string) =>
  queryOptions({
    queryKey: ['ingress', 'jupyter', name],
    queryFn: () => apiJupyterTokenGet(name ?? '0'),
    retry: 3,
    retryDelay: 2000,
    select: (res) => res.data,
    enabled: !!name,
  })
