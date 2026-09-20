import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import {
  IKthenaInferenceStatus,
  apiAdminGetKthenaInferenceStatus,
  apiAdminSetKthenaInferenceStatus,
} from '@/services/api/system-config'

import { showErrorToast } from '@/utils/toast'

const adminQueryKey = ['admin', 'system-config', 'kthena-inference'] as const
const publicQueryKey = ['system-config', 'kthena-inference'] as const

export function useKthenaInferenceSettings() {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const statusQuery = useQuery({
    queryKey: adminQueryKey,
    queryFn: () => apiAdminGetKthenaInferenceStatus().then((res) => res.data),
  })

  const updateStatus = useMutation({
    mutationFn: apiAdminSetKthenaInferenceStatus,
    onMutate: () => queryClient.cancelQueries({ queryKey: adminQueryKey }),
    onSuccess: async (_data, enabled) => {
      // The PUT response contains a message, not a status object. Cache the
      // confirmed value before refreshing both admin and portal consumers.
      queryClient.setQueryData<IKthenaInferenceStatus>(adminQueryKey, { enabled })
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: adminQueryKey }),
        queryClient.invalidateQueries({ queryKey: publicQueryKey }),
      ])
      toast.success(
        enabled
          ? t('systemConfig.kthenaInference.enabledSuccess')
          : t('systemConfig.kthenaInference.disabledSuccess')
      )
    },
    onError: showErrorToast,
  })

  return {
    enabled: statusQuery.data?.enabled,
    isLoading: statusQuery.isPending,
    isError: statusQuery.isError,
    isFetching: statusQuery.isFetching,
    isPending: updateStatus.isPending,
    refetch: statusQuery.refetch,
    onToggle: (enabled: boolean) => {
      if (
        statusQuery.isSuccess &&
        !updateStatus.isPending &&
        enabled !== statusQuery.data.enabled
      ) {
        updateStatus.mutate(enabled)
      }
    },
  }
}
