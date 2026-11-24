import { useQuery } from '@tanstack/react-query'
import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

import { apiGetUserFields } from '@/services/api/system-config'
import { UserFieldKey, UserFields, userFieldLabel } from '@/services/user-fields'

const emptyFields: UserFields = {}

export function useUserFields() {
  const { t, i18n } = useTranslation()
  const query = useQuery({
    queryKey: ['system-config', 'user-fields'],
    queryFn: () => apiGetUserFields().then((response) => response.data),
  })
  const fields: UserFields = query.data ?? emptyFields
  const label = useCallback(
    (key: UserFieldKey) =>
      userFieldLabel(fields, key, i18n.language, t(`userFields.labels.${key}`)),
    [fields, i18n.language, t]
  )
  return {
    fields,
    label,
    isPending: query.isPending,
    isError: query.isError,
    refetch: query.refetch,
  }
}
