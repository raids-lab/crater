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

// i18n-processed-v1.1.0
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
import { Badge } from '../ui/badge'
import { ReactNode } from 'react'

const TipBadge = ({ className, title }: { className?: string; title?: ReactNode }) => {
  const { t } = useTranslation()

  return (
    <Badge
      className={cn(
        'rounded-full px-2 py-0 shadow-none select-none',
        'bg-highlight-orange/15 text-highlight-orange',
        className
      )}
    >
      {title || t('tipBadge.defaultTitle')}
    </Badge>
  )
}

export default TipBadge
