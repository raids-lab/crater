import { HTTPError, TimeoutError } from 'ky'
import assert from 'node:assert/strict'
import { test } from 'node:test'

import { createSessionClient } from '../src/services/session-client.ts'

const json = (status, data) =>
  new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })
const expired = () => json(401, { code: 40101 })
function harness(handler, timeout) {
  let access = 'old-access',
    refresh = 'old-refresh',
    invalidations = 0
  const client = createSessionClient({
    prefixUrl: 'https://example.invalid/api',
    getAccessToken: () => access,
    getRefreshToken: () => refresh,
    saveTokens: (a, r) => {
      access = a
      refresh = r
    },
    onInvalid: () => {
      invalidations++
      access = null
      refresh = null
    },
    fetch: async (request) => {
      // A real fetch consumes upload streams, including the refresh POST body.
      const body = request.body ? await request.text() : undefined
      return handler(new Request(request, body === undefined ? {} : { body }))
    },
    timeout,
  })
  return {
    client,
    get access() {
      return access
    },
    get invalidations() {
      return invalidations
    },
    logout() {
      access = null
      refresh = null
    },
  }
}
const renewal = () =>
  json(200, { data: { accessToken: 'new-access', refreshToken: 'new-refresh' } })

test('concurrent expired requests share one renewal and replay POST bodies once', async () => {
  let renewals = 0,
    initial = 0,
    replays = 0
  let release
  const gate = new Promise((resolve) => {
    release = resolve
  })
  const h = harness(async (request) => {
    if (request.url.endsWith('/auth/refresh')) {
      renewals++
      await gate
      return renewal()
    }
    assert.deepEqual(await request.json(), { value: 42 })
    if (request.headers.get('Authorization') === 'Bearer old-access') {
      initial++
      return expired()
    }
    assert.equal(request.headers.get('Authorization'), 'Bearer new-access')
    replays++
    return json(200, { code: 0 })
  })
  const requests = [
    h.client.post('v1/action', { json: { value: 42 } }).json(),
    h.client.post('v1/action', { json: { value: 42 } }).json(),
  ]
  while (initial < 2 || renewals < 1) await new Promise(setImmediate)
  release()
  await Promise.all(requests)
  assert.equal(renewals, 1)
  assert.equal(replays, 2)
  assert.equal(h.invalidations, 0)
})

test('refresh transport and service failures keep the session and settle all waiters', async () => {
  for (const kind of ['network', 'service', 'invalid JSON']) {
    const h = harness(async (request) => {
      if (!request.url.endsWith('/auth/refresh')) return expired()
      if (kind === 'network') throw new TypeError('offline')
      if (kind === 'invalid JSON') return new Response('<html>', { status: 502 })
      return json(500, { code: 50002 })
    })
    await assert.rejects(h.client.get('auth/check'))
    assert.equal(h.access, 'old-access')
    assert.equal(h.invalidations, 0)
  }
})

test('terminal refresh rejection invalidates the session', async () => {
  for (const [status, code] of [
    [401, 40102],
    [401, 40109],
    [403, 40304],
    [403, 40303],
  ]) {
    const h = harness(async (request) =>
      request.url.endsWith('/auth/refresh') ? json(status, { code }) : expired()
    )
    await assert.rejects(h.client.get('auth/check'), HTTPError)
    assert.equal(h.invalidations, 1)
    assert.equal(h.access, null)
  }
})

test('a second expired response terminates after one replay', async () => {
  let calls = 0
  const h = harness(async (request) => {
    calls++
    return request.url.endsWith('/auth/refresh') ? renewal() : expired()
  })
  await assert.rejects(h.client.get('auth/check'), HTTPError)
  assert.equal(calls, 3)
  assert.equal(h.invalidations, 1)
})

test('public authentication errors never trigger renewal or clear a stored session', async () => {
  let calls = 0
  const h = harness(async () => {
    calls++
    return expired()
  })
  await assert.rejects(h.client.post('auth/login', { json: {} }), HTTPError)
  assert.equal(calls, 1)
  assert.equal(h.invalidations, 0)
})

test('logout during renewal prevents session resurrection', async () => {
  let release, started
  const gate = new Promise((resolve) => {
    release = resolve
  })
  const ready = new Promise((resolve) => {
    started = resolve
  })
  const h = harness(async (request) => {
    if (request.url.endsWith('/auth/refresh')) {
      started()
      await gate
      return renewal()
    }
    return expired()
  })
  const result = h.client.get('auth/check')
  await ready
  h.logout()
  release()
  await assert.rejects(result)
  assert.equal(h.access, null)
})

test('refresh timeout ends the request without clearing the session', async () => {
  const h = harness(async (request) => {
    if (request.url.endsWith('/auth/refresh')) return new Promise(() => {})
    return expired()
  }, 25)
  await assert.rejects(h.client.get('auth/check'), TimeoutError)
  assert.equal(h.invalidations, 0)
})

test('replayed service failure preserves diagnostic body and performs no extra retries', async () => {
  let calls = 0
  const h = harness(async (request) => {
    calls++
    if (request.url.endsWith('/auth/refresh')) return renewal()
    return request.headers.get('Authorization') === 'Bearer old-access'
      ? expired()
      : json(500, { code: 50002, msg: 'safe database error' })
  })
  const error = await h.client.get('auth/check').catch((error) => error)
  assert(error instanceof HTTPError)
  assert.equal((await error.response.json()).code, 50002)
  assert.equal(calls, 3)
  assert.equal(h.invalidations, 0)
})

test('a restriction on job submission does not revoke platform login', async () => {
  const h = harness(async () => json(403, { code: 40303 }))
  await assert.rejects(h.client.post('v1/jobs', { json: {} }), HTTPError)
  assert.equal(h.invalidations, 0)
  assert.equal(h.access, 'old-access')
})
