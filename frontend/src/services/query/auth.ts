import { queryOptions } from '@tanstack/react-query'

import { apiGetAuthMode } from '../api/auth'

export const queryAuthMode = queryOptions({
  queryKey: ['authmode'],
  queryFn: apiGetAuthMode,
  select: (res) => res.data,
})
