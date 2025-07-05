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
import { BugIcon } from 'lucide-react'
import { TerminatedState } from '@/services/api/tool'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { ReactNode } from 'react'

// 退出码对照表
const EXIT_CODE_MAP: Record<number, { name: string; suggestion: ReactNode }> = {
  0: {
    name: '正常退出',
    suggestion: '容器正常结束运行，无需特别处理。',
  },
  1: {
    name: '应用错误',
    suggestion: <>容器因应用程序错误而停止，请检查「基本信息」中的日志内容。</>,
  },
  125: {
    name: '容器未能运行',
    suggestion: 'docker run 命令未能成功执行，请检查容器运行环境和配置。',
  },
  126: {
    name: '命令调用错误',
    suggestion: '无法调用镜像中指定的命令，请确认命令路径是否正确及是否有执行权限。',
  },
  127: {
    name: '找不到文件或目录',
    suggestion: '找不到镜像中指定的文件或目录，请检查文件路径是否正确。',
  },
  128: {
    name: '退出时使用的参数无效',
    suggestion: '退出是用无效的退出码触发的，请检查应用的退出处理逻辑。',
  },
  134: {
    name: '异常终止 (SIGABRT)',
    suggestion:
      '容器使用 abort() 函数自行中止，通常由程序内部逻辑触发，请检查应用代码中的异常处理。',
  },
  137: {
    name: '立即终止 (SIGKILL)',
    suggestion:
      '容器被操作系统通过 SIGKILL 信号终止，通常由于内存不足，请尝试增加 Memory 资源申请量。',
  },
  139: {
    name: '分段错误 (SIGSEGV)',
    suggestion: '容器试图访问未分配给它的内存并被终止，请检查应用代码中的内存访问逻辑。',
  },
  143: {
    name: '优雅终止 (SIGTERM)',
    suggestion: '容器收到优雅终止信号后正常关闭，这通常是由外部请求引起的，可能是正常的调度行为。',
  },
  255: {
    name: '退出状态超出范围',
    suggestion:
      '容器退出，返回可接受范围之外的退出代码，错误原因未知，请检查容器日志获取更多信息。',
  },
}

// 获取退出码信息
const getExitCodeInfo = (exitCode: number) => {
  return (
    EXIT_CODE_MAP[exitCode] || {
      name: '未知错误',
      suggestion: '未知的退出码，请检查作业日志获取更多信息。',
    }
  )
}

export const TerminatedSection = ({ state, index }: { state: TerminatedState; index: number }) => {
  const exitCodeInfo =
    state.exitCode !== undefined ? getExitCodeInfo(state.exitCode) : { name: '', suggestion: '' }

  return (
    <MetricSection title={`作业诊断信息 #${index + 1}`} icon={<BugIcon className="h-5 w-5" />}>
      <Card className="border">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">退出原因</CardTitle>
          <p className="text-muted-foreground text-xs">
            {state.exitCode === 0 ? '正常退出' : '异常退出'}的原因
          </p>
        </CardHeader>
        <CardContent>
          <div className="text-destructive font-mono text-2xl font-bold">
            {state.reason || 'Unknown'}
            <span className="ml-0.5 text-xl"></span>
          </div>
        </CardContent>
      </Card>
      {state.exitCode !== undefined && (
        <Card className="border">
          <CardHeader className="pb-2">
            <CardTitle className="text-sm font-medium">退出代码</CardTitle>
            <p className="text-muted-foreground text-xs">{exitCodeInfo.name || '未知错误'}</p>
          </CardHeader>
          <CardContent>
            <div className="text-destructive font-mono text-2xl font-bold">
              {state.exitCode}
              <span className="ml-0.5 text-xl"></span>
            </div>
          </CardContent>
        </Card>
      )}
      <Card className="justify-between border">
        <CardHeader className="pb-2">
          <CardTitle className="text-sm font-medium">问题建议</CardTitle>
        </CardHeader>
        <CardContent className="text-muted-foreground text-sm text-balance">
          {state.exitCode !== undefined
            ? exitCodeInfo.suggestion
            : '未能获取退出代码，请检查作业日志获取更多信息。'}
        </CardContent>
      </Card>
    </MetricSection>
  )
}
