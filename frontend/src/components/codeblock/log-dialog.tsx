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
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  CalendarArrowDown,
  CalendarOff,
  CopyCheckIcon,
  CopyIcon,
  DownloadIcon,
  ExternalLink,
  RefreshCcw,
} from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import { toast } from 'sonner'
import { useResizeObserver } from 'usehooks-ts'
import { useCopyToClipboard } from 'usehooks-ts'

import { Card } from '@/components/ui/card'

import { ContainerInfo, apiGetPodContainerLog } from '@/services/api/tool'

import { logger } from '@/utils/loglevel'

import TooltipButton from '../button/tooltip-button'
import LoadingCircleIcon from '../icon/loading-circle-icon'
import { ButtonGroup } from '../ui-custom/button-group'
import {
  PodContainerDialog,
  PodContainerDialogProps,
  PodNamespacedName,
} from './pod-container-dialog'
import SimpleLogViewer from './simple-log-viewer'

// 辅助函数：正确解码包含UTF-8字符的base64字符串
const decodeBase64ToUtf8 = (base64: string): string => {
  try {
    // 将base64转换为二进制字符串
    const binaryString = atob(base64)
    // 创建一个Uint8Array来存储二进制数据
    const bytes = new Uint8Array(binaryString.length)
    // 将二进制字符串的每个字符转换为其对应的字节值
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    // 使用TextDecoder将字节数组解码为UTF-8字符串
    return new TextDecoder('utf-8').decode(bytes)
  } catch (error) {
    logger.error('Base64解码失败:', error)
    return '日志解码失败'
  }
}

export function LogCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName
  selectedContainer: ContainerInfo
}) {
  const queryClient = useQueryClient()
  const [timestamps, setTimestamps] = useState(false)
  const [copied, setCopied] = useState(false)
  const [, copy] = useCopyToClipboard()
  const refRoot = useRef<HTMLDivElement | null>(null)
  // usehooks-ts expects a RefObject<HTMLElement>, cast is safe for HTMLDivElement
  const { width = 0, height = 0 } = useResizeObserver({
    ref: refRoot as React.RefObject<HTMLElement>,
  })

  // Fetch static logs - no tailLines limit, get all logs
  const { data: logText } = useQuery({
    queryKey: [
      'logtext',
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name,
      'log',
      timestamps,
    ],
    queryFn: () =>
      apiGetPodContainerLog(namespacedName.namespace, namespacedName.name, selectedContainer.name, {
        timestamps: timestamps,
        follow: false,
        previous: false,
        // No tailLines - get all logs
      }),
    select: (res) => {
      // 使用改进的方法解码base64字符串为UTF-8文本
      return decodeBase64ToUtf8(res.data)
    },
    enabled:
      !selectedContainer.state.waiting &&
      !!namespacedName.namespace &&
      !!namespacedName.name &&
      !!selectedContainer?.name,
  })

  const { mutate: downloadAllLog } = useMutation({
    mutationFn: () =>
      apiGetPodContainerLog(namespacedName.namespace, namespacedName.name, selectedContainer.name, {
        timestamps: timestamps,
        follow: false,
        previous: false,
      }),
    onSuccess: (res) => {
      // 使用改进的方法解码base64字符串为UTF-8文本
      const logText = decodeBase64ToUtf8(res.data)
      const blob = new Blob([logText], {
        type: 'text/plain;charset=utf-8',
      })
      const filename = selectedContainer?.name ?? 'log'
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = filename
      a.click()
    },
  })

  const copyCode = () => {
    if (!logText) {
      toast.warning('没有日志可供复制')
      return
    }
    copy(logText)
      .then(() => {
        setCopied(true)
        toast.success('已复制到剪贴板')
      })
      .catch(() => {
        toast.error('复制失败')
      })
  }

  const handleDownload = () => {
    if (!logText) {
      toast.warning('没有日志可供下载')
      return
    }
    downloadAllLog()
  }

  const handleRefresh = () => {
    setCopied(false)
    void queryClient.invalidateQueries({ queryKey: ['logtext'] })
  }

  const openFullscreenLog = () => {
    const url = `/logs/fullscreen?namespace=${namespacedName.namespace}&pod=${namespacedName.name}&container=${selectedContainer.name}&timestamps=${timestamps}`
    window.open(url, '_blank', 'noopener,noreferrer')
  }

  const [showLog, setShowLog] = useState(false)
  useEffect(() => {
    setShowLog(true)
    return () => {
      setShowLog(false)
    }
  }, [])

  return (
    <Card
      className="dark:bg-muted/30 bg-sidebar relative h-full overflow-hidden rounded-md p-1 md:col-span-2 xl:col-span-3 dark:border"
      ref={refRoot}
    >
      {showLog ? (
        <>
          <div style={{ width, height }}>
            <SimpleLogViewer logText={logText} height={height} followLog={true} />
          </div>
          <ButtonGroup className="border-input bg-background text-foreground absolute top-5 right-5 z-10 rounded-md border">
            <TooltipButton
              tooltipContent="刷新"
              onClick={handleRefresh}
              className="hover:text-primary border-0 border-r focus-visible:ring-0"
              variant="ghost"
              size="icon"
              title="刷新"
            >
              <RefreshCcw className="size-4" />
            </TooltipButton>
            <TooltipButton
              onClick={() => {
                setTimestamps((prev) => !prev)
                // Re-fetch logs with new timestamp setting
                void queryClient.invalidateQueries({ queryKey: ['logtext'] })
              }}
              className="hover:text-primary border-0 border-r focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent={timestamps ? '隐藏时间戳' : '显示时间戳'}
            >
              {timestamps ? (
                <CalendarOff className="size-4" />
              ) : (
                <CalendarArrowDown className="size-4" />
              )}
            </TooltipButton>
            <TooltipButton
              onClick={copyCode}
              className="hover:text-primary border-0 border-r focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent="复制"
            >
              {copied ? <CopyCheckIcon className="size-4" /> : <CopyIcon className="size-4" />}
            </TooltipButton>
            <TooltipButton
              onClick={handleDownload}
              className="hover:text-primary border-0 border-r focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent="下载"
            >
              <DownloadIcon className="size-4" />
            </TooltipButton>
            <TooltipButton
              onClick={openFullscreenLog}
              className="hover:text-primary border-0 focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent="在新标签页打开实时日志"
            >
              <ExternalLink className="size-4" />
            </TooltipButton>
          </ButtonGroup>
        </>
      ) : (
        <LoadingCircleIcon />
      )}
    </Card>
  )
}

export default function LogDialog({ namespacedName, setNamespacedName }: PodContainerDialogProps) {
  return (
    <PodContainerDialog
      namespacedName={namespacedName}
      setNamespacedName={setNamespacedName}
      ActionComponent={LogCard}
      type="log"
    />
  )
}
