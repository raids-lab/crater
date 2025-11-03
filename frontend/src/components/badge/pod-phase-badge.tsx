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
import { PhaseBadge } from './phase-badge'

export const podPhases = [
  {
    value: 'Pending',
    label: '等待中',
    color: 'text-highlight-purple bg-highlight-purple/20',
    description: 'Pod 正在等待被调度或容器镜像正在下载',
  },
  {
    value: 'Running',
    label: '运行中',
    color: 'text-highlight-blue bg-highlight-blue/20',
    description: 'Pod 已被调度到节点上且至少一个容器正在运行',
  },
  {
    value: 'Succeeded',
    label: '已完成',
    color: 'text-highlight-emerald bg-highlight-emerald/20',
    description: 'Pod 中的所有容器已成功终止且不会再启动',
  },
  {
    value: 'Failed',
    label: '失败',
    color: 'text-highlight-red bg-highlight-red/20',
    description: 'Pod 中的所有容器已终止且至少一个容器因错误终止',
  },
  {
    value: 'Unknown',
    label: '未知',
    color: 'text-highlight-slate bg-highlight-slate/20',
    description: '由于某些原因无法获取 Pod 的状态信息',
  },
]

const getPodPhaseLabel = (
  phase: string
): {
  label: string
  color: string
  description: string
} => {
  const foundPhase = podPhases.find((p) => p.value === phase)
  if (foundPhase) {
    return foundPhase
  } else {
    return {
      label: '未知',
      color: 'text-highlight-slate bg-highlight-slate/20',
      description: '由于某些原因无法获取 Pod 的状态信息',
    }
  }
}

const PodPhaseLabel = ({ podPhase }: { podPhase: string }) => {
  return <PhaseBadge phase={podPhase} getPhaseLabel={getPodPhaseLabel} />
}

export default PodPhaseLabel
