// Copyright 2026 The Crater Project Team, RAIDS-Lab
// SPDX-License-Identifier: Apache-2.0
import type { createSessionClient } from './session-client.ts'

interface TerminalSessionOptions {
  client: Pick<ReturnType<typeof createSessionClient>, 'withAccessToken'>
  origin: string
  apiBase: string
  namespace: string
  podName: string
  containerName: string
  signal: AbortSignal
  createSocket?: (url: string) => WebSocket
}

export function openTerminalWebSocket(options: TerminalSessionOptions) {
  return options.client.withAccessToken(
    (token) => {
      const path = [options.namespace, options.podName, options.containerName].map(
        encodeURIComponent
      )
      const url = new URL(
        `${options.apiBase}/api/v1/websocket/namespaces/${path[0]}/pods/${path[1]}/containers/${path[2]}/terminal`,
        options.origin
      )
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:'
      url.searchParams.set('token', token)
      return options.createSocket ? options.createSocket(url.toString()) : new WebSocket(url)
    },
    { signal: options.signal }
  )
}
