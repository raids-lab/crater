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
import { UsersRoundIcon } from 'lucide-react'
import { useMemo } from 'react'

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
  useSidebar,
} from '@/components/ui/sidebar'

import { GitHubStarCard } from '@/components/layout/github-star-card'
import { NavGroup } from '@/components/sidebar/nav-main'
import { NavUser } from '@/components/sidebar/nav-user'
import { TeamSwitcher } from '@/components/sidebar/team-switcher'

import useIsAdmin from '@/hooks/use-admin'

import { atomUserContext } from '@/utils/store'

import { NavGroupProps } from './types'

function GitHubStarCardWrapper() {
  const { state } = useSidebar()

  // 只在侧边栏展开时显示卡片
  if (state === 'collapsed') {
    return null
  }

  return (
    <div className="px-2 pb-2">
      <GitHubStarCard />
    </div>
  )
}

export function AppSidebar({
  groups,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  groups: NavGroupProps[]
}) {
  const isAdminView = useIsAdmin()
  const accountInfo = useAtomValue(atomUserContext)

  // Special rule: when current account is not default account and in user view, add account management menu
  const filteredGroups = useMemo(() => {
    if (
      !isAdminView &&
      accountInfo?.queue !== 'default' &&
      groups.length > 0 &&
      groups[groups.length - 1].items.length > 0 &&
      groups[groups.length - 1].items[0].title !== '账户管理'
    ) {
      groups[groups.length - 1].items = [
        {
          title: '账户管理',
          icon: UsersRoundIcon,
          items: [
            {
              title: '成员管理',
              url: '/portal/account/member',
            },
            {
              title: '卡时统计',
              url: '/portal/account/statistics',
            },
          ],
        },
        ...groups[groups.length - 1].items,
      ]
      return groups
    }
    // Revert: remove account management menu if in admin view or default account
    if (
      (isAdminView || accountInfo?.queue === 'default') &&
      groups.length > 0 &&
      groups[groups.length - 1].items.length > 0 &&
      groups[groups.length - 1].items[0].title === '账户管理'
    ) {
      groups[groups.length - 1].items = groups[groups.length - 1].items.slice(1)
    }
    return groups
  }, [isAdminView, accountInfo, groups])

  return (
    <Sidebar collapsible="icon" {...props}>
      <SidebarHeader>
        <TeamSwitcher />
      </SidebarHeader>
      <SidebarContent className="gap-0">
        {filteredGroups.map((group) => (
          <NavGroup key={group.title} {...group} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <GitHubStarCardWrapper />
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
