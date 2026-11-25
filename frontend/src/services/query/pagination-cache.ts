import type { QueryClient, QueryKey } from '@tanstack/react-query'

function invalidateLists(queryClient: QueryClient, ...keys: QueryKey[]) {
  return Promise.all(keys.map((queryKey) => queryClient.invalidateQueries({ queryKey })))
}

export function invalidateImageLists(queryClient: QueryClient) {
  return invalidateLists(
    queryClient,
    ['remote-list', 'imagelink-user'],
    ['remote-list', 'imagelink-admin'],
    ['images'],
    ['allImageTags']
  )
}

export function invalidateImageBuildLists(queryClient: QueryClient) {
  return invalidateLists(
    queryClient,
    ['remote-list', 'imagepack-user'],
    ['remote-list', 'imagepack-admin'],
    ['imagepack', 'quota']
  )
}

export function invalidateModelDownloadLists(queryClient: QueryClient) {
  return invalidateLists(
    queryClient,
    ['remote-list', 'model-downloads'],
    ['model-downloads-summary']
  )
}

export function invalidateApprovalOrderLists(queryClient: QueryClient) {
  return invalidateLists(
    queryClient,
    ['remote-list', 'approval-orders-admin'],
    ['remote-list', 'approval-orders-me'],
    ['approval-order-summary'],
    ['approvalorders', 'byName']
  )
}

export function invalidateDatasetLists(queryClient: QueryClient, type: string) {
  return invalidateLists(queryClient, ['data', type], ['remote-list', 'admin-datasets'])
}
