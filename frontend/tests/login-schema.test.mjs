/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { zodResolver } from '@hookform/resolvers/zod'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { test } from 'node:test'
import { fileURLToPath } from 'node:url'
import vm from 'node:vm'
import { createFormControl } from 'react-hook-form'
import ts from 'typescript'
import { z } from 'zod'

// Transpile the production schema so these tests also run on Node 22.
const source = readFileSync(
  new URL('../src/routes/auth/-components/login-schema.ts', import.meta.url),
  'utf8'
)
const { outputText } = ts.transpileModule(source, {
  compilerOptions: { module: ts.ModuleKind.CommonJS },
})
const schemaModule = { exports: {} }
vm.runInNewContext(outputText, {
  exports: schemaModule.exports,
  require: (name) => {
    assert.equal(name, 'zod')
    return { z }
  },
})
const { createLoginFormSchema } = schemaModule.exports
const schema = createLoginFormSchema((key) => key)
const resolver = zodResolver(schema)
const validForm = { username: 'alice', password: ' PaSsWord ', acceptPrivacy: true }
const resolve = (values) => resolver(values, {}, { fields: {}, shouldUseNativeValidation: false })

test('normal and LDAP login normalize retained uppercase values before validation', async () => {
  const form = createFormControl({ resolver, defaultValues: validForm })
  form.register('username')
  form.register('password')
  form.register('acceptPrivacy')
  // Simulate a retained/autofilled value that bypassed the input's onChange.
  form.setValue('username', '  ALIce-01  ')
  let auth = 'normal'
  const submissions = []
  const submit = form.handleSubmit(
    (values) => submissions.push({ ...values, auth }),
    (errors) => assert.fail(JSON.stringify(errors))
  )
  await submit()
  auth = 'ldap'
  await submit()
  auth = 'normal'
  await submit()
  assert.deepEqual(
    submissions,
    ['normal', 'ldap', 'normal'].map((mode) => ({
      ...validForm,
      username: 'alice-01',
      auth: mode,
    }))
  )
})

test('existing lowercase usernames and boundary lengths remain valid', async () => {
  for (const username of ['ab', 'alice', 'alice-01', 'a'.repeat(20)]) {
    const result = await resolve({ ...validForm, username })
    assert.deepEqual(result.errors, {})
    assert.equal(result.values.username, username)
  }
})

test('normalization does not weaken username syntax or length constraints', async () => {
  for (const username of [
    '',
    '   ',
    'A',
    '1alice',
    '-alice',
    'alice-',
    'alice_01',
    'al ice',
    '用户名',
    'a'.repeat(21),
  ]) {
    const result = await resolve({ ...validForm, username })
    assert.ok(result.errors.username, username)
    assert.deepEqual(result.values, {})
  }
})

test('password and privacy validation still prevent submission', async () => {
  for (const password of ['', 'x'.repeat(21)]) {
    const result = await resolve({ ...validForm, password })
    assert.ok(result.errors.password)
  }
  const result = await resolve({ ...validForm, acceptPrivacy: false })
  assert.ok(result.errors.acceptPrivacy)
})

test('every locale includes the login rules and validation messages', () => {
  for (const locale of ['zhCN', 'enUS', 'ja', 'ko']) {
    const path = fileURLToPath(
      new URL(`../src/i18n/locales/${locale}/translation.json`, import.meta.url)
    )
    const messages = JSON.parse(readFileSync(path, 'utf8'))
    for (const key of [
      'usernameHint',
      'usernameInvalid',
      'usernameRequired',
      'usernameMaxLength',
      'passwordRequired',
      'passwordMaxLength',
      'privacyRequired',
    ]) {
      assert.equal(typeof messages[`auth.${key}`], 'string')
      assert.ok(messages[`auth.${key}`].length > 0)
    }
  }
})
