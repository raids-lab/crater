/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import type { TFunction } from 'i18next'

import type { JsonObject } from '@/services/api/admin/operationLog'

export function getLLMConfigAuditRows(details: JsonObject, t: TFunction) {
  const scope = details.scope
  const rows = [
    {
      key: 'scope',
      label: t('operationLog.details.llm.scope'),
      value:
        scope === 'personal' || scope === 'platform'
          ? t(`operationLog.details.llm.scope.${scope}`)
          : t('operationLog.details.na'),
    },
  ]
  if (
    scope === 'personal' &&
    typeof details.userId === 'number' &&
    Number.isSafeInteger(details.userId) &&
    details.userId > 0
  ) {
    rows.push({
      key: 'userId',
      label: t('operationLog.details.llm.userId'),
      value: String(details.userId),
    })
  }
  if (typeof details.keySubmitted === 'boolean') {
    rows.push({
      key: 'keySubmitted',
      label: t('operationLog.details.llm.keySubmitted'),
      value: t(`operationLog.details.llm.keySubmitted.${details.keySubmitted ? 'yes' : 'no'}`),
    })
  }
  return rows
}
