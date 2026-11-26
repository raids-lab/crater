import ky, { HTTPError } from 'ky'
import assert from 'node:assert/strict'
import { EventEmitter, once } from 'node:events'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import ts from 'typescript'

const accessKey = 'fixture-access'
const refreshKey = 'fixture-refresh'
const expired = 40101
const tokenKeys = { ACCESS_TOKEN_KEY: accessKey, REFRESH_TOKEN_KEY: refreshKey }
const portal = 'https://crater-fixture.invalid'

function loadModule(path, imports, globals) {
  const source = readFileSync(new URL(`../src/${path}`, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  const module = { exports: {} }
  new Function(
    'require',
    'module',
    'exports',
    'localStorage',
    'sessionStorage',
    'window',
    'globalThis',
    outputText
  )(
    (name) => {
      if (!(name in imports)) throw new Error('Unexpected test dependency: ' + name)
      return imports[name]
    },
    module,
    module.exports,
    globals.localStorage,
    globals.sessionStorage,
    globals.window,
    { fetch: globals.fetch }
  )
  return module.exports
}

function createFixture() {
  const stored = new Map()
  const events = new EventEmitter()
  const requests = []
  const window = {
    location: {
      href: portal + '/portal',
      pathname: '/portal',
      assign(path) {
        this.href = path
        this.pathname = path
      },
    },
  }
  const localStorage = {
    getItem: (key) => stored.get(key) ?? null,
    setItem: (key, value) => stored.set(key, value),
    removeItem: (key) => stored.delete(key),
  }
  const fetch = async (request) =>
    new Promise((resolve) => {
      // Ky still creates requests and invokes the production hooks; only network I/O is replaced.
      const { origin, pathname } = new URL(request.url)
      assert.equal(origin, portal)
      // Consume the request body as fetch would, allowing Ky to cancel the unused clone.
      const body = request.text()
      requests.push({
        pathname,
        method: request.method,
        authorization: request.headers.get('Authorization'),
        readBody: async () => JSON.parse(await body),
        respond: (data, status = 200) => resolve(Response.json(data, { status })),
      })
      events.emit('request')
    })

  const sessionStorage = { setItem() {} }
  const globals = { localStorage, sessionStorage, window, fetch }
  const session = loadModule('services/auth-session.ts', { '@/utils/store': tokenKeys }, globals)
  const fixtureKy = (input, options) => ky(input, { fetch, ...options })
  fixtureKy.create = (options) => ky.create({ fetch, ...options })
  const codes = loadModule('services/error_code.ts', {}, globals)
  const sessionClient = loadModule(
    'services/session-client.ts',
    { ky: { default: fixtureKy, HTTPError }, './error_code.ts': codes },
    globals
  )
  const client = loadModule(
    'services/client.ts',
    {
      jotai: { getDefaultStore: () => ({ get: () => portal }) },
      ky: { default: fixtureKy, HTTPError },
      sonner: { toast: { error() {} } },
      '@/utils/loglevel': { logger: { error() {} } },
      '@/utils/store': tokenKeys,
      '@/utils/store/config': { configAPIBaseAtom: {} },
      '@/utils/toast': { showErrorToast() {} },
      './auth-errors': { getAuthErrorMessage: () => 'Authentication failed' },
      './auth-session': session,
      './session-client': sessionClient,
      './error_code': codes,
      './error_code_legacy': {},
    },
    globals
  )
  const auth = loadModule(
    'services/api/auth.ts',
    {
      '@/services/client': client,
      '@/services/auth-session': session,
      '../error_code': { OK: 0 },
    },
    globals
  )

  session.replaceAuthSession('A-access', 'A-refresh')
  return {
    client: client.apiClient,
    auth,
    session,
    localStorage,
    window,
    requests,
    async next(pathname) {
      for (;;) {
        const request = requests.find((entry) => entry.pathname === pathname && !entry.taken)
        if (request) {
          request.taken = true
          return request
        }
        await once(events, 'request')
      }
    },
    async respond(pathname, data, status) {
      const request = await this.next(pathname)
      request.respond(data, status)
      return request
    },
  }
}

const expiredResponse = { code: expired }
const tokens = (scope) => ({ accessToken: scope + '-access', refreshToken: scope + '-refresh' })
const switchResponse = (queue) => ({ code: 0, data: { ...tokens(queue), context: { queue } } })
const settled = (request) => request.catch((error) => error)
const verify = (description, fn) => test(description, { timeout: 5000 }, fn)

verify('late expiry of a write without an AbortSignal cannot use the new account', async () => {
  const f = createFixture()
  const original = settled(f.client.post('fixture-write'))
  const sent = await f.next('/api/fixture-write')
  assert.equal(sent.authorization, 'Bearer A-access')

  const applied = []
  const switching = f.auth.apiQueueSwitch('B', (context) => applied.push(context.queue))
  await f.respond('/api/v1/auth/switch', switchResponse('B'))
  await switching
  sent.respond(expiredResponse, 401)

  assert.equal((await original).name, 'AbortError')
  assert.deepEqual(applied, ['B'])
  assert.equal(f.requests.length, 2)
  assert.equal(f.localStorage.getItem(accessKey), 'B-access')
  assert.equal(f.window.location.href, portal + '/portal')
})

verify(
  'two concurrent expired requests share one refresh and replay with the rotated token',
  async () => {
    const f = createFixture()
    const results = [f.client.get('one').json(), f.client.get('two').json()]
    await f.respond('/api/one', expiredResponse, 401)
    await f.respond('/api/two', expiredResponse, 401)
    const refresh = await f.next('/api/auth/refresh')
    assert.deepEqual(await refresh.readBody(), { refreshToken: 'A-refresh' })
    refresh.respond({ data: tokens('A-renewed') })

    for (const path of ['/api/one', '/api/two']) {
      const replay = await f.respond(path, { ok: true })
      assert.equal(replay.authorization, 'Bearer A-renewed-access')
    }
    assert.deepEqual(await Promise.all(results), [{ ok: true }, { ok: true }])
    assert.equal(f.requests.filter((r) => r.pathname === '/api/auth/refresh').length, 1)
    assert.equal(f.localStorage.getItem(refreshKey), 'A-renewed-refresh')
  }
)

verify('a late expiry in the same session reuses an already rotated token', async () => {
  const f = createFixture()
  const first = f.client.get('first')
  const late = f.client.get('late')
  const lateRequest = await f.next('/api/late')
  const session = f.session.getAuthSession()
  assert.deepEqual(Object.keys(session), [])
  await f.respond('/api/first', expiredResponse, 401)
  await f.respond('/api/auth/refresh', { data: tokens('A-renewed') })
  await f.respond('/api/first', { ok: true })
  await first

  lateRequest.respond(expiredResponse, 401)
  const replay = await f.respond('/api/late', { ok: true })
  assert.equal(replay.authorization, 'Bearer A-renewed-access')
  await late
  assert.equal(f.session.getAuthSession(), session)
  assert.equal(f.requests.filter((r) => r.pathname === '/api/auth/refresh').length, 1)
})

verify('refreshes are isolated and a late A refresh cannot overwrite B', async () => {
  const f = createFixture()
  const old = settled(f.client.get('A-list'))
  const waiter = settled(f.client.get('A-other'))
  await f.respond('/api/A-list', expiredResponse, 401)
  const aRefresh = await f.next('/api/auth/refresh')
  await f.respond('/api/A-other', expiredResponse, 401)

  f.session.replaceAuthSession('B-access', 'B-refresh')
  const current = f.client.get('B-list')
  await f.respond('/api/B-list', expiredResponse, 401)
  const bRefresh = await f.next('/api/auth/refresh')
  assert.deepEqual(await bRefresh.readBody(), { refreshToken: 'B-refresh' })
  bRefresh.respond({ data: tokens('B-renewed') })
  const bReplay = await f.respond('/api/B-list', { ok: true })
  assert.equal(bReplay.authorization, 'Bearer B-renewed-access')
  await current

  aRefresh.respond({ data: tokens('A-renewed') })
  assert.equal((await old).name, 'AbortError')
  assert.equal((await waiter).name, 'AbortError')
  assert.equal(f.localStorage.getItem(accessKey), 'B-renewed-access')
  assert.equal(f.requests.filter((r) => r.pathname.startsWith('/api/A-')).length, 2)
  assert.equal(f.window.location.href, portal + '/portal')
})

for (const stale of [false, true]) {
  verify(
    stale ? 'stale refresh failure cannot redirect B' : 'active refresh failure redirects to login',
    async () => {
      const f = createFixture()
      const result = settled(f.client.get('protected'))
      await f.respond('/api/protected', expiredResponse, 401)
      const refresh = await f.next('/api/auth/refresh')
      if (stale) f.session.replaceAuthSession('B-access', 'B-refresh')
      refresh.respond({ code: 40100 }, 401)
      const error = await result
      if (stale) {
        assert.equal(error.name, 'AbortError')
        assert.equal(f.window.location.href, portal + '/portal')
      } else {
        assert.equal(error.response.status, 401)
        assert.equal(f.window.location.href, '/auth')
      }
    }
  )
}

verify('a replay that also returns expiry is not replayed again', async () => {
  const f = createFixture()
  const result = settled(f.client.delete('write-once'))
  await f.respond('/api/write-once', expiredResponse, 401)
  await f.respond('/api/auth/refresh', { data: tokens('A-renewed') })
  const replay = await f.respond('/api/write-once', expiredResponse, 401)
  assert.equal(replay.method, 'DELETE')
  assert.equal((await result).response.status, 401)
  assert.equal(f.requests.length, 3)
})

verify('a replay response from A cannot report an auth error into B after switching', async () => {
  const f = createFixture()
  const result = settled(f.client.delete('pending-replay'))
  await f.respond('/api/pending-replay', expiredResponse, 401)
  await f.respond('/api/auth/refresh', { data: tokens('A-renewed') })
  const replay = await f.next('/api/pending-replay')
  assert.equal(replay.authorization, 'Bearer A-renewed-access')
  f.session.replaceAuthSession('B-access', 'B-refresh')
  replay.respond(expiredResponse, 401)
  assert.equal((await result).name, 'AbortError')
  assert.equal(f.localStorage.getItem(accessKey), 'B-access')
  assert.equal(f.window.location.href, portal + '/portal')
})

for (const relogin of [false, true]) {
  verify(
    relogin
      ? 'pending switch cannot overwrite a new login with identical token strings'
      : 'pending switch cannot restore a logged-out session',
    async () => {
      const f = createFixture()
      const switching = f.auth.apiQueueSwitch('B', () => assert.fail('stale switch applied'))
      const request = await f.next('/api/v1/auth/switch')
      f.session.clearAuthSession()
      if (relogin) f.session.replaceAuthSession('A-access', 'A-refresh')
      request.respond(switchResponse('B'))
      await switching
      assert.equal(f.localStorage.getItem(accessKey), relogin ? 'A-access' : null)
      assert.equal(f.localStorage.getItem(refreshKey), relogin ? 'A-refresh' : null)
    }
  )
}

for (const olderFirst of [true, false]) {
  verify(
    `only the newest switch applies when the older result returns ${olderFirst ? 'first' : 'last'}`,
    async () => {
      const f = createFixture()
      const applied = []
      const apply = (context) => applied.push(context.queue)
      const earlier = f.auth.apiQueueSwitch('B', apply)
      const earlierRequest = await f.next('/api/v1/auth/switch')
      const latest = f.auth.apiQueueSwitch('C', apply)
      const latestRequest = await f.next('/api/v1/auth/switch')
      if (olderFirst) {
        earlierRequest.respond(switchResponse('B'))
        await earlier
        assert.deepEqual(applied, [])
        assert.equal(f.localStorage.getItem(accessKey), 'A-access')
      }
      latestRequest.respond(switchResponse('C'))
      await latest
      if (!olderFirst) {
        earlierRequest.respond(switchResponse('B'))
        await earlier
      }
      assert.deepEqual(applied, ['C'])
      assert.equal(f.localStorage.getItem(accessKey), 'C-access')
    }
  )
}

verify('a failed current switch leaves credentials and context untouched', async () => {
  const f = createFixture()
  const result = assert.rejects(
    f.auth.apiQueueSwitch('B', () => assert.fail('failed switch applied')),
    /switch denied/
  )
  await f.respond('/api/v1/auth/switch', { code: 987, msg: 'switch denied' })
  await result
  assert.equal(f.localStorage.getItem(accessKey), 'A-access')
  assert.equal(f.localStorage.getItem(refreshKey), 'A-refresh')
})

verify('a default-account switch can commit after a same-session token rotation', async () => {
  const f = createFixture()
  const applied = []
  const switching = f.auth.apiQueueSwitch('default', (context) => applied.push(context.queue))
  const request = await f.next('/api/v1/auth/switch')
  assert.deepEqual(await request.readBody(), { queue: 'default' })
  const expiredRequest = f.client.get('protected')
  await f.respond('/api/protected', expiredResponse, 401)
  await f.respond('/api/auth/refresh', { data: tokens('A-renewed') })
  await f.respond('/api/protected', { ok: true })
  await expiredRequest

  request.respond(switchResponse('default'))
  await switching
  assert.deepEqual(applied, ['default'])
  assert.equal(f.localStorage.getItem(accessKey), 'default-access')
})
