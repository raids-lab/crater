/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import assert from 'node:assert/strict'
import test from 'node:test'

import type { DataListRemoteQuery } from './data-list-remote.ts'
import { clampDataListRemotePage, reduceDataListRemoteQuery } from './data-list-remote.ts'

const baseQuery: DataListRemoteQuery = {
  page: 3,
  pageSize: 10,
  search: '',
  owner: 'all',
  tag: '',
  sortField: 'createdAt',
  sortDirection: 'descending',
}

test('page change preserves the remaining query state', () => {
  const next = reduceDataListRemoteQuery(baseQuery, { type: 'page', page: 4 })
  assert.deepEqual(next, { ...baseQuery, page: 4 })
})

test('page size change resets the current page', () => {
  const next = reduceDataListRemoteQuery(baseQuery, { type: 'pageSize', pageSize: 20 })
  assert.equal(next.page, 1)
  assert.equal(next.pageSize, 20)
})

test('search change resets the current page', () => {
  const next = reduceDataListRemoteQuery(baseQuery, { type: 'search', search: '测试' })
  assert.equal(next.page, 1)
  assert.equal(next.search, '测试')
})

test('filter and sort changes reset the current page', () => {
  const owner = reduceDataListRemoteQuery(baseQuery, { type: 'owner', owner: 'mine' })
  const tag = reduceDataListRemoteQuery(baseQuery, { type: 'tag', tag: 'dataset' })
  const sortField = reduceDataListRemoteQuery(baseQuery, {
    type: 'sortField',
    sortField: 'mountCount',
  })
  const sortDirection = reduceDataListRemoteQuery(baseQuery, {
    type: 'sortDirection',
    sortDirection: 'ascending',
  })

  assert.deepEqual([owner.page, tag.page, sortField.page, sortDirection.page], [1, 1, 1, 1])
  assert.equal(owner.owner, 'mine')
  assert.equal(tag.tag, 'dataset')
  assert.equal(sortField.sortField, 'mountCount')
  assert.equal(sortDirection.sortDirection, 'ascending')
})

test('empty result moves an out-of-range query back to page one', () => {
  const next = clampDataListRemotePage(baseQuery, 0)
  assert.equal(next.page, 1)
})

test('valid page keeps the same query object', () => {
  const next = clampDataListRemotePage(baseQuery, 35)
  assert.equal(next, baseQuery)
})
