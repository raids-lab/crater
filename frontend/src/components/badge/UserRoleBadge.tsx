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
import { Role } from '@/services/api/auth'

import { PhaseBadge } from './PhaseBadge'

export const userRoles = [
  {
    value: Role.Admin.toString(),
    label: '管理员',
    color: 'text-highlight-purple bg-highlight-purple/20',
    description: '账户内的用户管理、资源分配等操作，将由账户管理员负责',
  },
  {
    value: Role.User.toString(),
    label: '用户',
    color: 'text-highlight-blue bg-highlight-blue/20',
    description: '账户内的普通用户，只能使用资源，无法进行管理操作',
  },
]

const getUserRoleLabel = (
  phase: string
): {
  label: string
  color: string
  description: string
} => {
  const foundPhase = userRoles.find((p) => p.value === phase)
  if (foundPhase) {
    return foundPhase
  } else {
    return {
      label: '未知',
      color: 'text-highlight-slate bg-highlight-slate/20',
      description: '由于某些原因无法获取用户角色信息',
    }
  }
}

const UserRoleBadge = ({ role }: { role: string }) => {
  return <PhaseBadge phase={role} getPhaseLabel={getUserRoleLabel} />
}

export default UserRoleBadge
