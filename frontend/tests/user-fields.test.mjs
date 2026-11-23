import assert from 'node:assert/strict'
import { test } from 'node:test'
import { userFieldLabel, visibleUserFields } from '../src/services/user-fields.ts'

test('deployment labels prefer exact locale, base language, then built-in translation', () => {
  const fields = { teacher: { labels: { 'en-US': 'Manager', en: 'Supervisor', zh: '直属领导', ja: ' ' } } }
  assert.equal(userFieldLabel(fields, 'teacher', 'en-US', 'Advisor'), 'Manager')
  assert.equal(userFieldLabel(fields, 'teacher', 'en-GB', 'Advisor'), 'Supervisor')
  assert.equal(userFieldLabel(fields, 'teacher', 'zh-CN', '导师'), '直属领导')
  assert.equal(userFieldLabel(fields, 'teacher', 'ja', '指導教員'), '指導教員')
  assert.equal(userFieldLabel({}, 'identity', 'en', 'Identity'), 'Identity')
})

test('table and form visibility are independent and hidden fields are not generated', () => {
  const fields = {
    teacher: { showInTable: false, showInForm: true },
    identity: { showInTable: true, showInForm: false },
    lastLoginAt: { showInTable: true, showInForm: false },
  }
  assert.deepEqual(visibleUserFields(fields, 'showInTable'), ['identity', 'lastLoginAt'])
  assert.deepEqual(visibleUserFields(fields, 'showInForm'), ['teacher'])
  assert.deepEqual(visibleUserFields({}, 'showInTable'), [])
})

test('user role and status badges resolve labels again after changing language', async () => {
  const { readFileSync } = await import('node:fs')
  const { runInNewContext } = await import('node:vm')
  const { default: ts } = await import('typescript')
  const translations = Object.fromEntries(
    ['zhCN', 'enUS', 'ja', 'ko'].map((language) => [
      language,
      JSON.parse(readFileSync(new URL(`../src/i18n/locales/${language}/translation.json`, import.meta.url), 'utf8')),
    ])
  )
  let language = 'zhCN'
  const t = (key) => translations[language][key] ?? key
  const modules = {
    'react/jsx-runtime': { jsx: (_, props) => props },
    'react-i18next': { useTranslation: () => ({ t }) },
    '@/services/api/auth': { Role: { Admin: 3, User: 2 } },
    '@/services/api/account': { ProjectStatus: { Active: 2, Inactive: 3 } },
    './phase-badge': { PhaseBadge() {} },
  }
  function loadBadge(name) {
    const source = readFileSync(new URL(`../src/components/badge/${name}.tsx`, import.meta.url), 'utf8')
    const { outputText } = ts.transpileModule(source, {
      compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX },
    })
    const exports = {}
    runInNewContext(outputText, {
      exports,
      require(name) {
        assert.ok(name in modules, `Unexpected badge dependency: ${name}`)
        return modules[name]
      },
    })
    return exports
  }
  const roles = loadBadge('user-role-badge')
  const status = loadBadge('user-status-badge')
  for (language of ['zhCN', 'enUS', 'ja', 'ko', 'zhCN']) {
    assert.equal(roles.default({ role: '2' }).getPhaseLabel('2').label, t('userTable.roles.2'))
    assert.equal(roles.getUserRoles(t)[0].label, t('userTable.roles.3'))
    assert.equal(status.default({ status: '2' }).getPhaseLabel('2').label, t('userFields.status.active'))
    assert.equal(status.default({ status: '3' }).getPhaseLabel('3').label, t('userFields.status.inactive'))
    assert.equal(roles.default({ role: 'unknown' }).getPhaseLabel('unknown').label, t('userFields.unknown'))
    assert.equal(status.default({ status: 'unknown' }).getPhaseLabel('unknown').label, t('userFields.unknown'))
  }
})
