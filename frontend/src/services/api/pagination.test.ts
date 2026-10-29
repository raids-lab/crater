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

const response = <T>(items: T[], total: number, page = 1, pageSize = 2) => ({
  code: 0,
  msg: '',
  data: { items, total, page, page_size: pageSize },
})

test('fetchAllPages returns an empty collection when total is zero', async () => {
  const result = await fetchAllPages(async () => response([], 0))
  assert.deepEqual(result.data, [])
})

test('fetchAllPages aggregates complete pages and a short final page', async () => {
  const pages = [response([1, 2], 5), response([3, 4], 5, 2), response([5], 5, 3)]
  const result = await fetchAllPages(async (page) => pages[page - 1])
  assert.deepEqual(result.data, [1, 2, 3, 4, 5])
})

test('fetchAllPages rejects an empty page while total is not satisfied', async () => {
  let requests = 0
  await assert.rejects(
    fetchAllPages(async (page) => {
      requests += 1
      return page === 1 ? response([1, 2], 5) : response([], 5, page)
    }),
    /empty page 2 after collecting 2 of 5 reported items/
  )
  assert.equal(requests, 2)
})

test('fetchAllPages follows an increased total during pagination', async () => {
  const result = await fetchAllPages(async (page) => {
    if (page === 1) return response([1, 2], 4)
    if (page === 2) return response([3, 4], 5, 2)
    return response([5], 5, 3)
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
      return response([page], page + 1, page, 1)
    }),
    /exceeded the maximum of 10000 page requests/
  )
  assert.equal(requests, 10000)
})

const firstPage = response([1, 2], 4)
const invalidPages = [
  {
    name: 'missing page',
    pages: [
      {
        ...response([], 0),
        data: { ...response([], 0).data, page: undefined as unknown as number },
      },
    ],
    error: /received page/,
  },
  { name: 'zero page', pages: [response([], 0, 0)], error: /received page 0/ },
  { name: 'negative page', pages: [response([], 0, -1)], error: /received page -1/ },
  { name: 'fractional page', pages: [response([], 0, 1.5)], error: /received page 1.5/ },
  { name: 'skipped first page', pages: [response([], 0, 2)], error: /received page 2/ },
  {
    name: 'repeated page',
    pages: [firstPage, firstPage],
    error: /requested page 2, received page 1/,
  },
  {
    name: 'skipped page',
    pages: [firstPage, response([3, 4], 4, 3)],
    error: /requested page 2, received page 3/,
  },
  ...[0, -1, 1.5, Number.NaN, Number.POSITIVE_INFINITY, Number.MAX_SAFE_INTEGER + 1].map(
    (size) => ({
      name: `invalid page size ${size}`,
      pages: [response([], 0, 1, size)],
      error: /page_size/,
    })
  ),
  {
    name: 'changing page size',
    pages: [firstPage, response([3], 3, 2, 1)],
    error: /page_size/,
  },
  ...[-1, 0.5, Number.NaN, Number.POSITIVE_INFINITY, Number.MAX_SAFE_INTEGER + 1].map((total) => ({
    name: `invalid total ${total}`,
    pages: [response([], total)],
    error: /invalid total/,
  })),
  { name: 'oversized page', pages: [response([1, 2, 3], 3)], error: /item count/ },
  { name: 'null items', pages: [response(null as unknown as number[], 0)], error: /item count/ },
  { name: 'nonempty zero total', pages: [response([1], 0)], error: /exceeding reported total/ },
  { name: 'short intermediate page', pages: [response([1], 3)], error: /short page 1/ },
  {
    name: 'total below collected count',
    pages: [firstPage, response([3, 4], 3, 2)],
    error: /exceeding reported total/,
  },
  {
    name: 'empty page total below collected count',
    pages: [firstPage, response([], 1, 2)],
    error: /exceeding reported total/,
  },
]

for (const { name, pages, error } of invalidPages) {
  test(`fetchAllPages rejects ${name} without requesting more pages`, async () => {
    let requests = 0
    await assert.rejects(
      fetchAllPages(async () => {
        assert.ok(requests < pages.length, 'requested a page after invalid response')
        return pages[requests++]
      }),
      error
    )
    assert.equal(requests, pages.length)
  })
}

test('fetchAllPages accepts a decreased total that matches a short final page', async () => {
  const result = await fetchAllPages(async (page) =>
    page === 1 ? response([1, 2], 5) : response([3], 3, page)
  )
  assert.deepEqual(result.data, [1, 2, 3])
})

test('fetchAllPages accepts an empty final page when the total decreases to the collected count', async () => {
  const result = await fetchAllPages(async (page) =>
    page === 1 ? response([1, 2], 5) : response([], 2, page)
  )
  assert.deepEqual(result.data, [1, 2])
})

test('fetchAllPages accepts equal generic values on distinct pages', async () => {
  const result = await fetchAllPages(async (page) => response([1], 2, page, 1))
  assert.deepEqual(result.data, [1, 1])
})

test('fetchAllPages can complete on the last allowed request', async () => {
  let requests = 0
  const result = await fetchAllPages(async (page) => {
    requests += 1
    return response([page], 10000, page, 1)
  })
  assert.equal(requests, 10000)
  assert.equal(result.data.length, 10000)
})
