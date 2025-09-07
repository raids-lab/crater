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
import { linkOptions } from '@tanstack/react-router'
import { ChevronDownIcon, LucideIcon } from 'lucide-react'
import { ReactNode, useMemo, useState } from 'react'

import { Button } from '@/components/ui/button'
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/components/ui/dropdown-menu'
import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import useFixedLayout from '@/hooks/useFixedLayout'

import { cn } from '@/lib/utils'

interface DetailSubTabProps {
  key: string
  label: string
  children: ReactNode
}

interface DetailTabProps {
  key: string
  icon?: LucideIcon
  label: string
  children?: ReactNode
  subTabs?: DetailSubTabProps[]
  scrollable?: boolean
  hidden?: boolean
}

interface DetailInfoProps {
  icon: LucideIcon
  title: string
  value: ReactNode
  className?: string
}

export interface DetailPageCoreProps {
  currentTab?: string
  setCurrentTab?: (tab: string) => void
}

export type DetailPageSearch = {
  tab?: string
}

export const detailValidateSearch = (search: Record<string, unknown>): DetailPageSearch => {
  return {
    tab: (search.tab as string) || undefined,
  }
}

export const detailLinkOptions = (tab: string) =>
  linkOptions({
    to: '.',
    search: { tab },
    replace: true,
  })

interface DetailPageProps extends DetailPageCoreProps {
  header: ReactNode
  info: DetailInfoProps[]
  tabs: DetailTabProps[]
}

export default function DetailPage({
  header,
  info,
  tabs: rawTabs,
  currentTab,
  setCurrentTab,
}: DetailPageProps) {
  useFixedLayout()

  const [selectedSubTab, setSelectedSubTab] = useState<Record<string, string>>({})

  const tabs = useMemo(() => {
    return rawTabs.filter((tab) => !tab.hidden)
  }, [rawTabs])

  const tab = useMemo(() => {
    if (tabs.length === 0) {
      return ''
    }
    return currentTab === undefined || currentTab === '' ? tabs[0].key : currentTab
  }, [currentTab, tabs])

  // 获取当前选中的子选项卡
  const getCurrentSubTab = (tabKey: string) => {
    const currentTabData = tabs.find((t) => t.key === tabKey)
    if (!currentTabData?.subTabs || currentTabData.subTabs.length === 0) {
      return null
    }
    return selectedSubTab[tabKey] || currentTabData.subTabs[0].key
  }

  // 设置子选项卡
  const setSubTab = (tabKey: string, subTabKey: string) => {
    setSelectedSubTab((prev) => ({
      ...prev,
      [tabKey]: subTabKey,
    }))
  }

  return (
    <div className="flex h-full w-full flex-col space-y-6">
      <div className="h-32 space-y-6">
        {header}
        <div className="text-muted-foreground grid grid-cols-3 gap-3 text-sm">
          {info.map((data, index) => (
            <div key={index} className={cn('flex items-center', data.className)}>
              <data.icon className="text-muted-foreground mr-1.5 size-4" />
              <span className="text-muted-foreground mr-1.5 truncate text-sm">{data.title}:</span>
              <span className="truncate">{data.value}</span>
            </div>
          ))}
        </div>
      </div>
      <Tabs className="w-full grow overflow-hidden" value={tab} onValueChange={setCurrentTab}>
        <TabsList className="tabs-list-underline">
          {tabs.map((tabItem) => {
            // 如果有子选项卡，渲染下拉菜单
            if (tabItem.subTabs && tabItem.subTabs.length > 0) {
              const currentSubTab = getCurrentSubTab(tabItem.key)

              return (
                <DropdownMenu key={tabItem.key}>
                  <DropdownMenuTrigger asChild>
                    <Button
                      variant="ghost"
                      className="tabs-trigger-underline data-[state=active]:border-primary flex items-center gap-1 data-[state=active]:border-b-2"
                      data-state={tab === tabItem.key ? 'active' : 'inactive'}
                      onClick={() => setCurrentTab?.(tabItem.key)}
                    >
                      {tabItem.icon && <tabItem.icon className="size-4" />}
                      <p className="hidden md:block">{tabItem.label}</p>
                      <ChevronDownIcon className="size-3" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent>
                    {tabItem.subTabs.map((subTab) => (
                      <DropdownMenuItem
                        key={subTab.key}
                        onClick={() => {
                          setCurrentTab?.(tabItem.key)
                          setSubTab(tabItem.key, subTab.key)
                        }}
                        className={currentSubTab === subTab.key ? 'bg-accent' : ''}
                      >
                        {subTab.label}
                      </DropdownMenuItem>
                    ))}
                  </DropdownMenuContent>
                </DropdownMenu>
              )
            }

            // 普通选项卡
            return (
              <TabsTrigger key={tabItem.key} className="tabs-trigger-underline" value={tabItem.key}>
                {tabItem.icon && <tabItem.icon className="size-4" />}
                <p className="hidden md:block">{tabItem.label}</p>
              </TabsTrigger>
            )
          })}
        </TabsList>
        {tabs.map((tabItem) => (
          <TabsContent key={tabItem.key} value={tabItem.key} asChild>
            <div className="w-full">
              {(() => {
                // 如果有子选项卡，渲染对应的子选项卡内容
                if (tabItem.subTabs && tabItem.subTabs.length > 0) {
                  const currentSubTab = getCurrentSubTab(tabItem.key)
                  const currentSubTabData = tabItem.subTabs.find((sub) => sub.key === currentSubTab)
                  const content = currentSubTabData?.children || null

                  if (tabItem.scrollable) {
                    return (
                      <ScrollArea className="h-[calc(100vh_-_300px)] w-full">
                        {content}
                        <ScrollBar orientation="horizontal" />
                      </ScrollArea>
                    )
                  } else {
                    return <div className="h-[calc(100vh_-_300px)] w-full">{content}</div>
                  }
                }

                // 普通选项卡内容
                if (tabItem.scrollable) {
                  return (
                    <ScrollArea className="h-[calc(100vh_-_300px)] w-full">
                      {tabItem.children}
                      <ScrollBar orientation="horizontal" />
                    </ScrollArea>
                  )
                } else {
                  return <div className="h-[calc(100vh_-_300px)] w-full">{tabItem.children}</div>
                }
              })()}
            </div>
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
