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
import { QueryClient } from '@tanstack/react-query'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import { renderToStaticMarkup } from 'react-dom/server'
import ts from 'typescript'

const require = createRequire(import.meta.url)
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

test('preloading user and admin details keeps permissions in separate caches', async () => {
  const calls = []
  const { queryBuildDetail } = await compile('../src/services/query/image.ts', {
    '@/services/api/imagepack': {
      apiUserGetKaniko: async (name) => {
        calls.push(['user', name])
        return { data: { canManage: false } }
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
    const userOptions = queryBuildDetail('public-build', false)
    const adminOptions = queryBuildDetail('public-build', true)
    await client.ensureQueryData(adminOptions)
    const user = await client.ensureQueryData(userOptions)
    assert.equal(user.data.canManage, false)
    assert.equal(client.getQueryData(adminOptions.queryKey).data.canManage, true)
    assert.deepEqual(calls, [
      ['admin', 'public-build'],
      ['user', 'public-build'],
    ])
    client.removeQueries({ queryKey: userOptions.queryKey })
    assert.equal(client.getQueryData(userOptions.queryKey), undefined)
    assert.equal(client.getQueryData(adminOptions.queryKey).data.canManage, true)
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
    assert.match(html, /Backend explanation/)
    assert.match(html, new RegExp(`HTTP ${status}`))
    assert.match(html, /imageBuild.detail.retry/)
    assert.doesNotMatch(html, /NOT_FOUND/)
  }
})
