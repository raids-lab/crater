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
// Modified code
import Identicon from '@polkadot/react-identicon'
import { useTranslation } from 'react-i18next'

import { Avatar, AvatarFallback, AvatarImage } from '@/components/ui/avatar'

import { stringToSS58 } from '@/utils/ss58'

import { cn } from '@/lib/utils'

interface UserAvatarProps {
  user?: {
    avatar?: string
    name: string
  }
  size?: number
  className?: string
  identiconTheme?: 'beachball' | 'polkadot' | 'substrate'
}

export function UserAvatar({
  user,
  size = 32,
  className,
  identiconTheme = 'beachball',
}: UserAvatarProps) {
  const { t } = useTranslation()

  return (
    <Avatar className={cn('size-8', className)}>
      <AvatarImage src={user?.avatar} alt={t('userAvatar.alt')} />
      <AvatarFallback>
        <Identicon
          value={stringToSS58(user?.name || '')}
          size={size}
          theme={identiconTheme}
          className="cursor-default!"
        />
      </AvatarFallback>
    </Avatar>
  )
}
