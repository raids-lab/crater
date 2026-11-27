import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'

import { apiGetKthenaInferenceStatus } from '@/services/api/system-config'

const kthenaInferenceStatusQueryKey = ['system-config', 'kthena-inference'] as const

export const Route = createFileRoute('/portal/inference-services')({
  beforeLoad: async ({ context }) => {
    try {
      const status = await context.queryClient.ensureQueryData({
        queryKey: kthenaInferenceStatusQueryKey,
        queryFn: () => apiGetKthenaInferenceStatus().then((res) => res.data),
        retry: false,
        staleTime: 30_000,
      })

      if (status.enabled) {
        return
      }
    } catch {
      // Fail closed if the feature-status endpoint is unavailable during a rolling upgrade.
    }

    throw redirect({ to: '/portal/overview' })
  },
  component: Outlet,
})
