import { queryOptions } from '@tanstack/react-query'

import { apiAdminGetDatasetByID, apiGetDatasetByID } from '../api/dataset'

export const queryDataByID = (id: number) =>
  queryOptions({
    queryKey: ['data', 'datasetByID', id],
    queryFn: () => apiGetDatasetByID(id),
    select: (res) => res.data,
  })

export const queryAdminDataByID = (id: number) =>
  queryOptions({
    queryKey: ['data', 'adminDatasetByID', id],
    queryFn: () => apiAdminGetDatasetByID(id),
    select: (res) => res.data,
  })
