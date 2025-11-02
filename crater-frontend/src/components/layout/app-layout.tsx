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
import { useAtomValue } from 'jotai'
import { CogIcon } from 'lucide-react'
import { motion } from 'motion/react'
import { useMemo } from 'react'

import { Badge } from '@/components/ui/badge'
import { SidebarInset, SidebarProvider, SidebarTrigger } from '@/components/ui/sidebar'

import { NavBreadcrumb } from '@/components/layout/app-breadcrumb'
import { AppSidebar } from '@/components/sidebar/app-sidebar'

import { atomFixedLayout, globalSettings } from '@/utils/store'

import { cn } from '@/lib/utils'

import { NavGroupProps } from '../sidebar/types'
import { WhatsNewDialog } from './what-is-new'

interface AppLayoutProps {
  groups: NavGroupProps[]
  rawPath: string
  children?: React.ReactNode
}

const AppLayout = ({ groups, rawPath, children }: AppLayoutProps) => {
  const fixedLayout = useAtomValue(atomFixedLayout)
  const scheduler = useAtomValue(globalSettings).scheduler

  // 特殊规则，网盘路由切换时，不启用过渡动画
  const motionKey = useMemo(() => {
    // begins with /portal/files/
    if (rawPath.startsWith('/portal/files/')) {
      return '/portal/files/'
    }
    if (rawPath.startsWith('/admin/files/')) {
      return '/admin/files/'
    }
    return rawPath
  }, [rawPath])

  return (
    <SidebarProvider>
      <AppSidebar groups={groups} />
      <SidebarInset>
        <header
          className={cn(
            'flex h-16 shrink-0 items-center justify-between gap-2 px-6 transition-[width,height] ease-linear',
            // "group-has-data-[collapsible=icon]/sidebar-wrapper:h-16",
            fixedLayout && 'header-fixed peer/header fixed z-50 w-[inherit] rounded-md'
          )}
        >
          <div className="flex items-center gap-2">
            <SidebarTrigger className="-ml-1" />
            <NavBreadcrumb className="hidden md:flex" />
          </div>
          {scheduler !== 'volcano' && (
            <Badge variant="secondary" className="uppercase">
              <CogIcon />
              {scheduler}
            </Badge>
          )}
        </header>
        <motion.div
          key={motionKey}
          initial={{ opacity: 0, y: '3vh' }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ type: 'spring', duration: 1.2 }}
          className={cn(
            '@container/main flex flex-col gap-6 p-6 pt-0',
            fixedLayout &&
              'absolute top-0 right-0 bottom-0 left-0 w-full grow overflow-hidden peer-[.header-fixed]/header:mt-16'
          )}
        >
          {children}
        </motion.div>
        <WhatsNewDialog />
      </SidebarInset>
    </SidebarProvider>
  )
}

export default AppLayout
