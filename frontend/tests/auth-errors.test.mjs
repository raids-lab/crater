import { MutationObserver, QueryClient, onlineManager } from '@tanstack/react-query'
import { HTTPError, TimeoutError } from 'ky'
import assert from 'node:assert/strict'
import { test } from 'node:test'

import { getAuthErrorMessage } from '../src/services/auth-errors.ts'
import { authMutationOptions } from '../src/services/auth-mutation.ts'
import {
  ERROR_ACCOUNT_DISABLED,
  ERROR_ACCOUNT_LOCKED,
  ERROR_ACCOUNT_NOT_ACTIVATED,
  ERROR_USER_BANNED,
} from '../src/services/error_code.ts'

test('network and timeout failures always have actionable inline feedback', () => {
  const request = new Request('https://example.invalid/api/auth/login')
  assert.match(getAuthErrorMessage(new TimeoutError(request)), /超时.*重试/)
  assert.match(getAuthErrorMessage(new TypeError('Failed to fetch')), /检查网络连接/)
  assert.match(getAuthErrorMessage(new Error('internal implementation detail')), /稍后重试/)
  assert.match(getAuthErrorMessage(null), /操作失败/)
})

test('backend diagnostics stay out of user-facing authentication hints', () => {
  const request = new Request('https://example.invalid/api/auth/login')
  const error = new HTTPError(new Response('', { status: 500 }), request, {})
  error.data = {
    code: 50001,
    msg: 'LDAP authentication succeeded, but the user SID could not be mapped to a UID.',
  }
  const message = getAuthErrorMessage(error)
  assert.match(message, /服务暂不可用.*重试/)
  assert.doesNotMatch(message, /LDAP|SID|HTTP|50001|密码错误/)
})

test('non-JSON responses and empty messages still produce feedback', () => {
  const request = new Request('https://example.invalid/api/auth/signup')
  const error = new HTTPError(new Response('<html>gateway</html>', { status: 502 }), request, {})
  assert.match(getAuthErrorMessage(error), /服务暂不可用.*重试/)
  error.data = { code: 50201, msg: '' }
  assert.match(getAuthErrorMessage(error), /服务暂不可用.*重试/)
})

test('account state hints distinguish activation, disabling, locking, and bans', () => {
  const request = new Request('https://example.invalid/api/auth/login')
  const messages = new Set()
  for (const [status, code, expected] of [
    [401, ERROR_ACCOUNT_NOT_ACTIVATED, /未激活/],
    [403, ERROR_ACCOUNT_DISABLED, /已停用/],
    [401, ERROR_ACCOUNT_LOCKED, /已锁定/],
    [403, ERROR_USER_BANNED, /已封禁/],
  ]) {
    const error = new HTTPError(new Response('', { status }), request, {})
    error.data = { code, msg: 'Internal authentication diagnostics' }
    const message = getAuthErrorMessage(error)
    assert.match(message, expected)
    assert.match(message, /联系管理员/)
    assert.doesNotMatch(message, /密码错误|LDAP|HTTP|diagnostics/)
    messages.add(message)
  }
  assert.equal(messages.size, 4)
})

test(
  'offline authentication settles, reports the error, and does not replay on reconnect',
  { timeout: 2000 },
  async () => {
    const wasOnline = onlineManager.isOnline()
    const client = new QueryClient({ defaultOptions: { mutations: { retry: 3 } } })
    client.mount()
    let attempts = 0
    let message
    const observer = new MutationObserver(client, {
      ...authMutationOptions,
      mutationFn: async () => {
        attempts += 1
        throw new TypeError('Failed to fetch')
      },
      onError: (error) => {
        message = getAuthErrorMessage(error)
      },
    })

    try {
      onlineManager.setOnline(false)
      await assert.rejects(observer.mutate(), TypeError)
      assert.equal(attempts, 1)
      assert.equal(observer.getCurrentResult().isPending, false)
      assert.equal(observer.getCurrentResult().isPaused, false)
      assert.match(message, /检查网络连接/)

      onlineManager.setOnline(true)
      await new Promise(setImmediate)
      assert.equal(attempts, 1)
    } finally {
      client.unmount()
      client.clear()
      onlineManager.setOnline(wasOnline)
    }
  }
)

test('local login failures use short accurate hints without assuming a directory is enabled', () => {
  const request = new Request('https://example.invalid/api/auth/login')
  for (const [status, code, expected] of [
    [401, 40105, /未找到该账号/],
    [401, 40106, /账号或密码错误/],
    [401, 40110, /切换登录方式或联系管理员/],
    [403, 40305, /切换登录方式/],
    [403, 40307, /暂无可用项目/],
    [500, 50002, /服务暂不可用/],
  ]) {
    const error = new HTTPError(new Response('', { status }), request, {})
    error.data = { code, msg: 'Internal technical details' }
    const hint = getAuthErrorMessage(error)
    assert.match(hint, expected)
    assert.doesNotMatch(hint, /LDAP|HTTP|technical|\n/)
    assert(hint.length <= 18)
  }
})
