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

import { PhaseBadge } from './PhaseBadge'

// 数据集类型枚举
export enum DatasetType {
  Model = 'model',
  Dataset = 'dataset',
  ShareFile = 'sharefile',
}

export const datasetTypes = [
  {
    value: 'model',
    label: '模型',
  },
  {
    value: 'dataset',
    label: '数据集',
  },
  {
    value: 'sharefile',
    label: '共享文件',
  },
]

const getDatasetTypeLabel = (
  phase: DatasetType
): {
  label: string
  color: string
  description: string
} => {
  switch (phase) {
    case DatasetType.Model:
      return {
        label: '模型',
        color: 'text-highlight-blue bg-highlight-blue/10',
        description: '模型',
      }
    case DatasetType.Dataset:
      return {
        label: '数据集',
        color: 'text-highlight-green bg-highlight-green/10',
        description: '数据集',
      }
    case DatasetType.ShareFile:
      return {
        label: '共享文件',
        color: 'text-highlight-orange bg-highlight-orange/10',
        description: '共享文件',
      }
    default:
      return {
        label: '数据集',
        color: 'text-highlight-green bg-highlight-green/10',
        description: '数据集',
      }
  }
}

const DatasetTypeLabel = ({ datasetType }: { datasetType: DatasetType }) => {
  return <PhaseBadge phase={datasetType} getPhaseLabel={getDatasetTypeLabel} />
}

export default DatasetTypeLabel
