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

import { ChevronsUpDown } from 'lucide-react'

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import {
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  useSidebar,
} from '@/components/ui/sidebar'
import { useMemo } from 'react'
import { globalAccount } from '@/utils/store'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import { apiQueueSwitch } from '@/services/api/auth'
import { useLocation } from 'react-router-dom'
import { QueueBasic, apiQueueList } from '@/services/api/queue'
import { useAtom } from 'jotai'
import { showErrorToast } from '@/utils/toast'
import { Avatar } from '@radix-ui/react-avatar'
import Identicon from '@polkadot/react-identicon'
import { stringToSS58 } from '@/utils/ss58'
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '../ui/tooltip'
import { format, formatDistanceToNow } from 'date-fns'
import { zhCN } from 'date-fns/locale'

export function TeamSwitcher() {
  const { isMobile } = useSidebar()

  const [account, setAccount] = useAtom(globalAccount)
  const queryClient = useQueryClient()
  const location = useLocation()

  const isAdminView = useMemo(() => {
    const pathParts = location.pathname.split('/').filter(Boolean)
    return pathParts[0] === 'admin'
  }, [location])

  const { data: queues } = useQuery({
    queryKey: ['queues'],
    queryFn: apiQueueList,
    select: (res) => res.data.data,
  })

  const currentQueue = useMemo(() => {
    return queues?.find((p) => p.name === account.queue)
  }, [queues, account])

  const [currentExpiredAt, currentExpiredDiff] = useMemo(() => {
    if (!currentQueue || currentQueue.expiredAt === null || currentQueue.expiredAt === undefined) {
      return [null, 0]
    }
    const expiredAt = new Date(currentQueue.expiredAt)
    return [expiredAt, expiredAt.getTime() - Date.now()]
  }, [currentQueue])

  const { mutate: switchQueue } = useMutation({
    mutationFn: (project: QueueBasic) => apiQueueSwitch(project.name),
    onSuccess: ({ context }, { nickname: name }) => {
      setAccount(context)
      toast.success(`已切换至账户 ${name}`)
      void queryClient.invalidateQueries()
    },
    onError: (error) => {
      showErrorToast(error)
    },
  })

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger disabled={isAdminView} asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Avatar className="h-8 w-8 rounded-full">
                {currentQueue?.name && (
                  <Identicon
                    value={stringToSS58(currentQueue?.name)}
                    size={32}
                    // 'beachball' | 'empty' | 'ethereum' | 'jdenticon' | 'polkadot' | 'substrate'
                    theme="substrate"
                    className="cursor-pointer!"
                  />
                )}
              </Avatar>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-semibold">{currentQueue?.nickname}</span>
                {currentExpiredAt && (
                  <span className="truncate text-xs">
                    <TooltipProvider delayDuration={10}>
                      <Tooltip>
                        <TooltipTrigger>
                          <span className="text-muted-foreground text-xs">
                            {currentExpiredDiff < 0 ? (
                              <>已过期</>
                            ) : (
                              <>
                                {formatDistanceToNow(currentExpiredAt, {
                                  locale: zhCN,
                                  addSuffix: true,
                                })}
                                有效
                              </>
                            )}
                          </span>
                        </TooltipTrigger>
                        <TooltipContent
                          side="right"
                          className="bg-background text-foreground border"
                        >
                          {format(currentExpiredAt, 'PPP', {
                            locale: zhCN,
                          })}
                          过期
                        </TooltipContent>
                      </Tooltip>
                    </TooltipProvider>
                  </span>
                )}
              </div>
              <ChevronsUpDown className="ml-auto" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-44 rounded-lg"
            align="start"
            side={isMobile ? 'bottom' : 'right'}
            sideOffset={4}
          >
            <DropdownMenuLabel className="text-muted-foreground text-xs">账户</DropdownMenuLabel>
            {queues?.map((queue) => (
              <DropdownMenuItem
                key={queue.nickname}
                onClick={() => {
                  if (currentQueue?.name !== queue.name) {
                    switchQueue(queue)
                  }
                }}
                className="gap-2 p-2"
              >
                <div className="size-6 overflow-hidden">
                  <Identicon
                    value={stringToSS58(queue.name)}
                    size={24}
                    theme="substrate"
                    className="cursor-pointer!"
                  />
                </div>
                <span className="truncate">{queue.nickname}</span>
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
