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
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { useEffect, useState } from 'react'

import { ContainerInfo, apiGetPodContainers } from '@/services/api/tool'

import { cn } from '@/lib/utils'

import LoadingCircleIcon from '../icon/loading-circle-icon'
import { LogCard } from './log-dialog'
import { ContainerSelect, PodNamespacedName, TableCellForm } from './pod-container-dialog'

export default function DetailPageLog({
  namespacedName,
  children,
  appendInfos,
}: {
  namespacedName?: PodNamespacedName
  children?: React.ReactNode
  appendInfos?: {
    title: string
    content: React.ReactNode
  }[]
}) {
  const { namespace, name: podName } = namespacedName ?? {}
  const queryClient = useQueryClient()

  const [selectedContainer, setSelectedContainer] = useState<ContainerInfo | undefined>()

  const { data: containers } = useQuery({
    queryKey: ['log', 'containers', namespace, podName],
    queryFn: () => apiGetPodContainers(namespace, podName),
    select: (res) => res.data.containers.filter((c) => c.name !== ''),
    enabled: !!namespace && !!podName,
  })

  useEffect(() => {
    for (const container of containers || []) {
      if (!container.isInitContainer) {
        setSelectedContainer((prev) => {
          return prev || container
        })
        void queryClient.invalidateQueries({ queryKey: ['logtext'] })
        break
      }
    }
  }, [containers, queryClient])

  return (
    <>
      {!containers || !selectedContainer ? (
        <div className="flex h-[calc(100vh_-_300px)] w-full items-center justify-center">
          <LoadingCircleIcon />
        </div>
      ) : (
        <div className="grid h-[calc(100vh_-_300px)] w-full gap-6 md:grid-cols-3 xl:grid-cols-4">
          {namespacedName && selectedContainer && (
            <LogCard namespacedName={namespacedName} selectedContainer={selectedContainer} />
          )}
          <div className="space-y-4">
            {children ?? (
              <ContainerSelect
                currentContainer={selectedContainer}
                setCurrentContainer={setSelectedContainer}
                containers={containers}
              />
            )}
            <fieldset
              className={cn(
                'border-input hidden h-[calc(100vh_-374px)] max-h-full gap-6 overflow-y-auto rounded-lg border p-4 shadow-xs md:grid',
                {
                  'h-[calc(100vh_-350px)]': !!children,
                }
              )}
            >
              <legend className="-ml-1 px-2 text-sm font-medium">
                {selectedContainer.isInitContainer ? '初始化容器' : '容器信息'}
              </legend>
              <TableCellForm
                namespace={namespace}
                podName={podName}
                selectedContainer={selectedContainer}
                appendInfos={appendInfos}
              />
            </fieldset>
          </div>
        </div>
      )}
    </>
  )
}
