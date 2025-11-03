import { queryOptions } from '@tanstack/react-query'

import { apiAccountGet } from '../api/account'

export const queryAccountByID = (id: number) =>
  queryOptions({
    queryKey: ['admin', 'accounts', id],
    queryFn: () => apiAccountGet(id),
    select: (res) => res.data,
    enabled: !!id,
  })
