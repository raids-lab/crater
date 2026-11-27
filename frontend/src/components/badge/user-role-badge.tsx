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
import { t } from 'i18next'
import { useTranslation } from 'react-i18next'

import { Role } from '@/services/api/auth'

import { PhaseBadge } from './phase-badge'

export const userRoles = [
  {
    value: Role.Admin.toString(),
    get label() {
      return t('accountDetail.form.userRole.admin')
    },
    color: 'text-highlight-purple bg-highlight-purple/20',
    get description() {
      return t('accountDetail.role.adminDescription')
    },
  },
  {
    value: Role.User.toString(),
    get label() {
      return t('accountDetail.form.userRole.normal')
    },
    color: 'text-highlight-blue bg-highlight-blue/20',
    get description() {
      return t('accountDetail.role.userDescription')
    },
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
      label: t('accountDetail.role.unknown'),
      color: 'text-highlight-slate bg-highlight-slate/20',
      description: t('accountDetail.role.unknownDescription'),
    }
  }
}

const UserRoleBadge = ({ role }: { role: string }) => {
  useTranslation()
  return <PhaseBadge phase={role} getPhaseLabel={getUserRoleLabel} />
}

export default UserRoleBadge
