import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

import { createSessionClient } from '../src/services/session-client.ts'
import { openTerminalWebSocket } from '../src/services/terminal-session.ts'

const json = (status, data) =>
  new Response(JSON.stringify(data), { status, headers: { 'Content-Type': 'application/json' } })
const checked = () => json(200, { data: { user: { name: 'alice' } } })
const expired = () => json(401, { code: 40101 })
const renewed = () =>
  json(200, { data: { accessToken: 'new-access', refreshToken: 'new-refresh' } })

function deferred() {
  let resolve
  const promise = new Promise((done) => {
    resolve = done
  })
  return { promise, resolve }
}

function harness(fetch) {
  let access = 'access+&value',
    refresh = 'refresh-must-not-be-sent',
    invalidations = 0
  const sockets = []
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
      access = refresh = null
    },
    fetch: async (request) => {
      const body = request.body ? await request.text() : undefined
      return fetch(new Request(request, body === undefined ? {} : { body }))
    },
  })
  const createSocket = (url) => {
    const socket = {
      url,
      closed: false,
      close() {
        this.closed = true
      },
    }
    sockets.push(socket)
    return socket
  }
  return {
    client,
    createSocket,
    sockets,
    options: {
      client,
      createSocket,
      origin: 'https://example.invalid',
      apiBase: '',
      namespace: 'workspace',
      podName: 'job-0',
      containerName: 'main',
      signal: new AbortController().signal,
    },
    switchSession(a = 'other-access', r = 'other-refresh') {
      access = a
      refresh = r
    },
    get access() {
      return access
    },
    get invalidations() {
      return invalidations
    },
  }
}

test('terminal URL carries only the validated access token, including encoded characters', async () => {
  const h = harness(async (request) => {
    assert.equal(request.headers.get('Authorization'), 'Bearer access+&value')
    assert.equal(new URL(request.url).pathname, '/api/auth/check')
    return checked()
  })
  for (const origin of ['http://localhost:5183', 'https://example.invalid']) {
    const socket = await openTerminalWebSocket({ ...h.options, origin, apiBase: '/crater' })
    const url = new URL(socket.url)
    assert.equal(url.protocol, origin.startsWith('https:') ? 'wss:' : 'ws:')
    assert.equal(
      url.pathname,
      '/crater/api/v1/websocket/namespaces/workspace/pods/job-0/containers/main/terminal'
    )
    assert.equal(url.searchParams.get('token'), 'access+&value')
    assert.equal(url.toString().includes('refresh-must-not-be-sent'), false)
  }
})

test('terminals and HTTP requests share one renewal before opening sockets', async () => {
  const gate = deferred()
  let checks = 0,
    renewals = 0
  const h = harness(async (request) => {
    if (request.url.endsWith('/auth/refresh')) {
      renewals++
      assert.deepEqual(await request.json(), { refreshToken: 'refresh-must-not-be-sent' })
      await gate.promise
      return renewed()
    }
    if (request.headers.get('Authorization') === 'Bearer new-access') return checked()
    checks++
    return expired()
  })
  const pending = [
    openTerminalWebSocket(h.options),
    openTerminalWebSocket(h.options),
    h.client.get('auth/check'),
  ]
  for (let turn = 0; checks < 3 || renewals < 1; turn++) {
    assert.ok(turn < 1000, `checks=${checks}, renewals=${renewals}`)
    await new Promise(setImmediate)
  }
  assert.equal(h.sockets.length, 0)
  gate.resolve()
  await Promise.all(pending)
  assert.equal(renewals, 1)
  assert.equal(h.sockets.length, 2)
  for (const socket of h.sockets)
    assert.equal(new URL(socket.url).searchParams.get('token'), 'new-access')
})

for (const failure of ['expired refresh', 'network failure', 'empty check']) {
  test(`terminal never opens after ${failure}`, async () => {
    const h = harness(async (request) => {
      if (failure === 'network failure') throw new TypeError('offline')
      if (failure === 'empty check') return json(200, { data: null })
      return request.url.endsWith('/auth/refresh') ? json(401, { code: 40101 }) : expired()
    })
    await assert.rejects(openTerminalWebSocket(h.options))
    assert.equal(h.sockets.length, 0)
    assert.equal(h.invalidations, failure === 'expired refresh' ? 1 : 0)
  })
}

for (const change of ['switch', 'logout', 'access-only switch', 'close']) {
  test(`a pending terminal cannot open after ${change}`, async () => {
    const gate = deferred(),
      started = deferred(),
      abort = new AbortController()
    const h = harness(async () => {
      started.resolve()
      await gate.promise
      return checked()
    })
    const pending = openTerminalWebSocket({ ...h.options, signal: abort.signal })
    await started.promise
    if (change === 'close') abort.abort()
    else if (change === 'logout') h.switchSession(null, null)
    else
      h.switchSession(
        'other-access',
        change === 'access-only switch' ? 'refresh-must-not-be-sent' : 'other-refresh'
      )
    gate.resolve()
    await assert.rejects(pending)
    assert.equal(h.sockets.length, 0)
    assert.equal(h.invalidations, 0)
  })
}

function mountTerminal(h) {
  let cleanup,
    ref = 0
  const element = { querySelector: () => ({ classList: { add() {} } }) }
  const modules = {
    '@xterm/addon-fit': {
      FitAddon: class {
        fit() {}
      },
    },
    '@xterm/xterm': {
      Terminal: class {
        loadAddon() {}
        open() {}
        focus() {}
        onResize() {}
        onData() {}
        dispose() {}
        writeln() {}
        write() {}
      },
    },
    '@xterm/xterm/css/xterm.css': {},
    jotai: { useAtomValue: () => '' },
    react: {
      useRef: (current) => ({ current: ref++ === 1 ? element : current }),
      useEffect: (effect) => {
        cleanup = effect()
      },
    },
    'react/jsx-runtime': { jsx: (_, props) => props },
    'usehooks-ts': { useDebounceCallback: (fn) => Object.assign(fn, { cancel() {} }) },
    '@/components/ui/card': {},
    '@/services/client': { apiClient: h.client },
    '@/services/auth-errors': { getAuthErrorMessage: () => 'authentication failed' },
    '@/services/terminal-session': {
      openTerminalWebSocket: (options) =>
        openTerminalWebSocket({ ...options, createSocket: h.createSocket }),
    },
    '@/utils/store/config': {},
    './pod-container-dialog': {},
  }
  const source = readFileSync(
    new URL('../src/components/codeblock/terminal-dialog.tsx', import.meta.url),
    'utf8'
  )
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
  })
  const exports = {}
  runInNewContext(outputText, {
    exports,
    AbortController,
    require: (name) => {
      assert.ok(name in modules, `Unexpected dependency: ${name}`)
      return modules[name]
    },
    window: {
      location: { origin: 'https://example.invalid' },
      addEventListener() {},
      removeEventListener() {},
    },
  })
  const props = exports.default({})
  props.ActionComponent({
    namespacedName: { namespace: 'workspace', name: 'job-0' },
    selectedContainer: { name: 'main', state: { running: {} } },
  })
  return () => cleanup()
}

test('actual terminal component opens with access credentials and closes its socket on unmount', async () => {
  const h = harness(async () => checked())
  const unmount = mountTerminal(h)
  for (let turn = 0; !h.sockets.length; turn++) {
    assert.ok(turn < 1000, 'terminal did not open its socket')
    await new Promise(setImmediate)
  }
  assert.equal(new URL(h.sockets[0].url).searchParams.get('token'), 'access+&value')
  unmount()
  assert.equal(h.sockets[0].closed, true)
})

test('closing the actual terminal component cancels its pending authentication check', async () => {
  const gate = deferred(),
    started = deferred()
  const h = harness(async () => {
    started.resolve()
    await gate.promise
    return checked()
  })
  const unmount = mountTerminal(h)
  await started.promise
  unmount()
  gate.resolve()
  await new Promise(setImmediate)
  assert.equal(h.sockets.length, 0)
})
