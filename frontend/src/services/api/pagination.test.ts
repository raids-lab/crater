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

import { fetchAllPages } from './pagination.ts'

const response = <T>(items: T[], total: number) => ({
  code: 0,
  msg: '',
  data: { items, total, page: 1, page_size: 2 },
})

test('fetchAllPages returns an empty collection when total is zero', async () => {
  const result = await fetchAllPages(async () => response([], 0))
  assert.deepEqual(result.data, [])
})

test('fetchAllPages aggregates complete pages and a short final page', async () => {
  const pages = [response([1, 2], 5), response([3, 4], 5), response([5], 5)]
  const result = await fetchAllPages(async (page) => pages[page - 1])
  assert.deepEqual(result.data, [1, 2, 3, 4, 5])
})

test('fetchAllPages rejects an empty page while total is not satisfied', async () => {
  let requests = 0
  await assert.rejects(
    fetchAllPages(async (page) => {
      requests += 1
      return page === 1 ? response([1, 2], 5) : response([], 5)
    }),
    /empty page 2 after collecting 2 of 5 reported items/
  )
  assert.equal(requests, 2)
})

test('fetchAllPages follows an increased total during pagination', async () => {
  const result = await fetchAllPages(async (page) => {
    if (page === 1) return response([1, 2], 4)
    if (page === 2) return response([3, 4], 5)
    return response([5], 5)
  })
  assert.deepEqual(result.data, [1, 2, 3, 4, 5])
})

test('fetchAllPages surfaces page request failures without returning partial data', async () => {
  const failure = new Error('network failure')
  await assert.rejects(
    fetchAllPages(async (page) => {
      if (page === 1) return response([1, 2], 5)
      throw failure
    }),
    failure
  )
})

test('fetchAllPages stops after the maximum page request count', async () => {
  let requests = 0
  await assert.rejects(
    fetchAllPages(async (page) => {
      requests += 1
      return response([page], page + 1)
    }),
    /exceeded the maximum of 10000 page requests/
  )
  assert.equal(requests, 10000)
})
