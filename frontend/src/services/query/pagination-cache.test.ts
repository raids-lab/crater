import { QueryClient, type QueryKey } from '@tanstack/react-query'
import assert from 'node:assert/strict'
import test from 'node:test'

import {
  invalidateApprovalOrderLists,
  invalidateDatasetLists,
  invalidateImageBuildLists,
  invalidateImageLists,
  invalidateModelDownloadLists,
} from './pagination-cache.ts'

test('mutations invalidate every matching page, filter and related count', async () => {
  const client = new QueryClient()
  const cases = [
    {
      invalidate: () => invalidateImageLists(client),
      resources: ['imagelink-user', 'imagelink-admin'],
      related: [['images', 'Jupyter'], ['allImageTags']],
    },
    {
      invalidate: () => invalidateImageBuildLists(client),
      resources: ['imagepack-user', 'imagepack-admin'],
      related: [
        ['remote-list', 'imagepack-user', 'finished-count'],
        ['imagepack', 'quota'],
      ],
    },
    {
      invalidate: () => invalidateModelDownloadLists(client),
      resources: ['model-downloads'],
      related: [['model-downloads-summary', 'dataset']],
    },
    {
      invalidate: () => invalidateApprovalOrderLists(client),
      resources: ['approval-orders-admin', 'approval-orders-me'],
      related: [['approval-order-summary'], ['approvalorders', 'byName', 'job-1']],
    },
    {
      invalidate: () => invalidateDatasetLists(client, 'sharefile'),
      resources: ['admin-datasets'],
      related: [['data', 'sharefile', 'page', { page: 2, owner: 'mine' }]],
    },
  ]
  for (const { invalidate, resources, related } of cases) {
    client.clear()
    const unrelated = ['remote-list', 'admin-users', { page: 1 }]
    const keys: QueryKey[] = resources.flatMap((resource) => [
      ['remote-list', resource, { page: 1, search: 'a' }],
      ['remote-list', resource, { page: 2, search: 'b' }],
    ])
    keys.push(...related)
    for (const key of [...keys, unrelated]) client.setQueryData(key, [])
    await invalidate()
    for (const key of keys) {
      assert.equal(client.getQueryState(key)?.isInvalidated, true, JSON.stringify(key))
    }
    assert.equal(client.getQueryState(unrelated)?.isInvalidated, false)
  }
})
