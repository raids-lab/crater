/**
 * Copyright 2025 RAIDS Lab
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

import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import tseslint from 'typescript-eslint'
import pluginQuery from '@tanstack/eslint-plugin-query'
import eslintPluginPrettier from 'eslint-plugin-prettier/recommended'

export default tseslint.config({
  extends: [
    js.configs.recommended,
    ...tseslint.configs.recommended,
    ...pluginQuery.configs['flat/recommended'],
    eslintPluginPrettier,
  ],
  files: ['**/*.{ts,tsx}'],
  ignores: [
    '**/dist',
    '**/postcss.config.js',
    '**/tailwind.config.js',
    '**/vite.config.ts',
    '**/commitlint.config.cjs',
    // shadcn
    'src/components/ui/*',
    'src/components/ui-custom/*',
    'src/hooks/use-mobile.tsx',
    // msw
    'public/mockServiceWorker.js',
  ],
  languageOptions: {
    ecmaVersion: 2020,
    globals: globals.browser,
  },
  plugins: {
    'react-hooks': reactHooks,
    'react-refresh': reactRefresh,
  },
  rules: {
    ...reactHooks.configs.recommended.rules,
    'react-refresh/only-export-components': [
      'off', // todo(liyilong): change to warn
      { allowConstantExport: true },
    ],
    'no-console': 'error',
  },
})
