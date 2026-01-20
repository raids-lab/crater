import { format } from 'date-fns'
import { Calendar as CalendarIcon, Check } from 'lucide-react'
import * as React from 'react'
import { DateRange } from 'react-day-picker'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Calendar } from '@/components/ui/calendar'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'

import { cn } from '@/lib/utils'

interface DateRangePickerProps {
  className?: string
  value?: DateRange
  onUpdate: (range: DateRange | undefined) => void
}

export function DateRangePicker({ className, value, onUpdate }: DateRangePickerProps) {
  const { t } = useTranslation()
  const [date, setDate] = React.useState<DateRange | undefined>(value)
  const [isOpen, setIsOpen] = React.useState(false)

  // 当外部 value 改变时同步内部状态（例如切换预设时）
  React.useEffect(() => {
    setDate(value)
  }, [value])

  const handleApply = () => {
    onUpdate(date)
    setIsOpen(false)
  }

  return (
    <div className={cn('grid gap-2', className)}>
      <Popover open={isOpen} onOpenChange={setIsOpen}>
        <PopoverTrigger asChild>
          <Button
            id="date"
            variant={'outline'}
            size="sm"
            className={cn(
              'h-9 w-[260px] justify-start text-left font-normal',
              !date && 'text-muted-foreground'
            )}
          >
            <CalendarIcon className="mr-2 h-4 w-4" />
            {date?.from ? (
              date.to ? (
                <>
                  {format(date.from, 'LLL dd, y')} - {format(date.to, 'LLL dd, y')}
                </>
              ) : (
                format(date.from, 'LLL dd, y')
              )
            ) : (
              <span>{t('common.pickDate', { defaultValue: 'Pick a date' })}</span>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-auto p-0" align="end">
          <Calendar
            initialFocus
            mode="range"
            defaultMonth={date?.from}
            selected={date}
            onSelect={setDate}
            numberOfMonths={2}
          />
          <div className="flex items-center justify-end gap-2 border-t p-3">
            <Button
              variant="ghost"
              size="sm"
              onClick={() => {
                setDate(value) // 还原
                setIsOpen(false)
              }}
            >
              {t('common.cancel', { defaultValue: 'Cancel' })}
            </Button>
            <Button size="sm" disabled={!date?.from || !date?.to} onClick={handleApply}>
              <Check className="mr-2 h-4 w-4" />
              {t('common.apply', { defaultValue: 'Apply' })}
            </Button>
          </div>
        </PopoverContent>
      </Popover>
    </div>
  )
}
