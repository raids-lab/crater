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

// i18n-processed-v1.1.0 (no translatable strings)
import { ImagePackSource } from '@/services/api/imagepack'

export interface MetadataFormType {
  version: string
  type: string
}

export const MetadataFormAccount: MetadataFormType = {
  version: '20241208',
  type: 'account',
}

export const MetadataFormJupyter: MetadataFormType = {
  version: '20250420',
  type: 'jupyter',
}

export const MetadataFormCustom: MetadataFormType = {
  version: '20250317',
  type: 'custom',
}

export const MetadataFormCustomEmias: MetadataFormType = {
  version: '20250420',
  type: 'custom-emias',
}

export const MetadataFormTensorflow: MetadataFormType = {
  version: '20240528',
  type: 'tensorflow',
}

export const MetadataFormPytorch: MetadataFormType = {
  version: '20240528',
  type: 'pytorch',
}

// 基于Dockerfile构建
export const MetadataFormDockerfile: MetadataFormType = {
  version: '20250506',
  type: ImagePackSource.Dockerfile,
}

// 基于现有镜像构建
export const MetadataFormPipApt: MetadataFormType = {
  version: '20250506',
  type: ImagePackSource.PipApt,
}

// Python+CUDA自定义构建
export const MetadataFormEnvdAdvanced: MetadataFormType = {
  version: '20250506',
  type: ImagePackSource.EnvdAdvanced,
}

// 基于Envd构建
export const MetadataFormEnvdRaw: MetadataFormType = {
  version: '20250506',
  type: ImagePackSource.EnvdRaw,
}

export const MetadataFormJupyterEmias: MetadataFormType = {
  version: '20240528',
  type: 'jupyter-emias',
}
