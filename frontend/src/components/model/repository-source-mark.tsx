/**
 * Copyright 2026 The Crater Project Team, RAIDS-Lab
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
import { DatabaseIcon, PackageIcon } from 'lucide-react'
import { useState } from 'react'

import { cn } from '@/lib/utils'

interface RepositorySourceMarkProps {
  source?: string
  organization?: string
  logoURL?: string
  category?: 'model' | 'dataset' | 'sharefile'
  className?: string
}

const organizationGradients = [
  'from-violet-600 to-indigo-500',
  'from-sky-500 to-cyan-400',
  'from-emerald-500 to-teal-400',
  'from-orange-500 to-amber-400',
  'from-rose-500 to-pink-400',
  'from-fuchsia-600 to-purple-500',
]

export default function RepositorySourceMark({
  source,
  organization,
  logoURL,
  category = 'model',
  className,
}: RepositorySourceMarkProps) {
  const [imageFailed, setImageFailed] = useState(false)
  const normalizedSource = source?.toLowerCase()

  if (logoURL && !imageFailed) {
    return (
      <img
        src={logoURL}
        alt={organization || source || ''}
        title={organization || source}
        className={cn('size-5 shrink-0 rounded-md object-cover', className)}
        onError={() => setImageFailed(true)}
      />
    )
  }

  if (organization) {
    const colorIndex = Array.from(organization).reduce(
      (sum, character) => sum + character.charCodeAt(0),
      0
    )
    const gradient = organizationGradients[colorIndex % organizationGradients.length]
    return (
      <span
        aria-label={organization}
        title={organization}
        className={cn(
          'inline-flex size-5 shrink-0 items-center justify-center rounded-md bg-gradient-to-br text-[10px] font-bold text-white shadow-sm',
          gradient,
          className
        )}
      >
        {organization.slice(0, 1).toUpperCase()}
      </span>
    )
  }

  if (normalizedSource === 'huggingface') {
    return (
      <span
        aria-label="HuggingFace"
        title="HuggingFace"
        className={cn(
          'inline-flex size-5 shrink-0 items-center justify-center text-base',
          className
        )}
      >
        🤗
      </span>
    )
  }

  if (normalizedSource === 'modelscope') {
    return (
      <span
        aria-label="ModelScope"
        title="ModelScope"
        className={cn(
          'inline-flex size-5 shrink-0 items-center justify-center rounded-md bg-gradient-to-br from-violet-600 to-blue-500 text-[10px] font-bold text-white shadow-sm',
          className
        )}
      >
        M
      </span>
    )
  }

  const Icon = category === 'dataset' ? DatabaseIcon : PackageIcon
  return (
    <span
      className={cn(
        'bg-muted text-muted-foreground inline-flex size-5 shrink-0 items-center justify-center rounded-md',
        className
      )}
    >
      <Icon className="size-3" />
    </span>
  )
}
