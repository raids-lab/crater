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
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { createRequire } from 'node:module'
import path from 'node:path'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'
import ts from 'typescript'

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const require = createRequire(import.meta.url)
const i18next = require('i18next')
const modules = new Map()
const requests = []
const effects = []
let tableState

// Execute the real TS modules. Only transport, app state and badge rendering are
// isolated; no phase classifier or option metadata is reproduced in this test.
function load(relative) {
  const filename = path.join(root, relative)
  if (modules.has(filename)) return modules.get(filename).exports
  const module = { exports: {} }
  modules.set(filename, module)
  const source = ts.transpileModule(readFileSync(filename, 'utf8'), {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      jsx: ts.JsxEmit.ReactJSX,
      target: ts.ScriptTarget.ES2022,
    },
  }).outputText
  const localRequire = (id) => {
    if (id === 'react' && filename.endsWith('use-job-table-state.ts'))
      return { useEffect: (effect) => effects.push(effect) }
    if (id === './use-remote-table-state') return { __esModule: true, default: () => tableState }
    if (id === 'jotai') return { getDefaultStore: () => ({ get: () => 'vcjobs' }) }
    if (id === '@/utils/store') return { globalJobUrl: 'job-url' }
    if (id === '@/services/client')
      return {
        apiV1Get: (url, options) => {
          requests.push({ url, options })
          return options
        },
      }
    if (id === '@/components/query-table/remote-state')
      return load('src/components/query-table/remote-state.ts')
    if (id === '@/services/api/vcjob') return load('src/services/api/vcjob.ts')
    if (id === '@/components/badge/job-phase-badge')
      return load('src/components/badge/job-phase-badge.tsx')
    if (id === '@/components/badge/job-type-badge')
      return load('src/components/badge/job-type-badge.tsx')
    if (id === './phase-badge') return { PhaseBadge: () => null }
    return createRequire(filename)(id)
  }
  vm.runInThisContext(`(function(exports,require,module){${source}\n})`, { filename })(
    module.exports,
    localRequire,
    module
  )
  return module.exports
}

await i18next.init({
  lng: 'zhCN',
  fallbackLng: 'enUS',
  keySeparator: false,
  resources: Object.fromEntries(
    ['zhCN', 'enUS', 'ja', 'ko'].map((lang) => [
      lang,
      {
        translation: JSON.parse(
          readFileSync(path.join(root, 'src/i18n/locales', lang, 'translation.json'), 'utf8')
        ),
      },
    ])
  ),
})

const api = load('src/services/api/vcjob.ts')
const badge = load('src/components/badge/job-phase-badge.tsx')
const toolbar = load('src/components/job/statuses.ts')
const fixtures = JSON.parse(
  readFileSync(
    path.join(root, '../backend/internal/handler/vcjob/testdata/filter_phases.json'),
    'utf8'
  )
)

test('frontend display mapping agrees with the backend SQL fixture expectations', () => {
  for (const row of fixtures) {
    const displayed = api.getJobDisplayPhase(row.status, row.podGroupPhase)
    assert.equal(api.encodeJobFilterPhase(displayed), row.phase, JSON.stringify(row))
  }
})

test('non-waiting and historical states ignore all PodGroup combinations', () => {
  const states = [...Object.values(api.JobPhase), 'FuturePhase'].filter(
    (status) => status !== '' && status !== 'Pending'
  )
  for (const status of states) {
    for (const group of [undefined, null, '', ...Object.values(api.PodGroupPhase), 'FuturePhase']) {
      const displayed = api.getJobDisplayPhase(status, group)
      assert.equal(api.encodeJobFilterPhase(displayed), status === 'Prequeue' ? 'Pending' : status)
    }
  }
})

test('filter candidates reuse label resolution in every locale and merge legacy Prequeue', async () => {
  for (const locale of ['zhCN', 'enUS', 'ja', 'ko']) {
    await i18next.changeLanguage(locale)
    const options = badge.getJobFilterOptions()
    assert.equal(options.filter(({ value }) => value === 'Pending').length, 1)
    assert.ok(
      !options.some(({ value }) => value === 'Prequeue' || value === 'Cancelled' || value === '')
    )
    assert.equal(options.length, new Set(options.map(({ value }) => value)).size)
    for (const option of options) {
      const key = option.value === 'Init' ? 'created' : option.value.toLowerCase()
      for (const part of ['label', 'description']) {
        const translationKey = `jobs.statuses.${key}.${part}`
        assert.ok(i18next.exists(translationKey, { lng: locale, fallbackLng: false }))
        assert.equal(
          badge.getJobPhaseLabel(option.value === 'Init' ? '' : option.value)[part],
          i18next.t(translationKey)
        )
      }
      if (locale === 'enUS') assert.doesNotMatch(option.label, /\p{Script=Han}/u)
      assert.equal(
        option.label,
        badge.getJobPhaseLabel(option.value === 'Init' ? '' : option.value).label
      )
    }
    for (const raw of badge.jobPhases)
      assert.equal(
        raw.label,
        raw.value === 'Pending'
          ? i18next.t('jobs.statuses.waiting.label')
          : badge.getJobPhaseLabel(raw.value).label
      )
    assert.notEqual(
      badge.jobPhases.find(({ value }) => value === 'Pending').label,
      options.find(({ value }) => value === 'Pending').label
    )
    assert.equal(badge.getJobPhaseLabel('Prequeue').label, badge.getJobPhaseLabel('Pending').label)
    const remote = toolbar.getRemoteJobToolbarConfig()
    for (const config of [toolbar.jobToolbarConfig, remote]) {
      assert.equal(config.filterOptions[0].title, i18next.t('jobs.filters.jobType'))
      assert.equal(config.filterOptions[1].title, i18next.t('jobs.filters.status'))
    }
  }
})

test('the real API adapter encodes UI phases, preserves unrelated filters and leaves input intact', () => {
  const input = {
    page: 2,
    page_size: 10,
    filters: {
      status: ['Prequeue', 'Pending', '', 'Starting'],
      jobType: ['custom'],
      days: ['7'],
    },
  }
  const before = structuredClone(input)
  api.apiJobBatchList(input)
  const params = requests.at(-1).options.searchParams
  assert.deepEqual(params.getAll('phase'), ['Pending', 'Init', 'Starting'])
  assert.deepEqual(params.getAll('status'), [])
  assert.deepEqual(params.getAll('job_type'), ['custom'])
  assert.equal(params.get('days'), '7')
  assert.equal(params.get('page'), '2')
  assert.deepEqual(input, before)
})

test('saved legacy selection migrates through the same encoder and preserves other filters', () => {
  const updates = []
  tableState = {
    columnFilters: [
      { id: 'status', value: ['Prequeue', 'Pending', 'Running'] },
      { id: 'days', value: ['7'] },
    ],
    setColumnFilters: (filters) => updates.push(filters),
  }
  const before = structuredClone(tableState.columnFilters)
  const hook = load('src/hooks/use-job-table-state.ts').default
  hook('test')
  effects.pop()()
  assert.deepEqual(updates[0], [
    { id: 'status', value: ['Pending', 'Running'] },
    { id: 'days', value: ['7'] },
  ])
  assert.deepEqual(tableState.columnFilters, before)
  tableState.columnFilters = updates[0]
  hook('test')
  effects.pop()()
  assert.equal(updates.length, 1)
})

test('display-stage sorting uses the existing job protocol adapter for every list', () => {
  for (const [sort, expected] of [
    ['status', 'phase'],
    ['-status', '-phase'],
    ['name,-status,createdAt', 'name,-phase,createdAt'],
    ['-createdAt', '-createdAt'],
  ]) {
    const params = {
      page: 2,
      page_size: 3,
      sort,
      filters: { status: ['Prequeue', 'Pending', 'Inqueue', 'Starting'] },
    }
    const before = structuredClone(params)
    for (const name of [
      'apiAdminGetJobList',
      'apiJobAllList',
      'apiJobSelfList',
      'apiJobBatchList',
      'apiJobInteractiveList',
    ]) {
      api[name](params)
      const query = requests.at(-1).options.searchParams
      assert.equal(query.get('sort'), expected, name)
      assert.deepEqual(query.getAll('phase'), ['Pending', 'Inqueue', 'Starting'])
      assert.equal(query.get('page'), '2')
      assert.equal(query.get('page_size'), '3')
    }
    for (const name of ['apiAdminGetUserJobList', 'apiGetUserJobs']) {
      api[name]('test-user', params)
      assert.equal(requests.at(-1).options.searchParams.get('sort'), expected, name)
    }
    assert.deepEqual(params, before)
  }
})

test('the job menu keeps all 15 candidates and zero counts under any selection', () => {
  const options = badge.getJobFilterOptions()
  assert.equal(options.length, 15)
  for (const facets of [
    {},
    {
      phase: [
        { value: 'Running', count: 8 },
        { value: 'Pending', count: 0 },
      ],
    },
  ]) {
    const config = toolbar.getRemoteJobToolbarConfig({ facets })
    const status = config.filterOptions.find(({ key }) => key === 'status')
    assert.equal(status.showZeroCountOptions, true)
    assert.equal(status.option.length, 15)
    assert.equal(status.option.filter(({ value }) => value === 'Pending').length, 1)
    assert.equal(status.option.find(({ value }) => value === 'Starting').count, 0)
    assert.equal(
      config.filterOptions.find(({ key }) => key === 'jobType').showZeroCountOptions,
      false
    )
  }
})
