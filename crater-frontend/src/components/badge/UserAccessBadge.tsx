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
import { Access } from '@/services/api/account'

import { PhaseBadge } from './PhaseBadge'

export const userAccesses = [
  {
    value: Access.RW.toString(),
    label: '读写',
    color: 'text-highlight-orange bg-highlight-orange/20',
    description: '拥有对数据的读写权限',
  },
  {
    value: Access.RO.toString(),
    label: '只读',
    color: 'text-highlight-emerald bg-highlight-emerald/20',
    description: '只有对数据的读取权限',
  },
]

const getUserAccessLabel = (
  phase: string
): {
  label: string
  color: string
  description: string
} => {
  const foundPhase = userAccesses.find((p) => p.value === phase)
  if (foundPhase) {
    return foundPhase
  } else {
    return {
      label: '未知',
      color: 'text-highlight-slate bg-highlight-slate/20',
      description: '由于某些原因无法获取用户读写权限信息',
    }
  }
}

const UserAccessBadge = ({ access }: { access: string }) => {
  return <PhaseBadge phase={access} getPhaseLabel={getUserAccessLabel} />
}

export default UserAccessBadge
