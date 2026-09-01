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

interface EnvironmentVariable {
  name: string
  value: string
}

export const getDefaultTensorboardLogDir = (workspacePath: string, jobName: string) =>
  `${workspacePath.replace(/\/+$/, '')}/tensorboard-runs/${jobName}`

export const withTensorboardLogDirEnv = (
  envs: EnvironmentVariable[],
  logDir: string
): EnvironmentVariable[] => [
  ...envs.filter((env) => env.name !== TensorboardLogDirEnv),
  { name: TensorboardLogDirEnv, value: logDir },
]
