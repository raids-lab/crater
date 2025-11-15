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
import { ModelDownload } from '@/services/api/modeldownload'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export const getModelDownloadStatusLabel = (status: ModelDownload['status']): PhaseBadgeData => {
  switch (status) {
    case 'Pending':
      return {
        label: '等待中',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '模型下载等待中',
      }
    case 'Downloading':
      return {
        label: '下载中',
        color: 'text-highlight-sky bg-highlight-sky/20',
        description: '模型正在下载',
      }
    case 'Paused':
      return {
        label: '已暂停',
        color: 'text-amber-600 bg-amber-600/20',
        description: '模型下载已暂停',
      }
    case 'Ready':
      return {
        label: '完成',
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: '模型下载完成',
      }
    case 'Failed':
      return {
        label: '失败',
        color: 'text-highlight-red bg-highlight-red/20',
        description: '模型下载失败',
      }
    default:
      return {
        label: '未知',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '未知状态',
      }
  }
}

const ModelDownloadPhaseBadge = ({ status }: { status: ModelDownload['status'] }) => {
  return <PhaseBadge phase={status} getPhaseLabel={getModelDownloadStatusLabel} />
}

export default ModelDownloadPhaseBadge
