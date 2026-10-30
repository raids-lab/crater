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
import { readFile } from 'node:fs/promises'
import { createRequire } from 'node:module'
import test from 'node:test'
import { runInNewContext } from 'node:vm'
import ts from 'typescript'

const require = createRequire(import.meta.url)
const compile = (source, dependencies = {}) => {
  const { outputText } = ts.transpileModule(source, {
    compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
  })
  const module = { exports: {} }
  runInNewContext('(function(require, module, exports) {' + outputText + '\n})', {})(
    (name) => dependencies[name] ?? require(name),
    module,
    module.exports
  )
  return module.exports
}
const imageSource = await readFile(
  new URL('../src/services/api/imagepack.ts', import.meta.url),
  'utf8'
)
const imageFile = ts.createSourceFile('imagepack.ts', imageSource, ts.ScriptTarget.Latest)
const enumSource = imageFile.statements
  .find((statement) => ts.isEnumDeclaration(statement) && statement.name.text === 'ImagePackSource')
  .getText(imageFile)
const metadata = compile(
  await readFile(new URL('../src/components/form/types.ts', import.meta.url), 'utf8'),
  { '@/services/api/imagepack': compile(enumSource) }
)
const { importFromJsonString } = compile(
  await readFile(new URL('../src/utils/form.ts', import.meta.url), 'utf8'),
  { '@/components/form/types': metadata }
)
const importTemplate = (form, version, data) =>
  importFromJsonString(form, JSON.stringify({ type: form.type, version, data }))

for (const [name, oldest, legacy] of [
  ['Jupyter', '20250420', '20260729'],
  ['WebIDE', '20251126', '20260729'],
  ['Custom', '20250317', '20260728'],
  ['Tensorflow', '20240528', '20260729'],
  ['Pytorch', '20240528', '20260729'],
]) {
  const form = metadata['MetadataForm' + name]
  for (const version of [oldest, '20260707', legacy, '20260922']) {
    test(
      name + ': imports saved template ' + version + ' after scheduler and TensorBoard changes',
      () => {
        const result = importTemplate(form, version, {
          scheduleType: 'backfill',
          envs: [{ name: 'TENSORBOARD_LOGDIR', value: ' /home/user/logs ' }],
          tensorboardLogDir: '',
          name: 'saved job',
        })
        assert.equal(result.tensorboardLogDir, '/home/user/logs')
        assert.equal(result.name, 'saved job')
        if (version !== '20260922') assert.equal('scheduleType' in result, false)
      }
    )
  }
  test(name + ': explicit log directory wins over the legacy environment variable', () => {
    const result = importTemplate(form, legacy, {
      tensorboardLogDir: ' /home/user/current ',
      envs: [{ name: 'TENSORBOARD_LOGDIR', value: '/home/user/old' }],
    })
    assert.equal(result.tensorboardLogDir, '/home/user/current')
  })
}

for (const version of ['20260707', '20260930']) {
  test('TensorBoard panel: imports saved template ' + version, () => {
    const result = importTemplate(metadata.MetadataFormTensorboard, version, {
      ttlHours: 24,
      logDir: '/home/user/logs',
    })
    assert.equal(result.logDir, '/home/user/logs')
    assert.equal(result.name, '')
    if (version === '20260707') assert.equal('ttlHours' in result, false)
  })
}

test('unrelated single-job templates keep their scheduler migration', () => {
  const result = importTemplate(metadata.MetadataFormSingle, '20260707', {
    scheduleType: 'backfill',
    name: 'single job',
  })
  assert.equal('scheduleType' in result, false)
  assert.equal('tensorboardLogDir' in result, false)
})
