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
import { LazyLog } from '@melloware/react-logviewer'
import { CSSProperties } from 'react'

interface EnhancedLogViewerProps {
  logText?: string
  streamUrl?: string
  streaming?: boolean
  followLog?: boolean
  height?: string | number
  enableSearch?: boolean
  enableLineNumbers?: boolean
  caseInsensitive?: boolean
  selectableLines?: boolean
  websocket?: boolean
  websocketOptions?: {
    formatMessage?: (message: string) => string
  }
  extraLines?: number
  style?: CSSProperties
  fetchOptions?: RequestInit
}

/**
 * Enhanced Log Viewer Component using @melloware/react-logviewer
 * Provides advanced features like search, highlight, line numbers, etc.
 *
 * This component can work in two modes:
 * 1. Static mode: Pass logText prop to display static logs
 * 2. Streaming mode: Pass streamUrl prop to stream logs from a URL
 */
export function EnhancedLogViewer({
  logText,
  streamUrl,
  streaming = false,
  followLog = true,
  enableSearch = true,
  enableLineNumbers = true,
  caseInsensitive = true,
  selectableLines = true,
  websocket = false,
  websocketOptions,
  extraLines = 1,
  style,
  fetchOptions,
}: EnhancedLogViewerProps) {
  // Determine if we should use URL mode or text mode
  const useUrl = streaming && streamUrl

  return (
    <LazyLog
      // Either provide text or url, not both
      text={useUrl ? undefined : logText || ''}
      url={useUrl ? streamUrl : undefined}
      // Stream mode settings
      stream={streaming}
      follow={followLog}
      // Search and display settings
      enableSearch={enableSearch}
      enableSearchNavigation={enableSearch}
      enableHotKeys={enableSearch}
      caseInsensitive={caseInsensitive}
      // Line number settings
      enableLineNumbers={enableLineNumbers}
      selectableLines={selectableLines}
      // Highlight syntax (ANSI colors)
      enableGzip={false}
      // WebSocket settings
      websocket={websocket}
      websocketOptions={websocketOptions}
      // Fetch options for authentication
      fetchOptions={fetchOptions}
      // Performance settings
      overscanRowCount={100}
      extraLines={extraLines}
      style={{
        fontSize: '13px',
        fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace',
        ...style,
      }}
    />
  )
}

export default EnhancedLogViewer
