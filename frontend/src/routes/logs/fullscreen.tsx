/**import { createFileRoute } from '@tanstack/react-router'

 * Copyright 2025 RAIDS Lab

 *export const Route = createFileRoute('/logs/fullscreen')({

 * Licensed under the Apache License, Version 2.0 (the "License");  component: RouteComponent,

 * you may not use this file except in compliance with the License.})

 * You may obtain a copy of the License at

 *function RouteComponent() {

 *      http://www.apache.org/licenses/LICENSE-2.0  return <div>Hello "/logs/fullscreen"!</div>

 *}

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { createFileRoute } from '@tanstack/react-router'
import { useMemo } from 'react'

import { Button } from '@/components/ui/button'

import TipBadge from '@/components/badge/tip-badge'
import EnhancedLogViewer from '@/components/codeblock/log-viewer'

import { ACCESS_TOKEN_KEY } from '@/utils/store'

interface LogViewerSearch {
  namespace: string
  pod: string
  container: string
  timestamps?: boolean
}

export const Route = createFileRoute('/logs/fullscreen')({
  component: FullscreenLogViewer,
  validateSearch: (search: Record<string, unknown>): LogViewerSearch => {
    return {
      namespace: (search.namespace as string) || '',
      pod: (search.pod as string) || '',
      container: (search.container as string) || '',
      timestamps: search.timestamps === 'true' || search.timestamps === true,
    }
  },
})

function FullscreenLogViewer() {
  const { namespace, pod, container, timestamps } = Route.useSearch()

  // Build stream URL
  const streamUrl = useMemo(() => {
    if (!namespace || !pod || !container) {
      return undefined
    }

    const baseUrl = `${import.meta.env.VITE_API_BASE || ''}/api/v1/namespaces/${namespace}/pods/${pod}/containers/${container}/log/stream`
    const params = new URLSearchParams({
      timestamps: timestamps ? 'true' : 'false',
    })

    return `${baseUrl}?${params}`
  }, [namespace, pod, container, timestamps])

  // Fetch options for authentication
  const fetchOptions = useMemo(() => {
    const token = localStorage.getItem(ACCESS_TOKEN_KEY)
    return {
      headers: {
        Authorization: `Bearer ${token}`,
        Accept: 'text/plain',
      },
    }
  }, [])

  const title = `${pod}/${container}`

  return (
    <div className="bg-background flex h-screen w-screen flex-col">
      {/* Header */}
      <div className="border-border flex items-center justify-between border-b px-4 py-2">
        <div className="flex items-center gap-2">
          <span className="font-mono">{title}</span>
          <TipBadge title={'日志'} />
        </div>
        <Button
          variant="outline"
          size="sm"
          onClick={() => window.close()}
          className="text-muted-foreground"
        >
          关闭
        </Button>
      </div>

      {/* Log Viewer */}
      <div className="h-500 w-screen">
        <EnhancedLogViewer
          streamUrl={streamUrl}
          streaming={true}
          followLog={true}
          enableSearch={true}
          enableLineNumbers={true}
          caseInsensitive={true}
          selectableLines={true}
          fetchOptions={fetchOptions}
        />
      </div>
    </div>
  )
}
