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
import { t } from 'i18next'
import {
  Ban,
  Check,
  CircleDashed,
  CircleDot,
  Clock,
  Hourglass,
  Loader2,
  Play,
  RotateCcw,
  Square,
  X,
} from 'lucide-react'

import {
  JobDisplayPhase,
  JobDisplayPhaseValue,
  JobPhase,
  PodGroupPhase,
  getJobDisplayPhase,
} from '@/services/api/vcjob'

import { PhaseBadge, PhaseBadgeData } from './phase-badge'

export const jobPhases = [
  {
    value: 'Prequeue',
    label: t('jobs.statuses.prequeue.label'),
    color: '#8b5cf6',
  },
  {
    value: 'Pending',
    label: t('jobs.statuses.pending.label'),
    color: '#a855f7',
  },
  {
    value: 'Aborting',
    label: '即将中止',
    color: '#ec4899',
  },
  {
    value: 'Aborted',
    label: '已中止',
    color: '#ec4899',
  },
  {
    value: 'Running',
    label: '运行中',
    color: '#3b82f6',
  },
  {
    value: 'Restarting',
    label: '重启中',
    color: '#f43f5e',
  },
  {
    value: 'Completing',
    label: '即将完成',
    color: '#10b981',
  },
  {
    value: 'Completed',
    label: '已完成',
    color: '#10b981',
  },
  {
    value: 'Terminating',
    label: '即将终止',
    color: '#f97316',
  },
  {
    value: 'Terminated',
    label: '已终止',
    color: '#f97316',
  },
  {
    value: 'Deleted',
    label: '已停止',
    color: '#f97316',
  },
  {
    value: 'Freed',
    label: '已释放',
    color: '#f97316',
  },
  {
    value: 'Failed',
    label: '失败',
    color: '#ef4444',
  },
]

export const getJobPhaseLabel = (phase: JobDisplayPhaseValue): PhaseBadgeData => {
  switch (phase) {
    case JobPhase.Prequeue:
      return {
        label: t('jobs.statuses.prequeue.label'),
        color: 'text-highlight-violet bg-highlight-violet/20',
        description: t('jobs.statuses.prequeue.description'),
        icon: <Clock />,
      }
    case JobPhase.Pending:
      return {
        label: t('jobs.statuses.pending.label'),
        color: 'text-highlight-purple bg-highlight-purple/20',
        description: t('jobs.statuses.pending.description'),
        icon: <Hourglass />,
      }
    case JobDisplayPhase.Inqueue:
      return {
        label: t('jobs.statuses.inqueue.label'),
        color: 'text-highlight-indigo bg-highlight-indigo/20',
        description: t('jobs.statuses.inqueue.description'),
        icon: <CircleDashed />,
      }
    case JobDisplayPhase.Starting:
      return {
        label: t('jobs.statuses.starting.label'),
        color: 'text-highlight-sky bg-highlight-sky/20',
        description: t('jobs.statuses.starting.description'),
        icon: <Loader2 />,
      }
    case JobPhase.Init:
      return {
        label: '已创建',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '作业已提交到集群，等待信息同步',
        icon: <CircleDot />,
      }
    case JobPhase.Aborting:
      return {
        label: '即将中止',
        color: 'text-highlight-pink bg-highlight-pink/20',
        description: '作业即将被中止',
        icon: <Ban />,
      }
    case JobPhase.Aborted:
      return {
        label: '已中止',
        color: 'text-highlight-pink bg-highlight-pink/20',
        description: '作业已被中止',
        icon: <Ban />,
      }
    case JobPhase.Running:
      return {
        label: '运行中',
        color: 'text-highlight-blue bg-highlight-blue/20',
        description: '作业正在运行',
        icon: <Play className="fill-current" />,
      }
    case JobPhase.Restarting:
      return {
        label: '重启中',
        color: 'text-highlight-rose bg-highlight-rose/20',
        description: '作业正在重启',
        icon: <RotateCcw />,
      }
    case JobPhase.Completing:
      return {
        label: '即将完成',
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: '作业即将完成',
        icon: <Check />,
      }
    case JobPhase.Completed:
      return {
        label: '已完成',
        color: 'text-highlight-emerald bg-highlight-emerald/20',
        description: '作业已完成',
        icon: <Check />,
      }
    case JobPhase.Terminating:
      return {
        label: '即将终止',
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: '作业即将终止',
        icon: <Square />,
      }
    case JobPhase.Terminated:
      return {
        label: '已终止',
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: '作业已终止',
        icon: <Square />,
      }
    case JobPhase.Failed:
      return {
        label: '失败',
        color: 'text-highlight-red bg-highlight-red/20',
        description: '作业失败',
        icon: <X />,
      }
    case JobPhase.Deleted:
      return {
        label: '已停止',
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: '作业被手动停止，计算资源已释放，作业元数据仍保留',
        icon: <Square />,
      }
    case JobPhase.Freed:
      return {
        label: '已释放',
        color: 'text-highlight-orange bg-highlight-orange/20',
        description: '作业长期占用计算资源但未使用，已自动释放',
        icon: <Square />,
      }
    default:
      return {
        label: '未知',
        color: 'text-highlight-slate bg-highlight-slate/20',
        description: '作业状态未知',
        icon: <CircleDot />,
      }
  }
}

const JobPhaseLabel = ({
  jobPhase,
  podGroupPhase,
}: {
  jobPhase: JobPhase
  podGroupPhase?: PodGroupPhase
}) => {
  return (
    <PhaseBadge
      phase={getJobDisplayPhase(jobPhase, podGroupPhase)}
      getPhaseLabel={getJobPhaseLabel}
    />
  )
}

export default JobPhaseLabel
