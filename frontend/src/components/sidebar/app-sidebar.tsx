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
} from '@/components/ui/sidebar'

import { NavGroup } from '@/components/sidebar/nav-main'
import { NavUser } from '@/components/sidebar/nav-user'
import { TeamSwitcher } from '@/components/sidebar/team-switcher'

import { Role } from '@/services/api/auth'

import useIsAdmin from '@/hooks/use-admin'

import { atomUserContext } from '@/utils/store'

import { NavGroupProps } from './types'

export function AppSidebar({
  groups,
  ...props
}: React.ComponentProps<typeof Sidebar> & {
  groups: NavGroupProps[]
}) {
  const isAdminView = useIsAdmin()
  const accountInfo = useAtomValue(atomUserContext)

  // 特殊规则，当当前账户为其他账户时，且当前用户的权限是账户管理员，且当前处于用户模式时，添加账户管理菜单
  const filteredGroups = useMemo(() => {
    if (
      !isAdminView &&
      accountInfo?.queue !== 'default' &&
      accountInfo?.roleQueue === Role.Admin &&
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
              url: 'account/member',
            },
          ],
        },
        ...groups[groups.length - 1].items,
      ]
      return groups
    }
    // revert
    if (
      (isAdminView || accountInfo?.queue === 'default' || accountInfo?.roleQueue !== Role.Admin) &&
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
        <NavUser />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
