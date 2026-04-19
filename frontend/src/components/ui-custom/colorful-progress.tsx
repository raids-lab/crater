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
import { cn } from '@/lib/utils'

export type ProgressMode = 'usage' | 'balance'

const getProgressTone = (percent: number, mode: ProgressMode) => {
  if (mode === 'balance') {
    if (percent > 90) {
      return 'emerald'
    }
    if (percent > 70) {
      return 'sky'
    }
    if (percent > 50) {
      return 'yellow'
    }
    if (percent > 20) {
      return 'orange'
    }
    return 'red'
  }
  if (percent <= 20) {
    return 'emerald'
  }
  if (percent <= 50) {
    return 'sky'
  }
  if (percent <= 70) {
    return 'yellow'
  }
  if (percent <= 90) {
    return 'orange'
  }
  return 'red'
}

export const progressTextColor = (percent: number, mode: ProgressMode = 'usage') => {
  const tone = getProgressTone(percent, mode)
  return cn('text-highlight-emerald mb-0.5 font-mono text-sm font-bold', {
    'text-highlight-emerald': tone === 'emerald',
    'text-highlight-sky': tone === 'sky',
    'text-highlight-yellow': tone === 'yellow',
    'text-highlight-orange': tone === 'orange',
    'text-highlight-red': tone === 'red',
  })
}

export const ProgressBar = ({
  percent,
  label,
  className,
  mode = 'usage',
}: {
  percent: number
  label?: string
  className?: string
  mode?: ProgressMode
}) => {
  const normalizedPercent = Math.min(100, Math.max(0, percent))
  const tone = getProgressTone(normalizedPercent, mode)
  return (
    <div
      className={cn(
        'bg-accent text-foreground relative h-2 rounded-md',
        {
          'text-white': mode === 'usage' && normalizedPercent > 90,
        },
        className
      )}
    >
      <div
        className={cn(
          'h-2 rounded-md transition-all duration-500',
          {
            'bg-highlight-emerald': tone === 'emerald',
            'bg-highlight-sky': tone === 'sky',
            'bg-highlight-yellow': tone === 'yellow',
            'bg-highlight-orange': tone === 'orange',
            'bg-highlight-red': tone === 'red',
          },
          className
        )}
        style={{ width: `${normalizedPercent}%` }}
      ></div>
      {label && (
        <div className="absolute inset-0 font-mono text-xs font-medium">
          <div className="text-center">{label}</div>
        </div>
      )}
    </div>
  )
}
