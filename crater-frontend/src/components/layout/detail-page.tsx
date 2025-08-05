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
import { motion } from 'framer-motion'
import { LucideIcon } from 'lucide-react'
import { ReactNode, useEffect, useMemo, useRef } from 'react'

import { ScrollArea, ScrollBar } from '@/components/ui/scroll-area'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'

import useFixedLayout from '@/hooks/useFixedLayout'

import { cn } from '@/lib/utils'

interface DetailTabProps {
  key: string
  icon?: LucideIcon
  label: string
  children: ReactNode
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

  const tabs = useMemo(() => {
    return rawTabs.filter((tab) => !tab.hidden)
  }, [rawTabs])

  const previousTabRef = useRef<string>('')

  const tab = useMemo(() => {
    if (tabs.length === 0) {
      return ''
    }
    return currentTab === undefined || currentTab === '' ? tabs[0].key : currentTab
  }, [currentTab, tabs])

  // 计算动画方向 - 基于从哪个 tab 切换过来
  const animationDirection = useMemo(() => {
    const previousIndex = tabs.findIndex((t) => t.key === previousTabRef.current)
    const currentIndex = tabs.findIndex((t) => t.key === tab)

    // 如果没有previous tab（首次加载），不应用动画
    if (previousTabRef.current === '' || previousIndex === -1) {
      return 'none'
    }

    // 根据索引位置决定动画方向
    return currentIndex > previousIndex ? 'right' : 'left'
  }, [tabs, tab])

  // 更新 previous tab
  useEffect(() => {
    if (tab !== previousTabRef.current) {
      previousTabRef.current = tab || ''
    }
  }, [tab])

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
          {tabs.map((tab) => (
            <TabsTrigger key={tab.key} className="tabs-trigger-underline" value={tab.key}>
              {tab.icon && <tab.icon className="size-4" />}
              <p className="hidden md:block">{tab.label}</p>
            </TabsTrigger>
          ))}
        </TabsList>
        {tabs.map((tabItem) => (
          <TabsContent key={tabItem.key} value={tabItem.key} asChild>
            <motion.div
              key={`${tabItem.key}-${tab}`}
              className="w-full"
              initial={{
                opacity: animationDirection === 'none' ? 1 : 0,
                x:
                  animationDirection === 'none'
                    ? 0
                    : animationDirection === 'right'
                      ? '50%'
                      : '-50%',
              }}
              animate={{
                opacity: 1,
                x: 0,
              }}
              transition={{
                type: 'spring',
                stiffness: 300,
                damping: 30,
                duration: animationDirection === 'none' ? 0 : 1.2,
              }}
            >
              {tabItem.scrollable ? (
                <ScrollArea className="h-[calc(100vh_-_300px)] w-full">
                  {tabItem.children}
                  <ScrollBar orientation="horizontal" />
                </ScrollArea>
              ) : (
                <div className="h-[calc(100vh_-_300px)] w-full">{tabItem.children}</div>
              )}
            </motion.div>
          </TabsContent>
        ))}
      </Tabs>
    </div>
  )
}
