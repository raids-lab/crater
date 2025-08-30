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
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { useAtomValue } from 'jotai'
import { useEffect, useRef } from 'react'
import { useDebounceCallback } from 'usehooks-ts'

import { Card } from '@/components/ui/card'

import { ContainerInfo } from '@/services/api/tool'

import { REFRESH_TOKEN_KEY } from '@/utils/store'
import { configAPIPrefixAtom } from '@/utils/store/config'

import {
  PodContainerDialog,
  PodContainerDialogProps,
  PodNamespacedName,
} from './PodContainerDialog'

const buildWebSocketUrl = (
  apiPrefix: string,
  namespace: string,
  podName: string,
  containerName: string
) => {
  const protocol = window.location.protocol === 'https:' ? 'wss' : 'ws'
  const token = localStorage.getItem(REFRESH_TOKEN_KEY) || ''
  return `${protocol}://${window.location.host}${apiPrefix}/websocket/namespaces/${namespace}/pods/${podName}/containers/${containerName}/terminal?token=${token}`
}

// 发送终端大小变更消息
function sendTerminalResize(websocket: WebSocket, cols: number, rows: number) {
  if (websocket.readyState === WebSocket.OPEN) {
    const resizeMessage = {
      op: 'resize',
      cols: cols,
      rows: rows,
    }
    websocket.send(JSON.stringify(resizeMessage))
  }
}

// 发送终端输入消息
function sendTerminalInput(websocket: WebSocket, data: string) {
  if (websocket.readyState === WebSocket.OPEN) {
    const inputMessage = {
      op: 'stdin',
      data: data,
    }
    websocket.send(JSON.stringify(inputMessage))
  }
}

function TerminalCard({
  namespacedName,
  selectedContainer,
}: {
  namespacedName: PodNamespacedName
  selectedContainer: ContainerInfo
}) {
  const apiPrefix = useAtomValue(configAPIPrefixAtom)
  const terminalRef = useRef<Terminal | null>(null)
  const xtermRef = useRef<HTMLDivElement>(null)
  const fitAddonRef = useRef<FitAddon | null>(null)
  const wsRef = useRef<WebSocket | null>(null)

  // 使用防抖函数，延迟300ms发送尺寸变更信息
  const debouncedSendResize = useDebounceCallback((ws: WebSocket, cols: number, rows: number) => {
    sendTerminalResize(ws, cols, rows)
  }, 300)

  useEffect(() => {
    if (!selectedContainer.state.running || !xtermRef.current) {
      return
    }
    const terminal = new Terminal()
    const fitAddon = new FitAddon()

    terminalRef.current = terminal
    fitAddonRef.current = fitAddon

    terminal.loadAddon(fitAddon)
    terminal.open(xtermRef.current)

    // Add custom class to xterm-screen
    const xtermScreen = xtermRef.current.querySelector('.xterm')
    if (xtermScreen) {
      xtermScreen.classList.add('custom-xterm-screen')
    }

    terminal.focus()
    fitAddon.fit() // 调整终端大小

    const wsUrl = buildWebSocketUrl(
      apiPrefix,
      namespacedName.namespace,
      namespacedName.name,
      selectedContainer.name
    )

    const ws = new WebSocket(wsUrl)
    wsRef.current = ws

    ws.onmessage = (event: MessageEvent) => {
      terminal.write(event.data as string)
    }

    ws.onerror = () => {
      terminal.writeln('WebSocket error')
    }

    ws.onclose = () => {
      //另起一行，说明关闭的时间
      terminal.writeln('')
      terminal.writeln('Connection closed on ' + new Date().toLocaleString())
      terminal.writeln(
        '尊敬的用户您好，我们在网页端提供的终端，由于用户无操作、网络波动等因素，并不稳定。'
      )
      terminal.writeln(
        '这个终端比较适合简易调试的场景，如果您有复杂调试的需求，建议使用 SSH、Jupyter、CodeServer 等功能。'
      )
    }

    ws.onopen = () => {
      // 连接打开后立即发送初始终端大小（这里不需要防抖）
      const { cols, rows } = terminal
      sendTerminalResize(ws, cols, rows)
    }

    // 监听窗口调整，自动调整终端大小
    const handleResize = () => {
      fitAddon.fit()

      // 获取调整后的终端尺寸并发送到后端（使用防抖）
      const { cols, rows } = terminal
      debouncedSendResize(ws, cols, rows)
    }
    window.addEventListener('resize', handleResize)

    // 监听终端大小变化（使用防抖）
    terminal.onResize((dimensions) => {
      debouncedSendResize(ws, dimensions.cols, dimensions.rows)
    })

    // 将终端的输入发送到 WebSocket，使用新的格式
    // 注意：输入不需要防抖，应该立即发送
    terminal.onData((data) => {
      sendTerminalInput(ws, data)
    })

    return () => {
      window.removeEventListener('resize', handleResize)
      ws.close()
      terminal.dispose()
    }
  }, [namespacedName, selectedContainer, apiPrefix, debouncedSendResize])

  return (
    <Card className="overflow-hidden rounded-md bg-black p-1 md:col-span-2 xl:col-span-3 dark:border">
      <div ref={xtermRef} className="h-full w-full" />
    </Card>
  )
}

export default function TerminalDialog({
  namespacedName,
  setNamespacedName,
}: PodContainerDialogProps) {
  return (
    <PodContainerDialog
      namespacedName={namespacedName}
      setNamespacedName={setNamespacedName}
      ActionComponent={TerminalCard}
      type="shell"
    />
  )
}
