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

export const progressTextColor = (percent: number) => {
  return cn('text-highlight-emerald mb-0.5 font-mono text-sm font-bold', {
            'text-highlight-emerald': percent <= 20,
            'text-highlight-sky': percent > 20 && percent <= 50,
            'text-highlight-yellow': percent > 50 && percent <= 70,
            'text-highlight-orange': percent > 70 && percent <= 90,
            'text-highlight-red': percent > 90,
          })
}

export const ProgressBar = ({
  percent,
  label,
  className,
}: {
  percent: number
  label?: string
  className?: string
}) => {
  const newWidth = percent > 100 ? 100 : percent
  return (
    <div
      className={cn(
        'bg-accent text-foreground relative h-2 rounded-md',
        {
          'text-white': percent > 90,
        },
        className
      )}
    >
      <div
        className={cn(
          'h-2 rounded-md transition-all duration-500',
          {
            'bg-highlight-emerald': percent <= 20,
            'bg-highlight-sky': percent > 20 && percent <= 50,
            'bg-highlight-yellow': percent > 50 && percent <= 70,
            'bg-highlight-orange': percent > 70 && percent <= 90,
            'bg-highlight-red': percent > 90,
          },
          className
        )}
        style={{ width: `${newWidth}%` }}
      ></div>
      {label && (
        <div className="absolute inset-0 font-mono text-xs font-medium">
          <div className="text-center">{label}</div>
        </div>
      )}
    </div>
  )
}
