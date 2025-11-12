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
import { CSSProperties, useEffect, useRef } from 'react'

import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'

interface SimpleLogViewerProps {
  logText?: string
  streaming?: boolean
  followLog?: boolean
  height?: string | number
  enableSearch?: boolean
  enableLineNumbers?: boolean
  caseInsensitive?: boolean
  selectableLines?: boolean
  style?: CSSProperties
  // These props are for interface compatibility but not used in simple viewer
  streamUrl?: string
  fetchOptions?: RequestInit
  websocket?: boolean
  websocketOptions?: unknown
  extraLines?: number
}

/**
 * Simple Log Viewer Component - maintains the original style
 * This component provides a simple pre-based log viewer that matches
 * the original design while being compatible with EnhancedLogViewer interface
 */
export function SimpleLogViewer({
  logText = '',
  height = '100%',
  followLog = true,
  style,
}: SimpleLogViewerProps) {
  const logAreaRef = useRef<HTMLDivElement>(null)

  // Auto-scroll to bottom when followLog is enabled and content changes
  useEffect(() => {
    if (followLog && logAreaRef.current) {
      logAreaRef.current.scrollIntoView(false)
    }
  }, [logText, followLog])

  return (
    <ScrollArea style={{ height, width: '100%' }}>
      <div ref={logAreaRef}>
        <pre
          className="px-3 py-3 text-sm break-words whitespace-pre-wrap dark:text-blue-300"
          style={style}
        >
          {logText}
        </pre>
      </div>
      <ScrollBar orientation="horizontal" />
    </ScrollArea>
  )
}

export default SimpleLogViewer
