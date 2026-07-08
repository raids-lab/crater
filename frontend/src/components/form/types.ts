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
// i18n-processed-v1.1.0
import { ImagePackSource } from '@/services/api/imagepack'

export interface MetadataFormMigration {
  to: string
  migrate: (data: unknown) => unknown
}

export interface MetadataFormType {
  version: string
  type: string
  migrations?: Record<string, MetadataFormMigration>
}

const CurrentJobTemplateVersion = '20260707'

const NodeSelectorMode = {
  Include: 'include',
  Exclude: 'exclude',
} as const

const migrateToCurrentNodeSelector = {
  to: CurrentJobTemplateVersion,
  migrate: (data: unknown) => migrateRootNodeSelector(data),
}

const withNodeSelectorMigration = (fromVersion: string): Record<string, MetadataFormMigration> => ({
  [fromVersion]: migrateToCurrentNodeSelector,
})

const isRecord = (value: unknown): value is Record<string, unknown> => {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

const uniqueStrings = (values: unknown): string[] => {
  if (!Array.isArray(values)) {
    return []
  }

  return Array.from(
    new Set(
      values
        .filter((value): value is string => typeof value === 'string')
        .map((value) => value.trim())
    )
  ).filter(Boolean)
}

const migrateRootNodeSelector = (data: unknown): unknown => {
  if (!isRecord(data)) {
    throw new Error('模板数据格式无效，无法迁移节点控制配置')
  }

  const nodeSelector = isRecord(data.nodeSelector) ? data.nodeSelector : undefined

  return {
    ...data,
    nodeSelector: migrateNodeSelector(nodeSelector),
  }
}

const migrateNodeSelector = (nodeSelector?: Record<string, unknown>) => {
  if (!nodeSelector) {
    return {
      enable: false,
      mode: NodeSelectorMode.Include,
      nodes: [],
    }
  }

  const nodes = uniqueStrings(nodeSelector.nodes)
  if (nodes.length > 0) {
    return {
      enable: nodeSelector.enable === true,
      mode:
        nodeSelector.mode === NodeSelectorMode.Exclude
          ? NodeSelectorMode.Exclude
          : NodeSelectorMode.Include,
      nodes,
    }
  }

  const nodeName = typeof nodeSelector.nodeName === 'string' ? nodeSelector.nodeName.trim() : ''
  const excludedNodes = uniqueStrings(nodeSelector.excludedNodes)

  if (nodeSelector.enable === true && nodeName) {
    return {
      enable: true,
      mode: NodeSelectorMode.Include,
      nodes: [nodeName],
    }
  }

  if (excludedNodes.length > 0) {
    return {
      enable: true,
      mode: NodeSelectorMode.Exclude,
      nodes: excludedNodes,
    }
  }

  return {
    enable: false,
    mode: NodeSelectorMode.Include,
    nodes: [],
  }
}

export const MetadataFormAccount: MetadataFormType = {
  version: '20241208',
  type: 'account',
}

export const MetadataFormJupyter: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'jupyter',
  migrations: withNodeSelectorMigration('20250420'),
}

export const MetadataFormWebIDE: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'webide',
  migrations: withNodeSelectorMigration('20251126'),
}

export const MetadataFormCustom: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'custom',
  migrations: withNodeSelectorMigration('20250317'),
}

export const MetadataFormCustomEmias: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'custom-emias',
  migrations: withNodeSelectorMigration('20250420'),
}

export const MetadataFormTensorflow: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'tensorflow',
  migrations: withNodeSelectorMigration('20240528'),
}

export const MetadataFormPytorch: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'pytorch',
  migrations: withNodeSelectorMigration('20240528'),
}

export const MetadataFormSingle: MetadataFormType = {
  version: CurrentJobTemplateVersion,
  type: 'single',
  migrations: withNodeSelectorMigration('20240528'),
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
  version: CurrentJobTemplateVersion,
  type: 'jupyter-emias',
  migrations: withNodeSelectorMigration('20240528'),
}
