import i18next from 'i18next'
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
import test from 'node:test'
import ts from 'typescript'

const source = await readFile(
  new URL('../src/components/model/model-download-result.ts', import.meta.url),
  'utf8'
)
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.ESNext, target: ts.ScriptTarget.ES2022 },
})
const { getModelDownloadReuseNotice } = await import(
  `data:text/javascript;base64,${Buffer.from(outputText).toString('base64')}`
)
const actual = {
  source: 'modelscope',
  revision: 'v1',
  status: 'Ready',
  path: 'public/Models/owner/model',
}

for (const language of ['enUS', 'zhCN', 'ja', 'ko']) {
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
  const t = instance.t.bind(instance)

  test(`${language}: matched source and revision do not warn`, () => {
    assert.equal(getModelDownloadReuseNotice(actual, actual, t), null)
  })

  test(`${language}: different revision reports the unmet request and actual fields`, () => {
    const notice = getModelDownloadReuseNotice({ source: 'huggingface', revision: 'v2' }, actual, t)
    assert.equal(notice.title, t('modelDownload.dialog.reuseRevisionWarning'))
    for (const value of ['huggingface', 'v2', 'modelscope', 'v1', 'Ready', actual.path]) {
      assert.ok(notice.description.includes(value), value)
    }
  })

  test(`${language}: source-only mismatch warns without claiming a revision mismatch`, () => {
    const notice = getModelDownloadReuseNotice({ source: 'huggingface', revision: 'v1' }, actual, t)
    assert.equal(notice.title, t('modelDownload.dialog.reuseSourceWarning'))
  })

  test(`${language}: empty revisions are not presented as confirmed default branches`, () => {
    for (const [request, download] of [
      [{ source: 'modelscope' }, actual],
      [
        { source: 'modelscope', revision: 'v2' },
        { ...actual, revision: '' },
      ],
    ]) {
      const notice = getModelDownloadReuseNotice(request, download, t)
      assert.ok(notice.description.includes(t('modelDownload.dialog.unspecifiedRevision')))
    }
  })

  test(`${language}: pending and paused resources retain their actual status`, () => {
    for (const status of ['Pending', 'Downloading', 'Paused', 'Failed']) {
      const notice = getModelDownloadReuseNotice(
        { source: 'modelscope', revision: 'v2' },
        { ...actual, status },
        t
      )
      assert.ok(notice.description.includes(status))
      assert.equal(notice.title, t('modelDownload.dialog.reuseRevisionWarning'))
    }
  })
}
