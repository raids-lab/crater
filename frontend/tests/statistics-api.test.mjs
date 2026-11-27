import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import vm from 'node:vm'
import ts from 'typescript'

const source = readFileSync(new URL('../src/services/api/statistics.ts', import.meta.url), 'utf8')
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS },
})
const apiModule = { exports: {} }
vm.runInNewContext(outputText, {
  exports: apiModule.exports,
  require: (name) => {
    assert.equal(name, '@/services/client')
    return {
      apiV1Get: (path, options) => ({ path, query: new URLSearchParams(options.searchParams) }),
    }
  },
})
const {
  apiGetStatistics,
  apiGetStatisticsRankings,
  apiAdminGetStatistics,
  apiAdminGetStatisticsRankings,
} = apiModule.exports
const request = {
  startTime: '2026-11-01T00:00:00Z',
  endTime: '2026-11-02T00:00:00Z',
  step: 'day',
  scope: 'cluster',
  targetID: undefined,
}

test('cluster statistics omit undefined targetID instead of sending an invalid integer', () => {
  const { path, query } = apiAdminGetStatistics(request)
  assert.equal(path, 'admin/statistics')
  assert.equal(query.has('targetID'), false)
  assert.equal(query.get('scope'), 'cluster')
  assert.equal(query.get('startTime'), request.startTime)
})

test('rankings preserve exact resource selectors and scoped account IDs', () => {
  const { path, query } = apiGetStatisticsRankings({
    ...request,
    scope: 'account',
    targetID: 7,
    dimension: 'user',
    metric: 'utilization',
    order: 'asc',
    page: 2,
    pageSize: 20,
    resourceName: 'vendor.example/gpu-model',
    resourceType: undefined,
  })
  assert.equal(path, 'statistics/rankings')
  assert.equal(query.get('targetID'), '7')
  assert.equal(query.get('resourceName'), 'vendor.example/gpu-model')
  assert.equal(query.has('resourceType'), false)
  assert.equal(query.get('page'), '2')
})

test('overview and rankings use their matching permission routes', () => {
  assert.equal(apiGetStatistics({ ...request, scope: 'user', targetID: 3 }).path, 'statistics')
  assert.equal(
    apiAdminGetStatisticsRankings({
      ...request,
      dimension: 'user',
      metric: 'usage',
      order: 'desc',
      page: 1,
      pageSize: 20,
      resourceType: 'gpu',
    }).path,
    'admin/statistics/rankings'
  )
})

const displaySource = readFileSync(
  new URL('../src/components/statistics/statistics-display.ts', import.meta.url),
  'utf8'
)
const displayModule = { exports: {} }
vm.runInNewContext(
  ts.transpileModule(displaySource, { compilerOptions: { module: ts.ModuleKind.CommonJS } })
    .outputText,
  { exports: displayModule.exports }
)
const {
  allocatedChartValue,
  formatStatisticsNumber,
  hasStatisticsProfile,
  statisticsDateRange,
  statisticsResourceLabel,
} = displayModule.exports

test('personal statistics use the allocated-only route and do not infer missing profile fields', () => {
  const personalRequest = { ...request, scope: 'user', targetID: 3 }
  assert.equal(apiGetStatistics(personalRequest).path, 'statistics')
  assert.equal(apiAdminGetStatistics(personalRequest).path, 'admin/statistics')

  const allocated = { usage: 5, label: 'CPU', type: 'common' }
  const personalPoint = { timestamp: request.startTime, status: 'complete', usage: { cpu: 5 } }
  const personal = { totalUsage: { cpu: allocated }, series: [personalPoint] }
  assert.equal(hasStatisticsProfile(personal), false)
  for (const field of ['effectiveUsage', 'profiledUsage', 'utilization', 'coverage']) {
    assert.equal(Object.hasOwn(personal.totalUsage.cpu, field), false)
  }
  assert.equal(Object.hasOwn(personal.series[0], 'effectiveUsage'), false)

  const admin = {
    totalUsage: {
      cpu: { ...allocated, effectiveUsage: 0, profiledUsage: 5, utilization: 0, coverage: 1 },
    },
    series: [{ ...personalPoint, effectiveUsage: { cpu: 0 } }],
  }
  assert.equal(hasStatisticsProfile(admin), true)
  assert.equal(admin.series[0].effectiveUsage.cpu, 0)
})

test('chart data retains completed zeroes and leaves pending or partial days blank', () => {
  assert.equal(allocatedChartValue({ status: 'complete', usage: {} }, 'cpu'), 0)
  assert.equal(allocatedChartValue({ status: 'pending', usage: {} }, 'cpu'), null)
  assert.equal(allocatedChartValue({ status: 'partial', usage: { cpu: 3 } }, 'cpu'), null)
})

test('preset range advances at UTC rollover and custom dates stay fixed', () => {
  assert.equal(statisticsDateRange('2026-12-31', 30).endTime, '2027-01-01T00:00:00.000Z')
  assert.equal(statisticsDateRange('2027-01-01', 30).endTime, '2027-01-02T00:00:00.000Z')
  assert.equal(statisticsDateRange('2027-01-01', 30).startTime, '2026-12-03T00:00:00.000Z')
  const custom = { from: new Date(2026, 11, 28), to: new Date(2026, 11, 29) }
  const before = statisticsDateRange('2026-12-31', 30, custom)
  assert.deepEqual(statisticsDateRange('2027-01-01', 30, custom), before)
})

test('standard resources use the current locale before reading API labels', () => {
  for (const locale of ['enUS', 'zhCN', 'ja', 'ko']) {
    const messages = JSON.parse(
      readFileSync(new URL(`../src/i18n/locales/${locale}/translation.json`, import.meta.url))
    )
    const t = (key) => messages[key]
    for (const [id, key] of [
      ['cpu', 'statistics.cpu'],
      ['memory', 'statistics.memory'],
      ['__all_gpu__', 'statistics.allGPU'],
    ]) {
      assert.equal(typeof t(key), 'string')
      assert.equal(statisticsResourceLabel(t, id, '内存 (GiB)'), t(key))
    }
    assert.equal(statisticsResourceLabel(t, 'example.com/gpu', 'Vendor GPU'), 'Vendor GPU')
    assert.equal(typeof t('statistics.incompleteChartDays'), 'string')
  }
})

test('usage below 10,000 keeps full precision without a compact tooltip', () => {
  for (const value of [0, 42.5, 9876.54, -9999]) {
    const exact = value.toLocaleString(undefined, { maximumFractionDigits: 2 })
    const result = formatStatisticsNumber(value)
    assert.equal(result.text, exact)
    assert.equal(result.exact, exact)
    assert.equal(result.compact, false)
  }
})

test('usage at or above 10,000 is compacted and keeps the exact value for the tooltip', () => {
  for (const [value, text] of [
    [10_000, '10K'],
    [163_576.84, '163.6K'],
    [964_524.12, '964.5K'],
    [1_240_000, '1.2M'],
    [3_500_000_000, '3.5B'],
    [-25_000, '-25K'],
  ]) {
    const result = formatStatisticsNumber(value)
    assert.equal(result.text, text)
    assert.equal(result.exact, value.toLocaleString(undefined, { maximumFractionDigits: 2 }))
    assert.equal(result.compact, true)
  }
})
