import { QueryClient, QueryObserver, keepPreviousData } from '@tanstack/react-query'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { setImmediate } from 'node:timers/promises'

import {
  accountListQueryKey,
  accountListQueryOptions,
  assertAccountListScope,
  getAccountListScope,
} from '../src/services/query/account-list-scope.ts'
import { invalidateDatasetLists } from '../src/services/query/pagination-cache.ts'

const context = { queue: 'q-101', roleQueue: 2, rolePlatform: 2, accessQueue: 3, accessPublic: 2 }
const accountA = getAccountListScope(25, context)
const accountB = getAccountListScope(25, { ...context, queue: 'q-102' })
const pageA = { items: [{ id: 101, name: 'A_DATA' }], total: 17 }
const pageB = { items: [{ id: 202, name: 'B_DATA' }], total: 22 }
const firstPage = { page: 1, pageSize: 10, search: '', owner: 'all' }

function deferred() {
  let resolve
  let reject
  const promise = new Promise((resolvePromise, rejectPromise) => {
    resolve = resolvePromise
    reject = rejectPromise
  })
  return { promise, resolve, reject }
}

function datasetKey(type, scope, params = firstPage) {
  return accountListQueryKey(['data', type, 'page'], scope, params)
}

function mountList(client, queryKey, isCurrent, load) {
  // AccountDataView and InteractiveJobs create a new observer when their React key changes.
  const options = {
    ...accountListQueryOptions(queryKey, isCurrent, load),
    placeholderData: keepPreviousData,
    staleTime: Infinity,
  }
  const observer = new QueryObserver(client, options)
  return { observer, unmount: observer.subscribe(() => {}), options }
}

function assertPendingWithoutRows(observer) {
  const result = observer.getCurrentResult()
  assert.equal(result.isPending, true)
  assert.equal(result.isPlaceholderData, false)
  assert.equal(result.data, undefined)
}

test('account query keys separate users, queues, permissions, types, search and page size', () => {
  assert.equal(getAccountListScope(undefined, context), undefined)
  assert.equal(getAccountListScope(25, undefined), undefined)
  assert.equal(getAccountListScope(25, { ...context, queue: '' }), undefined)
  assert.equal(getAccountListScope(25, { ...context, space: 'other' }), accountA)
  assert.deepEqual(JSON.parse(accountA), [25, 'q-101', 2, 2, 3, 2])
  const key = datasetKey('dataset', accountA)
  for (const other of [
    datasetKey('dataset', accountB),
    datasetKey('dataset', getAccountListScope(26, context)),
    datasetKey('dataset', getAccountListScope(25, { ...context, accessQueue: 2 })),
    datasetKey('sharefile', accountA),
    datasetKey('dataset', accountA, { ...firstPage, search: 'name' }),
    datasetKey('dataset', accountA, { ...firstPage, owner: 'mine' }),
    datasetKey('dataset', accountA, { ...firstPage, page: 2 }),
    datasetKey('dataset', accountA, { ...firstPage, pageSize: 20 }),
    datasetKey('dataset', accountA, { ...firstPage, sortField: 'mountCount' }),
  ]) {
    assert.notDeepEqual(key, other)
  }
})

for (const type of ['dataset', 'sharefile']) {
  test(`${type} A → B hides old rows, total and page count while B is pending`, async (t) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    t.after(() => client.clear())
    let active = accountA
    client.setQueryData(datasetKey(type, accountA), pageA)
    const a = mountList(client, datasetKey(type, accountA), () => active === accountA)
    assert.deepEqual(a.observer.getCurrentResult().data, pageA)
    assert.equal(Math.ceil(a.observer.getCurrentResult().data.total / 10), 2)

    active = accountB
    a.unmount()
    const response = deferred()
    const b = mountList(
      client,
      datasetKey(type, accountB),
      () => active === accountB,
      () => response.promise
    )
    t.after(b.unmount)
    assertPendingWithoutRows(b.observer)
    response.resolve(pageB)
    await setImmediate()
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
    assert.equal(Math.ceil(b.observer.getCurrentResult().data.total / 10), 3)
    assert.deepEqual(client.getQueryData(datasetKey(type, accountA)), pageA)
  })

  test(`${type} B failure has no old result, and retry uses B`, async (t) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    t.after(() => client.clear())
    let active = accountA
    client.setQueryData(datasetKey(type, accountA), pageA)
    const a = mountList(client, datasetKey(type, accountA), () => active === accountA)
    active = accountB
    a.unmount()
    const response = deferred()
    const failure = new Error('HTTP 500')
    let calls = 0
    const b = mountList(
      client,
      datasetKey(type, accountB),
      () => active === accountB,
      () => (++calls === 1 ? response.promise : Promise.resolve(pageB))
    )
    t.after(b.unmount)
    response.reject(failure)
    await setImmediate()
    assert.equal(b.observer.getCurrentResult().isError, true)
    assert.equal(b.observer.getCurrentResult().error, failure)
    assert.equal(b.observer.getCurrentResult().data, undefined)
    await b.observer.refetch()
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
  })

  test(`${type} late A response is cancelled and cannot change the successful B result`, async (t) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    t.after(() => client.clear())
    let active = accountA
    const old = deferred()
    let aSignal
    const a = mountList(
      client,
      datasetKey(type, accountA),
      () => active === accountA,
      (signal) => {
        aSignal = signal
        return old.promise
      }
    )
    active = accountB
    a.unmount()
    const b = mountList(
      client,
      datasetKey(type, accountB),
      () => active === accountB,
      async () => pageB
    )
    t.after(b.unmount)
    await setImmediate()
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
    old.resolve(pageA)
    await setImmediate()
    assert.equal(aSignal.aborted, true)
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
    assert.equal(client.getQueryData(datasetKey(type, accountA)), undefined)
  })

  test(`${type} A → B → A cancels B and can re-use only the A cache`, async (t) => {
    const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
    t.after(() => client.clear())
    let active = accountA
    client.setQueryData(datasetKey(type, accountA), pageA)
    const a = mountList(client, datasetKey(type, accountA), () => active === accountA)
    active = accountB
    a.unmount()
    const response = deferred()
    let bSignal
    const b = mountList(
      client,
      datasetKey(type, accountB),
      () => active === accountB,
      (signal) => {
        bSignal = signal
        return response.promise
      }
    )
    assertPendingWithoutRows(b.observer)
    active = accountA
    b.unmount()
    const back = mountList(client, datasetKey(type, accountA), () => active === accountA)
    t.after(back.unmount)
    response.resolve(pageB)
    await setImmediate()
    assert.equal(bSignal.aborted, true)
    assert.deepEqual(back.observer.getCurrentResult().data, pageA)
    assert.equal(client.getQueryData(datasetKey(type, accountB)), undefined)
  })
}

test('late interactive job list and facet responses cannot overwrite active B', async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  t.after(() => client.clear())
  let active = accountA
  for (const namespace of ['remote-list', 'remote-list-facets']) {
    const prefix = [namespace, 'jobs-interactive']
    const oldPage = deferred()
    let oldSignal
    active = accountA
    const a = mountList(
      client,
      accountListQueryKey(prefix, accountA),
      () => active === accountA,
      (signal) => {
        oldSignal = signal
        return oldPage.promise
      }
    )
    active = accountB
    a.unmount()
    const b = mountList(
      client,
      accountListQueryKey(prefix, accountB),
      () => active === accountB,
      async () => pageB
    )
    await setImmediate()
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
    oldPage.resolve(pageA)
    await setImmediate()
    assert.equal(oldSignal.aborted, true)
    assert.deepEqual(b.observer.getCurrentResult().data, pageB)
    assert.equal(client.getQueryData(accountListQueryKey(prefix, accountA)), undefined)
    b.unmount()
  }
})

test('normal same-account paging still shows inert placeholder data before the new page', async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  t.after(() => client.clear())
  client.setQueryData(datasetKey('dataset', accountA), pageA)
  const { observer, unmount, options } = mountList(
    client,
    datasetKey('dataset', accountA),
    () => true
  )
  t.after(unmount)
  const secondPage = deferred()
  observer.setOptions({
    ...options,
    queryKey: datasetKey('dataset', accountA, { ...firstPage, page: 2 }),
    queryFn: () => secondPage.promise,
  })
  assert.equal(observer.getCurrentResult().isPlaceholderData, true)
  assert.deepEqual(observer.getCurrentResult().data, pageA)
  secondPage.resolve({ ...pageA, items: [{ id: 102 }] })
  await setImmediate()
  assert.equal(observer.getCurrentResult().isPlaceholderData, false)
  assert.equal(observer.getCurrentResult().data.items[0].id, 102)
})

test('dataset mutations stale both applicable scopes but only fetch the active account', async (t) => {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  t.after(() => client.clear())
  const oldKey = datasetKey('dataset', accountA)
  const activeKey = datasetKey('dataset', accountB)
  client.setQueryData(oldKey, pageA)
  client.setQueryData(activeKey, pageB)
  let fetches = 0
  const b = mountList(
    client,
    activeKey,
    () => true,
    async () => {
      fetches++
      return { ...pageB, total: 23 }
    }
  )
  t.after(b.unmount)
  await invalidateDatasetLists(client, 'dataset')
  assert.equal(fetches, 1)
  assert.equal(client.getQueryState(oldKey).isInvalidated, true)
  assert.deepEqual(client.getQueryData(oldKey), pageA)
  assert.equal(b.observer.getCurrentResult().data.total, 23)
})

test('scope guard rejects an old query even before observer teardown and after an unabortable response', async () => {
  let active = accountA
  const currentA = () => active === accountA
  const response = deferred()
  const { signal } = new AbortController()
  const queryKey = datasetKey('dataset', accountA)
  const oldRequest = accountListQueryOptions(queryKey, currentA, () => response.promise).queryFn({
    signal,
  })
  active = accountB
  const staleObserver = accountListQueryOptions(queryKey, currentA, () => {
    assert.fail('old observer must not issue an API request')
  })
  await assert.rejects(staleObserver.queryFn({ signal }), { name: 'AbortError' })
  response.resolve(pageA)
  await assert.rejects(oldRequest, { name: 'AbortError' })
})

test('account remount discards selected jobs and an old action cannot submit under the new scope', () => {
  const jobs = readFileSync(
    new URL('../src/routes/portal/jobs/inter/index.tsx', import.meta.url),
    'utf8'
  )
  const table = readFileSync(
    new URL('../src/components/query-table/remote.tsx', import.meta.url),
    'utf8'
  )
  assert.match(jobs, /<InteractiveJobs key=\{scope\}/)
  assert.match(jobs, /assertAccountListScope\(isCurrentScope\)\s*return apiJobDelete/)
  assert.match(table, /\[rowSelection, setRowSelection\] = useState\(\{\}\)/)

  const submitted = []
  let active = accountA
  const submit = (scope, selectedIds) => () => {
    assertAccountListScope(() => active === scope)
    submitted.push(...selectedIds)
  }
  const selectedInA = submit(accountA, [101])
  active = accountB
  assert.throws(selectedInA, { name: 'AbortError' })
  assert.deepEqual(submitted, [])
  submit(accountB, [202])()
  assert.deepEqual(submitted, [202])
})

test('datasets and shared-file routes both use the keyed DataView and expose no bulk selection', () => {
  for (const [route, type] of [
    ['datasets', 'dataset'],
    ['blocks', 'sharefile'],
  ]) {
    const page = readFileSync(
      new URL(`../src/routes/portal/data/${route}/index.tsx`, import.meta.url),
      'utf8'
    )
    assert.match(page, /<DataView/)
    assert.ok(page.includes(`sourceType="${type}"`))
    assert.match(page, /apiGetDatasetPaged=\{apiGetDatasetPaged\}/)
  }
  const view = readFileSync(
    new URL('../src/components/file/data-view.tsx', import.meta.url),
    'utf8'
  )
  assert.match(view, /if \(!scope\) return null/)
  assert.match(view, /<AccountDataView\s+key=\{scope\}/)
  assert.doesNotMatch(view, /handleDelete=|rowSelection=/)
  const list = readFileSync(
    new URL('../src/components/layout/data-list.tsx', import.meta.url),
    'utf8'
  )
  assert.match(list, /inert=\{rowsInactive\}/)
})
