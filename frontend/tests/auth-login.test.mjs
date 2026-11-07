import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

import { usernameSchema } from '../src/services/auth-username.ts'

test('login and signup usernames retain the platform naming restrictions', () => {
  for (const username of ['ab', 'fengjq29', 'alice-01', 'a'.repeat(20)]) {
    assert.equal(usernameSchema.safeParse(username).success, true, username)
  }
  for (const username of [
    '',
    'a',
    'Alice',
    'alice_01',
    'alice.01',
    '1alice',
    '-alice',
    'alice-',
    'a'.repeat(21),
  ]) {
    assert.equal(usernameSchema.safeParse(username).success, false, username)
  }
})

function loginHarness(check) {
  const storage = new Map()
  const state = new Map()
  const response = {
    code: 0,
    msg: '',
    data: { accessToken: 'access', refreshToken: 'refresh', user: {}, context: {} },
  }
  const store = {
    ACCESS_TOKEN_KEY: 'access',
    REFRESH_TOKEN_KEY: 'refresh',
    atomUserInfo: 'user',
    atomUserContext: 'context',
    atomBackendVersion: 'version',
    globalLastView: 'lastView',
    useResetStore: () => ({ resetAll() {} }),
  }
  const modules = {
    '@tanstack/react-query': { useQueryClient: () => ({ clear() {} }) },
    jotai: {
      useAtom: (key) => [state.get(key), (value) => state.set(key, value)],
      useSetAtom: (key) => (value) => state.set(key, value),
    },
    react: {
      createContext: () => ({ Provider: 'Provider' }),
      useEffect() {},
      useState: () => [false, (value) => state.set('authenticated', value)],
    },
    'react/jsx-runtime': { jsx: (_, props) => props },
    'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
    sonner: { toast: { success() {}, error() {} } },
    '@/components/placeholder/loading-spinner': {},
    '@/services/api/auth': {
      apiLogin: async () => response,
      apiCheckToken: async () => {
        assert.equal(storage.get('access'), 'access')
        assert.equal(storage.get('refresh'), 'refresh')
        return check()
      },
    },
    '@/services/auth-errors': {},
    '@/utils/loglevel': {},
    '@/utils/store': store,
  }
  const source = readFileSync(new URL('../src/hooks/use-auth.tsx', import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
  })
  const exports = {}
  runInNewContext(outputText, {
    exports,
    require: (name) => {
      assert.ok(name in modules, `Unexpected dependency: ${name}`)
      return modules[name]
    },
    localStorage: {
      setItem: (key, value) => storage.set(key, value),
      removeItem: (key) => storage.delete(key),
    },
  })
  return { login: exports.AuthProvider({ children: null }).value.login, storage, state }
}

test('login obtains metadata from auth/check using the newly saved tokens', async () => {
  let checks = 0
  const metadata = {
    user: { nickname: 'Alice' },
    context: { space: 'home' },
    version: { appVersion: 'test' },
  }
  const h = loginHarness(async () => {
    checks++
    return { data: metadata }
  })
  const result = await h.login({ auth: 'ldap' })
  assert.equal(checks, 1)
  assert.equal(h.state.get('version'), metadata.version)
  assert.equal(h.state.get('context'), metadata.context)
  assert.equal(h.state.get('user').space, 'home')
  assert.equal(h.state.get('authenticated'), true)
  assert.equal(result.data.user, metadata.user)
  assert.equal('version' in result.data, false)
})

test('failed or empty post-login checks do not leave a partial authenticated session', async () => {
  for (const check of [
    async () => {
      throw new TypeError('offline')
    },
    async () => ({ data: undefined }),
  ]) {
    const h = loginHarness(check)
    await assert.rejects(h.login({ auth: 'ldap' }))
    assert.equal(h.storage.size, 0)
    assert.equal(h.state.get('authenticated'), false)
    assert.equal(h.state.get('version'), undefined)
    assert.equal(h.state.get('user'), undefined)
    assert.equal(h.state.get('context'), undefined)
  }
})
