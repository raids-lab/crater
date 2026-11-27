import { CalendarIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'

const recentDays = [7, 14, 30, 90]

export function JobDateRangeSelect({
  days,
  onChange,
  allowAll = false,
  size,
}: {
  days: number
  onChange: (days: number) => void
  allowAll?: boolean
  size?: 'sm' | 'default'
}) {
  const { t } = useTranslation()
  const options = allowAll ? [...recentDays, -1] : recentDays
  return (
    <Select value={String(days)} onValueChange={(value) => onChange(Number(value))}>
      <SelectTrigger className="bg-background" size={size} aria-label={t('jobs.select.range')}>
        <CalendarIcon />
        <SelectValue />
      </SelectTrigger>
      <SelectContent align="end">
        {options.map((option) => (
          <SelectItem key={option} value={String(option)}>
            {option === -1 ? t('jobs.select.all') : t('jobs.select.recentDays', { days: option })}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  )
}
