/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import i18next from 'i18next'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(
  new URL('../src/components/agent/agent-response-progress.ts', import.meta.url),
  'utf8'
)
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2020 },
})
const { getAgentResponseProgress: progress } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)
const auditSource = await readFile(
  new URL('../src/components/agent/llm-config-audit.ts', import.meta.url),
  'utf8'
)
const { outputText: auditOutput } = ts.transpileModule(auditSource, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2020 },
})
const { getLLMConfigAuditRows } = await import(
  `data:text/javascript;base64,${Buffer.from(auditOutput).toString('base64')}`
)
const examplesSource = await readFile(
  new URL('../src/components/agent/agent-welcome-examples.ts', import.meta.url),
  'utf8'
)
const { outputText: examplesOutput } = ts.transpileModule(examplesSource, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2020 },
})
const { agentWelcomeExampleIDs, pickAgentWelcomeExamples } = await import(
  `data:text/javascript;base64,${Buffer.from(examplesOutput).toString('base64')}`
)

test('welcome samples three distinct examples without modifying the pool', () => {
  const original = [...agentWelcomeExampleIDs]
  assert.ok(original.length >= 20)
  assert.equal(new Set(original).size, original.length)
  assert.deepEqual(
    pickAgentWelcomeExamples(() => 0),
    original.slice(0, 3)
  )
  let rank = 0
  assert.deepEqual(
    pickAgentWelcomeExamples(() => --rank),
    original.slice(-3).reverse()
  )
  assert.deepEqual(agentWelcomeExampleIDs, original)
})

test('repeated welcome samples can reach every example', () => {
  let seed = 12345
  const random = () => {
    seed = (Math.imul(seed, 1664525) + 1013904223) >>> 0
    return seed / 2 ** 32
  }
  const seen = new Set()
  for (let attempt = 0; attempt < 100; attempt++) {
    const selected = pickAgentWelcomeExamples(random)
    assert.equal(selected.length, 3)
    assert.equal(new Set(selected).size, 3)
    for (const id of selected) {
      assert.ok(agentWelcomeExampleIDs.includes(id))
      seen.add(id)
    }
  }
  assert.equal(seen.size, agentWelcomeExampleIDs.length)
})
const user = { id: 'user-1', role: 'user', parts: [{ type: 'text', text: 'List my jobs' }] }
const answer = (parts) => ({ id: 'assistant-1', role: 'assistant', parts })
const text = (state = 'done') => ({ type: 'text', text: 'Answer', state })
const tool = (state, output) => ({ type: 'tool-list_jobs', toolCallId: 'call-1', state, output })

test('submission and empty streams have progress before any assistant text exists', () => {
  assert.equal(progress([user], 'submitted').phase, 'waiting')
  assert.equal(progress([user], 'submitted').messageID, undefined)
  assert.equal(progress([user], 'streaming').phase, 'processing')
  assert.equal(progress([user, answer([])], 'streaming').messageID, 'assistant-1')
})

test('reasoning-only streams show processing without disclosing model reasoning', () => {
  const result = progress(
    [user, answer([{ type: 'reasoning', text: 'private-model-reasoning' }])],
    'streaming'
  )
  assert.equal(result.phase, 'processing')
  assert.equal(JSON.stringify(result).includes('private-model-reasoning'), false)
})

test('streaming text and automatic tools select the actual active phase', () => {
  assert.equal(progress([user, answer([text('streaming')])], 'streaming').phase, 'responding')
  for (const state of ['input-streaming', 'input-available']) {
    assert.equal(progress([user, answer([text(), tool(state)])], 'streaming').phase, 'tools')
  }
  assert.equal(
    progress([user, answer([text(), tool('output-available', {})])], 'streaming').phase,
    'processing'
  )
})

test('completed generation does not expose tool counters or evidence in progress', () => {
  const result = progress(
    [
      user,
      answer([
        tool('output-available', { status: 'success' }),
        tool('output-error'),
        tool('output-available', { status: 'partial-error' }),
        tool('output-available', { status: 'error' }),
      ]),
    ],
    'ready',
    'complete'
  )
  assert.equal(result.phase, 'complete')
  assert.deepEqual(Object.keys(result).sort(), ['messageID', 'phase'])
})

test('interactive tools await confirmation and rejected actions are not query errors', () => {
  for (const type of [
    'tool-navigate_to_page',
    'tool-navigate_to_job_detail',
    'tool-propose_form_patch',
  ]) {
    const pending = { ...tool('input-available'), type }
    assert.equal(progress([user, answer([pending])], 'ready', 'complete').phase, 'confirmation')
    const rejected = { ...pending, state: 'output-available', output: { status: 'rejected' } }
    const result = progress([user, answer([rejected])], 'ready', 'complete')
    assert.equal(result.phase, 'complete')
  }
})

test('stop and failure terminate progress even with unfinished tool calls', () => {
  for (const parts of [[], [text('streaming'), tool('input-available')]]) {
    for (const outcome of ['stopped', 'failed']) {
      assert.equal(progress([user, answer(parts)], 'ready', outcome).phase, outcome)
    }
  }
  assert.equal(progress([user], 'error').phase, 'failed')
  assert.equal(progress([user], 'ready', 'failed').phase, 'failed')
})

test('restored histories and new turns do not inherit stale progress', () => {
  const old = answer([text(), tool('output-available', {})])
  assert.equal(progress([user, old], 'ready'), undefined)
  assert.equal(progress([], 'ready'), undefined)
  const result = progress([user, old, { ...user, id: 'user-2' }], 'streaming')
  assert.equal(result.messageID, undefined)
  assert.equal(progress([user, old], 'submitted', 'complete').phase, 'waiting')
  assert.equal(progress([user, old], 'submitted', 'complete').messageID, undefined)
})

test('continuation waits for fresh content rather than treating old text as streaming', () => {
  assert.equal(progress([user, answer([text()])], 'streaming', 'complete').phase, 'processing')
})

for (const language of ['zhCN', 'enUS', 'ja', 'ko']) {
  const translation = JSON.parse(
    await readFile(
      new URL(`../src/i18n/locales/${language}/translation.json`, import.meta.url),
      'utf8'
    )
  )
  const instance = i18next.createInstance()
  await instance.init({
    lng: language,
    resources: { [language]: { translation } },
    keySeparator: false,
    interpolation: { escapeValue: false },
  })
  test(`${language}: welcome and progress translations cover all states and identities`, () => {
    const keys = [
      'craterAgent.welcome',
      'craterAgent.intro.examples',
      ...['guide', 'jobs', 'resources', 'forms'].map(
        (key) => `craterAgent.intro.capability.${key}`
      ),
      ...['admin', 'user'].flatMap((identity) => [
        ...agentWelcomeExampleIDs.flatMap((key) => [
          `craterAgent.intro.example.${key}`,
          `craterAgent.intro.prompt.${key}.${identity}`,
        ]),
      ]),
      ...[
        'waiting',
        'processing',
        'tools',
        'responding',
        'confirmation',
        'complete',
        'stopped',
        'failed',
      ].map((key) => `craterAgent.progress.phase.${key}`),
    ]
    for (const key of keys) {
      assert.ok(instance.exists(key), key)
      const value = instance.t(key, { count: 1, total: 2, finished: 1 })
      assert.notEqual(value, key)
      assert.equal(value.includes('{{'), false, key)
    }
    assert.equal(
      new Set(agentWelcomeExampleIDs.map((id) => instance.t(`craterAgent.intro.example.${id}`)))
        .size,
      agentWelcomeExampleIDs.length
    )
    for (const identity of ['user', 'admin']) {
      const prompts = agentWelcomeExampleIDs.map((id) =>
        instance.t(`craterAgent.intro.prompt.${id}.${identity}`)
      )
      assert.equal(new Set(prompts).size, agentWelcomeExampleIDs.length)
      assert.equal(instance.exists(`craterAgent.intro.scope.${identity}`), false)
      assert.equal(instance.exists(`craterAgent.intro.confirmation.${identity}`), false)
      if (identity === 'admin') assert.ok(prompts.every((prompt) => !prompt.includes('/portal/')))
    }
  })
  test(`${language}: LLM audit details render only allowlisted metadata`, () => {
    const t = instance.t.bind(instance)
    const rows = getLLMConfigAuditRows(
      {
        scope: 'personal',
        userId: 7,
        keySubmitted: true,
        apiKey: 'audit-secret-sentinel',
        encryptedAPIKey: 'encrypted-sentinel',
        baseUrl: 'https://audit-secret-sentinel.example',
        modelName: 'secret-model',
        error: 'provider-echoed-audit-secret-sentinel',
      },
      t
    )
    assert.deepEqual(
      rows.map((row) => row.key),
      ['scope', 'userId', 'keySubmitted']
    )
    assert.equal(rows[2].value, t('operationLog.details.llm.keySubmitted.yes'))
    for (const secret of ['audit-secret-sentinel', 'encrypted-sentinel', 'secret-model']) {
      assert.equal(JSON.stringify(rows).includes(secret), false)
    }
    assert.equal(getLLMConfigAuditRows({ scope: 'platform' }, t).length, 1)
    assert.equal(
      getLLMConfigAuditRows({ scope: 'personal', keySubmitted: false }, t).at(-1).value,
      t('operationLog.details.llm.keySubmitted.no')
    )
    const malformed = getLLMConfigAuditRows(
      {
        scope: 'audit-secret-sentinel',
        userId: 'audit-secret-sentinel',
        keySubmitted: 'audit-secret-sentinel',
      },
      t
    )
    assert.equal(malformed.length, 1)
    assert.equal(malformed[0].value, t('operationLog.details.na'))
    for (const key of [
      'operationLog.type.UpdateLLMConfig',
      'operationLog.type.ResetLLMConfig',
      'operationLog.details.llm.scope',
      'operationLog.details.llm.scope.personal',
      'operationLog.details.llm.scope.platform',
      'operationLog.details.llm.userId',
      'operationLog.details.llm.keySubmitted',
      'operationLog.details.llm.keySubmitted.yes',
      'operationLog.details.llm.keySubmitted.no',
      'operationLog.details.llm.noSecrets',
    ])
      assert.ok(instance.exists(key), key)
  })
}
