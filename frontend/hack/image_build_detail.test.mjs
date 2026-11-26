// Copyright 2026 The Crater Project Team, RAIDS-Lab
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
import * as reactQuery from '@tanstack/react-query'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import { renderToStaticMarkup } from 'react-dom/server'
import ts from 'typescript'

const require = createRequire(import.meta.url)
const { QueryClient, isCancelledError } = reactQuery
const compile = async (path, dependencies) => {
  const source = await readFile(new URL(path, import.meta.url), 'utf8')
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      jsx: ts.JsxEmit.ReactJSX,
    },
  })
  const module = { exports: {} }
  runInNewContext('(function(require, module, exports) {' + outputText + '\n})', {})(
    (name) => dependencies[name] ?? require(name),
    module,
    module.exports
  )
  return module.exports
}

test('image detail caches keep account and admin permissions separate', async () => {
  const calls = []
  let context = { queue: 'account-a' }
  let permitted = true
  const forbidden = Object.assign(new Error('Permission denied'), { httpStatus: 403 })
  const { queryBuildDetail } = await compile('../src/services/query/image.ts', {
    '@tanstack/react-query': reactQuery,
    jotai: { getDefaultStore: () => ({ get: () => context }) },
    '@/utils/store': { atomUserContext: Symbol() },
    '@/services/api/imagepack': {
      apiUserGetKaniko: async (name) => {
        calls.push([context.queue, name])
        if (!permitted) throw forbidden
        return { data: { canManage: false, dockerfile: 'FROM private-image' } }
      },
    },
    '@/services/api/admin/imagepack': {
      apiAdminGetKaniko: async (name) => {
        calls.push(['admin', name])
        return { data: { canManage: true } }
      },
    },
    '@/services/api/vcjob': {},
  })
  const client = new QueryClient()
  try {
    const accountA = queryBuildDetail('shared-build', false, 'account-a')
    const accountB = queryBuildDetail('shared-build', false, 'account-b')
    const adminOptions = queryBuildDetail('shared-build', true)
    await client.ensureQueryData(adminOptions)
    const user = await client.ensureQueryData(accountA)
    assert.equal(user.data.canManage, false)
    assert.equal(client.getQueryData(adminOptions.queryKey).data.canManage, true)

    context = { queue: 'account-b' }
    permitted = false
    assert.equal(client.getQueryData(accountB.queryKey), undefined)
    await assert.rejects(client.fetchQuery(accountA), isCancelledError)
    assert.equal(client.getQueryState(accountA.queryKey).status, 'success')
    await assert.rejects(client.ensureQueryData(accountB), forbidden)
    assert.equal(client.getQueryData(accountA.queryKey).data.dockerfile, 'FROM private-image')
    assert.equal(client.getQueryData(accountB.queryKey), undefined)
    assert.deepEqual(calls, [
      ['admin', 'shared-build'],
      ['account-a', 'shared-build'],
      ['account-b', 'shared-build'],
    ])

    context = { queue: 'account-a' }
    await assert.rejects(client.fetchQuery(accountA), forbidden)
    assert.equal(client.getQueryState(accountA.queryKey).error, forbidden)
    assert.equal(client.getQueryData(accountA.queryKey).data.dockerfile, 'FROM private-image')
  } finally {
    client.clear()
  }
})

test('detail error shows 404 only for missing resources and preserves API error facts', async () => {
  const { default: DetailError } = await compile(
    '../src/components/image/registry/registry-detail-error.tsx',
    {
      '@tanstack/react-router': { useRouter: () => ({ invalidate: () => {} }) },
      'react-i18next': { useTranslation: () => ({ t: (key) => key }) },
      '@/components/ui/button': { Button: ({ children }) => children },
      '@/components/placeholder/not-found': { default: () => 'NOT_FOUND' },
    }
  )
  assert.equal(
    renderToStaticMarkup(DetailError({ error: Object.assign(new Error(), { httpStatus: 404 }) })),
    'NOT_FOUND'
  )
  for (const status of [403, 500]) {
    const error = Object.assign(new Error('fallback'), {
      httpStatus: status,
      data: { code: status * 100 + 1, msg: 'Backend explanation' },
    })
    const html = renderToStaticMarkup(DetailError({ error }))
    const title = status === 403 ? 'accessDenied' : 'loadError'
    assert.ok(html.includes(`imageBuild.detail.${title}`))
    assert.match(html, /Backend explanation/)
    assert.match(html, new RegExp(`HTTP ${status}`))
    assert.match(html, /imageBuild.detail.retry/)
    assert.doesNotMatch(html, /NOT_FOUND/)
  }
})
