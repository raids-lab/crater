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
  RefreshCcw,
} from 'lucide-react'
import { Pause, Play, PlayCircle, Square } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'
import { toast } from 'sonner'
import useResizeObserver from 'use-resize-observer'
import { useCopyToClipboard } from 'usehooks-ts'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'

import { ContainerInfo, apiGetPodContainerLog } from '@/services/api/tool'

import { logger } from '@/utils/loglevel'
import { ACCESS_TOKEN_KEY } from '@/utils/store'

import TooltipButton from '../button/tooltip-button'
import LoadingCircleIcon from '../icon/loading-circle-icon'
import { ButtonGroup } from '../ui-custom/button-group'
import {
  PodContainerDialog,
  PodContainerDialogProps,
  PodNamespacedName,
} from './pod-container-dialog'

const DEFAULT_TAIL_LINES = 500

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
  const [tailLines, setTailLines] = useState(DEFAULT_TAIL_LINES)
  const [timestamps, setTimestamps] = useState(false)
  const [copied, setCopied] = useState(false)
  const [, copy] = useCopyToClipboard()
  const { ref: refRoot, width, height } = useResizeObserver()
  const logAreaRef = useRef<HTMLDivElement>(null)

  const [streaming, setStreaming] = useState(false) // 新增：控制是否处于流式模式
  const [streamingPaused, setStreamingPaused] = useState(false) // 新增：控制流是否暂停
  const [streamedLogs, setStreamedLogs] = useState<string[]>([]) // 新增：存储流式日志
  const streamEventSource = useRef<EventSource | null>(null) // 新增：SSE连接引用
  const streamAbortController = useRef<AbortController | null>(null) // 新增：用于取消 fetch 请求

  const { data: logText, refetch } = useQuery({
    queryKey: [
      'logtext',
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name,
      'log',
      tailLines,
      timestamps,
    ],
    queryFn: () =>
      apiGetPodContainerLog(namespacedName.namespace, namespacedName.name, selectedContainer.name, {
        tailLines: tailLines,
        timestamps: timestamps,
        follow: false,
        previous: false,
      }),
    select: (res) => {
      // 使用改进的方法解码base64字符串为UTF-8文本
      return decodeBase64ToUtf8(res.data)
    },
    enabled:
      !selectedContainer.state.waiting &&
      !!namespacedName.namespace &&
      !!namespacedName.name &&
      !!selectedContainer?.name &&
      !!tailLines,
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
    setTailLines(DEFAULT_TAIL_LINES)
    setCopied(false)
    setTimestamps(false)
    void queryClient.invalidateQueries({ queryKey: ['logtext'] })
  }

  // scroll to bottom when init
  const [showLog, setShowLog] = useState(false)
  useEffect(() => {
    if (tailLines === DEFAULT_TAIL_LINES) {
      logAreaRef.current?.scrollIntoView(false)
    }
    setShowLog(true)
    return () => {
      setShowLog(false)
    }
  }, [logText, tailLines])

  const showMore = useMemo(() => {
    if (logText) {
      const lines = logText.split('\n').length
      return lines > tailLines
    }
    return false
  }, [logText, tailLines])

  const handleShowMore = () => {
    setTailLines((prev) => prev + DEFAULT_TAIL_LINES)
  }

  // 添加启动和停止流式日志的函数
  const startStreaming = async () => {
    if (streamAbortController.current) {
      stopStreaming()
    }

    try {
      const abortController = new AbortController()
      streamAbortController.current = abortController

      // 构建流式日志的 URL
      const baseUrl = `${import.meta.env.VITE_API_BASE || ''}/api/v1/namespaces/${namespacedName.namespace}/pods/${namespacedName.name}/containers/${selectedContainer.name}/log/stream`
      const params = new URLSearchParams({
        timestamps: timestamps.toString(),
      })
      const url = `${baseUrl}?${params}`

      // 获取认证 token
      const token = localStorage.getItem(ACCESS_TOKEN_KEY)

      // 使用原生 fetch 来处理流式响应
      const response = await fetch(url, {
        method: 'GET',
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: 'application/octet-stream',
        },
        signal: abortController.signal,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('无法获取响应流')
      }

      setStreaming(true)

      // 读取流数据
      const decoder = new TextDecoder()
      let buffer = '' // 用于处理不完整的行

      while (true) {
        const { done, value } = await reader.read()

        if (done || abortController.signal.aborted) {
          break
        }

        // 解码数据并添加到缓冲区
        buffer += decoder.decode(value, { stream: true })

        // 按行分割数据
        const lines = buffer.split('\n')
        // 保留最后一个可能不完整的行
        buffer = lines.pop() || ''

        // 处理完整的行
        for (const line of lines) {
          if (line.trim()) {
            try {
              // 后端发送的是 base64 编码的数据，直接解码
              const logLine = decodeBase64ToUtf8(line.trim())
              setStreamedLogs((prev) => [...prev, logLine])

              // 自动滚动到底部（如果没有暂停）
              if (!streamingPaused) {
                setTimeout(() => {
                  logAreaRef.current?.scrollIntoView(false)
                }, 0)
              }
            } catch (error) {
              logger.error('处理流式日志失败:', error, 'Line:', line)
            }
          }
        }
      }

      // 处理最后可能剩余的数据
      if (buffer.trim()) {
        try {
          const logLine = decodeBase64ToUtf8(buffer.trim())
          setStreamedLogs((prev) => [...prev, logLine])
        } catch (error) {
          logger.error('处理最后一行日志失败:', error)
        }
      }
    } catch (error) {
      if (error instanceof Error && error.name !== 'AbortError') {
        logger.error('流式日志连接失败:', error)
        toast.error('流式日志连接失败，已自动切换到普通模式')
        refetch()
      }
    } finally {
      setStreaming(false)
      streamAbortController.current = null
    }
  }

  const stopStreaming = () => {
    if (streamAbortController.current) {
      streamAbortController.current.abort()
      streamAbortController.current = null
    }
    if (streamEventSource.current) {
      streamEventSource.current.close()
      streamEventSource.current = null
    }
    setStreaming(false)
    setStreamedLogs([]) // 清空之前的日志
    setStreamingPaused(false)
  }

  // 组件卸载时清理资源
  useEffect(() => {
    return () => {
      stopStreaming()
    }
  }, [])

  // 确保流模式和普通模式切换时清理/重置
  useEffect(() => {
    if (streaming) {
      // 切换到流模式时清空缓存的流日志
      setStreamedLogs([])
    }
  }, [streaming])

  return (
    <Card
      className="dark:bg-muted/30 bg-sidebar relative h-full overflow-hidden rounded-md p-1 md:col-span-2 xl:col-span-3 dark:border"
      ref={refRoot}
    >
      {showLog ? (
        <>
          <ScrollArea style={{ width, height }}>
            <div ref={logAreaRef}>
              {showMore && (
                <div className="flex justify-center pt-5 select-none">
                  <Button size="sm" onClick={handleShowMore}>
                    显示更多
                  </Button>
                </div>
              )}
              <pre className="px-3 py-3 text-sm break-words whitespace-pre-wrap dark:text-blue-300">
                {streaming ? streamedLogs.join('') : logText}
              </pre>
            </div>
            <ScrollBar orientation="horizontal" />
          </ScrollArea>
          <ButtonGroup className="border-input bg-background text-foreground absolute top-5 right-5 rounded-md border">
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
              onClick={() => setTimestamps((prev) => !prev)}
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
              onClick={() => setStreamingPaused(!streamingPaused)}
              className="hover:text-primary border-0 border-r focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent={streamingPaused ? '继续自动滚动' : '暂停自动滚动'}
              hidden={!streaming}
            >
              {streamingPaused ? <Play className="size-4" /> : <Pause className="size-4" />}
            </TooltipButton>
            <TooltipButton
              onClick={() => (streaming ? stopStreaming() : startStreaming())}
              className="hover:text-primary border-0 focus-visible:ring-0"
              variant="ghost"
              size="icon"
              tooltipContent={streaming ? '停止实时日志' : '启动实时日志'}
            >
              {streaming ? (
                <Square className="size-4" /> // 停止图标
              ) : (
                <PlayCircle className="size-4" /> // 播放图标
              )}
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
