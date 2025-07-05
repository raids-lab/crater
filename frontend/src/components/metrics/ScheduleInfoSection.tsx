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

import { MetricSection } from '../ui-custom/metric-section'
import { MetricCard } from '../ui-custom/metric-card'
import { ClockIcon } from 'lucide-react'
import { IJupyterDetail } from '@/services/api/vcjob'
import { Fragment, useMemo } from 'react'
import { formatDistanceStrict, formatDuration, intervalToDuration } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { TerminatedSection } from './TerminatedSection'

const parseDurationString = (durationStr: string): number => {
  if (!durationStr) return 0

  let totalSeconds = 0

  // 处理小时部分 - 必须跟着h且前面没有m或s
  const hoursMatch = durationStr.match(/(\d+)h/)
  if (hoursMatch) {
    totalSeconds += parseInt(hoursMatch[1], 10) * 3600
  }

  // 处理分钟部分 - 必须跟着m且后面不是s
  const minutesMatch = durationStr.match(/(\d+)m(?!s)/)
  if (minutesMatch) {
    totalSeconds += parseInt(minutesMatch[1], 10) * 60
  }

  // 处理秒部分 - 必须跟着s且前面不是m
  const secondsMatch = durationStr.match(/(\d+\.?\d*)(?<!m)s/)
  if (secondsMatch) {
    totalSeconds += parseFloat(secondsMatch[1])
  }

  // 处理毫秒部分 - 必须跟着ms
  const millisecondsMatch = durationStr.match(/(\d+)ms/)
  if (millisecondsMatch) {
    totalSeconds += parseInt(millisecondsMatch[1], 10) / 1000
  }

  return totalSeconds
}

// 辅助函数：将秒数转换为易读的持续时间字符串
const formatSecondsToDuration = (seconds: number): string => {
  const duration = intervalToDuration({ start: 0, end: seconds * 1000 })

  return formatDuration(duration, {
    locale: zhCN,
    format: ['hours', 'minutes', 'seconds'],
    zero: false,
    delimiter: ' ',
  })
}

// 新增组件：作业调度信息区域
interface ScheduleInfoSectionProps {
  data: IJupyterDetail
}

export const JobInfoSections = ({
  data: { terminatedStates, ...props },
}: ScheduleInfoSectionProps) => {
  const scheduleInfo = useMemo(() => {
    const createdAt = props.createdAt ? new Date(props.createdAt) : null
    const startedAt = props.startedAt ? new Date(props.startedAt) : null
    const finishedAt = props.completedAt ? new Date(props.completedAt) : null
    const imagePullTime = props.scheduleData?.imagePullTime || ''

    const updateTimeDiff = (later: Date, earlier: Date) => {
      const timeDifference = formatDistanceStrict(later, earlier, {
        locale: zhCN,
        addSuffix: false,
      })
      return timeDifference
    }

    if (createdAt && startedAt && finishedAt) {
      // 运行时间（开始到结束）
      const runTime = updateTimeDiff(finishedAt, startedAt)

      // 总等待时间（创建到开始）
      const totalWaitTime = updateTimeDiff(startedAt, createdAt)

      // 镜像拉取时间（从字符串解析）
      const imagePullSeconds = parseDurationString(imagePullTime)
      const pullTime = formatSecondsToDuration(imagePullSeconds)

      // 排队时间（总等待时间 - 镜像拉取时间）
      const totalWaitSeconds = (startedAt.getTime() - createdAt.getTime()) / 1000
      const queueSeconds = Math.max(0, totalWaitSeconds - imagePullSeconds)
      const queueTime = formatSecondsToDuration(queueSeconds)

      return {
        runTime,
        pullTime,
        waitTime: queueTime,
        totalWaitTime, // 如果需要也可以返回总等待时间
      }
    }
    return {
      runTime: undefined,
      pullTime: undefined,
      waitTime: undefined,
    }
  }, [props.completedAt, props.createdAt, props.startedAt, props.scheduleData])

  return (
    <>
      {terminatedStates &&
        terminatedStates.length > 0 &&
        terminatedStates.map((state, index) => (
          <Fragment key={index}>
            <TerminatedSection state={state} index={index} />
          </Fragment>
        ))}
      {(scheduleInfo.waitTime || scheduleInfo.pullTime || scheduleInfo.runTime) && (
        <MetricSection title="作业调度信息" icon={<ClockIcon className="h-5 w-5" />}>
          {scheduleInfo.waitTime && (
            <MetricCard
              title="作业排队用时"
              value={scheduleInfo.waitTime}
              unit=""
              description="作业从创建到被调度到节点的时间"
            />
          )}
          {!!scheduleInfo.pullTime && (
            <MetricCard
              title="镜像拉取用时"
              value={scheduleInfo.pullTime}
              unit=""
              description="镜像拉取所花费的时间，为空说明镜像已缓存"
            />
          )}
          {scheduleInfo.runTime && (
            <MetricCard
              title="作业运行用时"
              value={scheduleInfo.runTime}
              unit=""
              description="作业执行的总时长"
            />
          )}
        </MetricSection>
      )}
    </>
  )
}
