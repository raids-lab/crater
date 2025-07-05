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

// i18n-processed-v1.1.0 (no translatable strings)
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from '@/components/ui/tooltip'
import { cn } from '@/lib/utils'
import { format, formatDistanceToNow } from 'date-fns'
import { zhCN, ja, ko, enUS, type Locale } from 'date-fns/locale'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

export const TimeDistance = ({ date, className }: { date?: string; className?: string }) => {
  const [timeDiff, setTimeDiff] = useState('')
  const [startTime, setStartTime] = useState<Date | null>(null)
  const { i18n } = useTranslation()
  const [locale, setLocale] = useState<Locale>(zhCN)

  useEffect(() => {
    switch (i18n.language) {
      case 'zh':
        setLocale(zhCN)
        break
      case 'ja':
        setLocale(ja)
        break
      case 'ko':
        setLocale(ko)
        break
      default:
        setLocale(enUS)
    }
  }, [i18n.language])

  useEffect(() => {
    if (!date) {
      setStartTime(null)
      setTimeDiff('')
      return
    }

    const time = new Date(date)
    setStartTime(time)

    const updateTimeDiff = () => {
      const timeDifference = formatDistanceToNow(time, {
        locale,
        addSuffix: true,
      })
      setTimeDiff(timeDifference.replace(/^[^\d]*\s*/, ''))
    }

    updateTimeDiff()
    const timer = setInterval(updateTimeDiff, 10000)
    return () => clearInterval(timer)
  }, [date, locale])

  if (!startTime) {
    return null
  }

  return (
    <TooltipProvider delayDuration={10}>
      <Tooltip>
        <TooltipTrigger className={cn('cursor-help', className)}>{timeDiff}</TooltipTrigger>
        <TooltipContent>{format(startTime, 'PPPp', { locale: locale })}</TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
