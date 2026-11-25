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
import { JobDisplayPhase } from '@/services/api/vcjob'

import { getJobPhaseLabel } from './job-phase-badge'
import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export const podPhases = [
  {
    value: 'Pending',
    label: '等待中',
    color: 'text-highlight-purple bg-highlight-purple/20',
    description: 'Pod 尚未分配节点，正在等待调度',
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

const getPodPhaseLabel = (phase: string): PhaseBadgeData => {
  if (phase === JobDisplayPhase.Starting) {
    // Pod badges carry no icon; take only the copy and color so both badges read the same.
    const { label, color, description } = getJobPhaseLabel(JobDisplayPhase.Starting)
    return { label, color, description }
  }
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

// A bound pod stays Pending until its containers start; display-only, so filters keep matching the raw phase.
const getPodDisplayPhase = (phase: string, nodeName?: string): string =>
  phase === 'Pending' && nodeName ? JobDisplayPhase.Starting : phase

const PodPhaseLabel = ({ podPhase, nodeName }: { podPhase: string; nodeName?: string }) => {
  return (
    <PhaseBadge phase={getPodDisplayPhase(podPhase, nodeName)} getPhaseLabel={getPodPhaseLabel} />
  )
}

export default PodPhaseLabel
