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

import { ProjectStatus } from '@/services/api/account'

import { PhaseBadge } from './phase-badge'

const UserStatusBadge = ({ status }: { status: string }) => {
  const { t } = useTranslation()
  const phases = [
    {
      value: ProjectStatus.Inactive.toString(),
      label: t('userFields.status.inactive'),
      color: 'text-highlight-orange bg-highlight-orange/20',
      description: t('userFields.statusDescription.inactive'),
    },
    {
      value: ProjectStatus.Active.toString(),
      label: t('userFields.status.active'),
      color: 'text-highlight-emerald bg-highlight-emerald/20',
      description: t('userFields.statusDescription.active'),
    },
  ]
  return (
    <PhaseBadge
      phase={status}
      getPhaseLabel={(phase) =>
        phases.find((item) => item.value === phase) ?? {
          label: t('userFields.unknown'),
          color: 'text-highlight-slate bg-highlight-slate/20',
          description: t('userFields.statusDescription.unknown'),
        }
      }
    />
  )
}

export default UserStatusBadge
