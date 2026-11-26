import * as ky from 'ky'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

import * as codes from '../src/services/error_code.ts'
import * as legacy from '../src/services/error_code_legacy.ts'
import { createSessionClient } from '../src/services/session-client.ts'

const json = (status, data) =>
  new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })

function transportHarness(fetch) {
  const storage = new Map([
    ['access', 'old-access'],
    ['refresh', 'old-refresh'],
  ])
  const transfers = []
  const tokenKeys = { ACCESS_TOKEN_KEY: 'access', REFRESH_TOKEN_KEY: 'refresh' }
  const localStorage = {
    getItem: (key) => storage.get(key) ?? null,
    setItem: (key, value) => storage.set(key, value),
    removeItem: (key) => storage.delete(key),
  }
  const authSession = {}
  const sessionSource = readFileSync(
    new URL('../src/services/auth-session.ts', import.meta.url),
    'utf8'
  )
  runInNewContext(
    ts.transpileModule(sessionSource, {
      compilerOptions: { module: ts.ModuleKind.CommonJS },
    }).outputText,
    { exports: authSession, require: () => tokenKeys, localStorage, DOMException }
  )
  const modules = {
    jotai: { getDefaultStore: () => ({ get: () => 'https://example.invalid' }) },
    ky,
    sonner: { toast: { error() {} } },
    '@/utils/loglevel': { logger: { error() {}, debug() {} } },
    '@/utils/store': tokenKeys,
    '@/utils/store/config': {},
    '@/utils/toast': {},
    './auth-errors': { getAuthErrorMessage: () => 'session expired' },
    './auth-session': authSession,
    './error_code': codes,
    './error_code_legacy': legacy,
    './session-client': {
      createSessionClient: (settings) =>
        createSessionClient({
          ...settings,
          fetch: async (request) => {
            const body = request.body ? await request.text() : undefined
            return fetch(new Request(request, body === undefined ? {} : { body }))
          },
        }),
    },
  }
  class XHR {
    constructor() {
      this.headers = {}
      this.events = {}
      this.upload = {
        addEventListener: (name, callback) => {
          this.events[name] = callback
        },
      }
    }
    open(method, url) {
      this.method = method
      this.url = url
    }
    setRequestHeader(key, value) {
      this.headers[key] = value
    }
    addEventListener(name, callback) {
      this.events[name] = callback
    }
    send(body) {
      transfers.push({ method: this.method, headers: this.headers, body })
      this.status = 200
      this.responseText = JSON.stringify({ code: 0 })
      this.response = new Blob(['file'])
      this.events.progress?.({ lengthComputable: true, loaded: 4, total: 4 })
      this.events.load()
    }
  }
  const source = readFileSync(new URL('../src/services/client.ts', import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS },
  })
  const exports = {}
  runInNewContext(outputText, {
    exports,
    Blob,
    URL,
    XMLHttpRequest: XHR,
    require: (name) => {
      assert.ok(name in modules, `Unexpected dependency: ${name}`)
      return modules[name]
    },
    localStorage,
    sessionStorage: { setItem() {} },
    window: { location: { pathname: '/auth' } },
    document: {
      createElement: () => ({ style: {}, click() {} }),
      body: { appendChild() {}, removeChild() {} },
    },
  })
  return { client: exports, storage, transfers }
}

for (const method of ['PUT', 'GET']) {
  test(`actual progress ${method} refreshes before sending its body once with an access token`, async () => {
    let renewals = 0
    const h = transportHarness(async (request) => {
      if (request.url.endsWith('/auth/refresh')) {
        renewals++
        assert.deepEqual(await request.json(), { refreshToken: 'old-refresh' })
        return json(200, { data: { accessToken: 'new-access', refreshToken: 'new-refresh' } })
      }
      assert.equal(new URL(request.url).pathname, '/api/auth/check')
      return request.headers.get('Authorization') === 'Bearer old-access'
        ? json(401, { code: 40101 })
        : json(200, { data: { user: {} } })
    })
    const progress = []
    const body = new Uint8Array([1, 2, 3, 4]).buffer
    if (method === 'PUT')
      await h.client.apiXMLPut('ss/files/test', body, (event) => progress.push(event))
    else await h.client.apiXMLDownload('ss/files/test', 'test.txt', (event) => progress.push(event))
    assert.equal(renewals, 1)
    assert.equal(h.transfers.length, 1)
    assert.equal(h.transfers[0].headers.Authorization, 'Bearer new-access')
    assert.equal(h.transfers[0].method, method)
    assert.equal(h.transfers[0].body, method === 'PUT' ? body : undefined)
    assert.equal(progress[0].loaded, 4)
  })

  test(`actual progress ${method} never starts under a newly switched account`, async () => {
    let release, started
    const gate = new Promise((resolve) => {
      release = resolve
    })
    const ready = new Promise((resolve) => {
      started = resolve
    })
    const h = transportHarness(async () => {
      started()
      await gate
      return json(200, { data: { user: {} } })
    })
    const pending =
      method === 'PUT'
        ? h.client.apiXMLPut('ss/files/test', new ArrayBuffer(0))
        : h.client.apiXMLDownload('ss/files/test', 'test.txt')
    await ready
    h.storage.set('access', 'other-access')
    h.storage.set('refresh', 'other-refresh')
    release()
    await assert.rejects(pending, { name: 'AbortError', message: /session changed/i })
    assert.equal(h.transfers.length, 0)
  })
}

test('streamed log responses retain streaming and use the normal renewal path', async () => {
  let calls = 0
  const h = transportHarness(async (request) => {
    calls++
    assert.ok(calls <= 3, 'stream request must not enter a renewal loop')
    if (request.url.endsWith('/auth/refresh'))
      return json(200, { data: { accessToken: 'new-access', refreshToken: 'new-refresh' } })
    assert.equal(request.headers.get('Accept'), 'application/octet-stream')
    if (request.headers.get('Authorization') === 'Bearer old-access')
      return json(401, { code: 40101 })
    return new Response(
      new ReadableStream({
        start(controller) {
          controller.enqueue(new TextEncoder().encode('log-line\n'))
          controller.close()
        },
      }),
      { headers: { 'Content-Type': 'application/octet-stream' } }
    )
  })
  const response = await h.client.apiClient.get('v1/log/stream', {
    headers: { Accept: 'application/octet-stream' },
    timeout: false,
  })
  assert.equal(response.bodyUsed, false)
  assert.equal(await response.text(), 'log-line\n')
  assert.equal(calls, 3)
})
