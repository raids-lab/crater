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
import { useTranslation } from 'react-i18next'

import { Role } from '@/services/api/auth'

import { PhaseBadge } from './phase-badge'

export const getUserRoles = (t: (key: string) => string) => [
  {
    value: Role.Admin.toString(),
    label: t(`userTable.roles.${Role.Admin}`),
    color: 'text-highlight-purple bg-highlight-purple/20',
    description: t('userFields.roleDescription.admin'),
  },
  {
    value: Role.User.toString(),
    label: t(`userTable.roles.${Role.User}`),
    color: 'text-highlight-blue bg-highlight-blue/20',
    description: t('userFields.roleDescription.user'),
  },
]

const UserRoleBadge = ({ role }: { role: string }) => {
  const { t } = useTranslation()
  return (
    <PhaseBadge
      phase={role}
      getPhaseLabel={(phase) =>
        getUserRoles(t).find((item) => item.value === phase) ?? {
          label: t('userFields.unknown'),
          color: 'text-highlight-slate bg-highlight-slate/20',
          description: t('userFields.roleDescription.unknown'),
        }
      }
    />
  )
}

export default UserRoleBadge
