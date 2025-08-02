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
import { Link, linkOptions } from '@tanstack/react-router'
import { useAtomValue } from 'jotai'

import { IUserInfo } from '@/services/api/vcjob'

import useIsAdmin from '@/hooks/use-admin'

import { getUserPseudonym } from '@/utils/pseudonym'
import { globalHideUsername } from '@/utils/store'

import { cn } from '@/lib/utils'

import SimpleTooltip from './simple-tooltip'

const adminUserLinkOptions = linkOptions({
  to: '/admin/users/$name',
  params: { name: '' },
  search: { tab: '' },
})

const portalUserLinkOptions = linkOptions({
  to: '/portal/users/$name',
  params: { name: '' },
  search: { tab: '' },
})

const UserLabel = ({ info, className }: { info: IUserInfo; className?: string }) => {
  const hideUsername = useAtomValue(globalHideUsername)
  const isAdminMode = useIsAdmin()

  // 根据 hideUsername 状态决定显示真实名称还是假名
  const displayName = hideUsername
    ? getUserPseudonym(info.username)
    : info.nickname || info.username

  return (
    <SimpleTooltip
      tooltip={
        <p>
          查看{displayName}
          <span className="mx-0.5 font-mono">
            (@{hideUsername ? getUserPseudonym(info.username) : info.username})
          </span>
          信息
        </p>
      }
    >
      <Link
        {...(isAdminMode ? adminUserLinkOptions : portalUserLinkOptions)}
        params={{ name: info.username }}
        search={{ tab: 'gpu' }}
      >
        <span className={cn('truncate text-sm font-normal', className)}>{displayName}</span>
      </Link>
    </SimpleTooltip>
  )
}

export default UserLabel
