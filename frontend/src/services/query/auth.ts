import { queryOptions } from '@tanstack/react-query'

import { apiGetAuthMode } from '../api/auth'

export const queryAuthMode = queryOptions({
  queryKey: ['authmode'],
  queryFn: apiGetAuthMode,
  networkMode: 'always',
  retry: false,
  select: (res) => res.data,
})
