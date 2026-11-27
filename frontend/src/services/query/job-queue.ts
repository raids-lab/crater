import { queryOptions } from '@tanstack/react-query'

import { apiJobQueue } from '@/services/api/job-queue'

import { REFETCH_INTERVAL } from '@/lib/constants'

export const queryJobQueue = (
  userId: number | undefined,
  account: string | undefined,
  admin = false,
  mine = false
) =>
  queryOptions({
    queryKey: ['job-queue', admin, userId, admin ? null : account, mine],
    queryFn: ({ signal }) => apiJobQueue(admin, mine, signal).then((r) => r.data),
    enabled: userId !== undefined && (admin || !!account),
    refetchInterval: REFETCH_INTERVAL,
  })
