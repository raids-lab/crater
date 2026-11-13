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

export const TensorboardLogDirEnv = 'TENSORBOARD_LOGDIR'
export const TensorboardJobNamePlaceholder = '__CRATER_JOB_NAME__'

// Only personal panels share the storage API's logical "user" root.
export const getTensorboardPersonalStoragePath = (logDir: string, username?: string) => {
  if (!username || /[\\/]/.test(username)) return null
  const trimmed = logDir.trim()
  if (!trimmed.startsWith('/') || trimmed.includes('\\')) return null
  const segments = trimmed.split('/').filter(Boolean)
  if (segments.some((segment) => segment === '.' || segment === '..')) return null
  const cleanLogDir = `/${segments.join('/')}`
  const personalRoot = `/home/${username}`
  if (cleanLogDir === personalRoot) return 'user'
  if (!cleanLogDir.startsWith(`${personalRoot}/`)) return null
  return `user/${cleanLogDir.slice(personalRoot.length + 1)}`
}

interface EnvironmentVariable {
  name: string
  value: string
}

export const getDefaultTensorboardLogDir = (
  workspacePath: string,
  jobName = TensorboardJobNamePlaceholder
) => `${workspacePath.replace(/\/+$/, '')}/tensorboard-runs/${jobName}`

export const withTensorboardLogDirEnv = (
  envs: EnvironmentVariable[],
  logDir: string
): EnvironmentVariable[] => [
  ...envs.filter((env) => env.name !== TensorboardLogDirEnv),
  { name: TensorboardLogDirEnv, value: logDir },
]
