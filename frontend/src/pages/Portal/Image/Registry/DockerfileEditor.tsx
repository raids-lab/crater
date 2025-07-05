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

// TODO: https://github.com/suren-atoyan/monaco-react?tab=readme-ov-file#use-monaco-editor-as-an-npm-package
import { useTheme } from '@/utils/theme'
import { Editor } from '@monaco-editor/react'

interface DockerfileEditorProps {
  value: string
  onChange: (value: string | undefined) => void
  language?: string
}

export function DockerfileEditor({
  value,
  onChange,
  language = 'dockerfile',
}: DockerfileEditorProps) {
  const { theme } = useTheme()
  return (
    <Editor
      height="300px"
      defaultLanguage={language}
      value={value}
      onChange={onChange}
      theme={theme === 'light' ? 'light' : 'vs-dark'}
      options={{
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        fontSize: 13,
        lineNumbers: 'on',
        renderLineHighlight: 'all',
        tabSize: 2,
      }}
      className="bg-transparen border-input overflow-hidden rounded-md border"
    />
  )
}
