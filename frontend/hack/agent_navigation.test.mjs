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
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const compilerOptions = { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2020 }
const capabilitySource = await readFile(
  new URL('../src/components/agent/form-capabilities.ts', import.meta.url),
  'utf8'
)
const { outputText } = ts.transpileModule(capabilitySource, { compilerOptions })
const { needsAgentFormOptions, isAgentNavigationBlocked, isAgentNavigationDestination } =
  await import(`data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`)
const panelSource = await readFile(
  new URL('../src/components/agent/crater-agent-panel.tsx', import.meta.url),
  'utf8'
)
const panelAST = ts.createSourceFile(
  'panel.tsx',
  panelSource,
  ts.ScriptTarget.Latest,
  true,
  ts.ScriptKind.TSX
)

// Exercise the production event handlers without mounting the entire chat panel.
function panelHandler(name, dependencies) {
  let initializer
  function visit(node) {
    if (ts.isVariableDeclaration(node) && node.name.getText(panelAST) === name) {
      initializer = node.initializer
    }
    ts.forEachChild(node, visit)
  }
  visit(panelAST)
  assert.ok(initializer, name)
  const compiled = ts.transpileModule(
    `const implementation = ${initializer.getText(panelAST)}; implementation;`,
    { compilerOptions }
  )
  return runInNewContext(compiled.outputText, { ...dependencies })
}

const jupyter = '/portal/jobs/new/jupyter-job'
const images = '/portal/env/images'
const navigation = (id, path, assistForm = false) => ({
  type: 'tool-navigate_to_page',
  toolCallId: id,
  state: 'input-available',
  input: { path, assistForm },
})

function navigationHarness(parts, ready = true) {
  const events = []
  const notices = []
  const dependencies = {
    messages: [{ id: 'answer', parts }],
    t: (key) => key,
    isAgentNavigationBlocked,
    isAgentNavigationDestination,
    needsAgentFormOptions,
    identity: 'user',
    formRuntimeRef: { current: undefined },
    withToolAction: (action) => action(),
    navigate: async ({ to }) => {
      events.push(`navigate:${to}`)
    },
    waitForFormOptions: async () => {
      events.push('wait')
      return ready
    },
    toast: { error: (message) => notices.push(message) },
    saveToolOutput: async (_, part, output) => {
      events.push(`save:${part.toolCallId}`)
      part.state = 'output-available'
      part.output = output
    },
  }
  dependencies.ensureNavigationOrder = panelHandler('ensureNavigationOrder', dependencies)
  return { events, notices, navigate: panelHandler('navigateFromTool', dependencies) }
}

test('pure Jupyter and image-list navigation skips unopened forms and preserves card order', async () => {
  const first = navigation('jupyter', jupyter)
  const second = navigation('images', images)
  const readResult = {
    type: 'tool-list_nodes',
    toolCallId: 'nodes',
    state: 'output-available',
    output: [{ name: 'worker' }],
  }
  const harness = navigationHarness([readResult, first, second])
  await assert.rejects(harness.navigate('answer', second, images), /navigationWaiting/)
  assert.deepEqual(harness.events, [])
  await harness.navigate('answer', first, jupyter)
  await harness.navigate('answer', second, images)
  assert.deepEqual(harness.events, [
    `navigate:${jupyter}`,
    'save:jupyter',
    `navigate:${images}`,
    'save:images',
  ])
  assert.deepEqual(harness.notices, [])
})

test('old navigation cards without assistForm also skip form loading', async () => {
  const part = navigation('legacy', images)
  delete part.input.assistForm
  const harness = navigationHarness([part])
  await harness.navigate('answer', part, images)
  assert.deepEqual(harness.events, [`navigate:${images}`, 'save:legacy'])
  assert.deepEqual(harness.notices, [])
})

test('explicit form assistance still waits for options', async () => {
  const part = navigation('assist', jupyter, true)
  const harness = navigationHarness([part])
  await harness.navigate('answer', part, jupyter)
  assert.deepEqual(harness.events, [`navigate:${jupyter}`, 'wait', 'save:assist'])
  assert.deepEqual(harness.notices, [])
})

test('missing requested forms report readiness rather than invalid GPU or image values', async () => {
  const part = navigation('assist', jupyter, true)
  const harness = navigationHarness([part], false)
  await harness.navigate('answer', part, jupyter)
  assert.deepEqual(harness.notices, ['craterAgent.formNotReady'])
})

test('navigation keeps identity checks and cannot open an administrator page as a user', async () => {
  const part = navigation('forbidden', '/admin/jobs')
  const harness = navigationHarness([part])
  await assert.rejects(harness.navigate('answer', part, '/admin/jobs'), /navigationNotAllowed/)
  assert.deepEqual(harness.events, [])
})

test('resolved, canceled, failed, or incomplete navigation does not deadlock later cards', () => {
  const first = navigation('first', jupyter)
  const second = navigation('second', images)
  assert.equal(isAgentNavigationBlocked([first, second], 'first'), false)
  assert.equal(isAgentNavigationBlocked([first, second], 'second'), true)
  for (const state of ['output-available', 'output-error', 'input-streaming']) {
    assert.equal(isAgentNavigationBlocked([{ ...first, state }, second], 'second'), false)
  }
  assert.equal(
    isAgentNavigationBlocked(
      [{ ...first, state: 'output-available', output: { status: 'rejected' } }, second],
      'second'
    ),
    false
  )
  assert.equal(
    isAgentNavigationBlocked([{ ...first, type: 'tool-navigate_to_job_detail' }, second], 'second'),
    true
  )
  assert.equal(
    isAgentNavigationBlocked([first, { ...second, type: 'tool-propose_form_patch' }], 'second'),
    false
  )
  assert.equal(isAgentNavigationBlocked([first], 'stale-tool'), true)
  assert.equal(needsAgentFormOptions({ assistForm: true }, '/portal/overview'), false)
})

for (const locale of ['zhCN', 'enUS', 'ja', 'ko']) {
  test(`${locale}: navigation feedback is translated`, async () => {
    const translation = JSON.parse(
      await readFile(
        new URL(`../src/i18n/locales/${locale}/translation.json`, import.meta.url),
        'utf8'
      )
    )
    for (const key of ['craterAgent.formNotReady', 'craterAgent.navigationWaiting']) {
      assert.equal(typeof translation[key], 'string')
      assert.ok(translation[key].trim())
    }
  })
}
