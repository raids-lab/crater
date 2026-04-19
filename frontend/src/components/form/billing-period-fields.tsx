import { useEffect, useMemo, useRef, useState } from 'react'

import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

export type BillingPeriodValue = {
  days: number
  hours: number
  minutes: number
  totalMinutes: number
}

export const billingPeriodFromMinutes = (totalMinutes?: number): BillingPeriodValue => {
  const minutes = Math.max(0, Math.floor(totalMinutes ?? 0))
  const days = Math.floor(minutes / (24 * 60))
  const remainAfterDays = minutes % (24 * 60)
  const hours = Math.floor(remainAfterDays / 60)
  const mins = remainAfterDays % 60
  return { days, hours, minutes: mins, totalMinutes: minutes }
}

export interface BillingPeriodFieldsProps {
  totalMinutes?: number
  onChange?: (value: BillingPeriodValue) => void
  disabled?: boolean
  className?: string
}

export function BillingPeriodFields({
  totalMinutes,
  onChange,
  disabled,
  className,
}: BillingPeriodFieldsProps) {
  const initial = billingPeriodFromMinutes(totalMinutes)
  const [days, setDays] = useState<number>(initial.days)
  const [hours, setHours] = useState<number>(initial.hours)
  const [minutes, setMinutes] = useState<number>(initial.minutes)

  useEffect(() => {
    const next = billingPeriodFromMinutes(totalMinutes)
    setDays(next.days)
    setHours(next.hours)
    setMinutes(next.minutes)
  }, [totalMinutes])

  const normalized = useMemo(() => {
    const d = Math.max(0, Number.isFinite(days) ? Math.floor(days) : 0)
    const h = Math.min(23, Math.max(0, Number.isFinite(hours) ? Math.floor(hours) : 0))
    const m = Math.min(59, Math.max(0, Number.isFinite(minutes) ? Math.floor(minutes) : 0))
    return {
      days: d,
      hours: h,
      minutes: m,
      totalMinutes: d * 24 * 60 + h * 60 + m,
    }
  }, [days, hours, minutes])

  const didMountRef = useRef(false)
  const onChangeRef = useRef<typeof onChange>(undefined)
  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  useEffect(() => {
    if (!didMountRef.current) {
      didMountRef.current = true
      return
    }
    onChangeRef.current?.(normalized)
  }, [normalized])

  return (
    <div className={className}>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <div className="space-y-1">
          <Label>天</Label>
          <Input
            type="number"
            min={0}
            disabled={disabled}
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
          />
        </div>
        <div className="space-y-1">
          <Label>小时</Label>
          <Input
            type="number"
            min={0}
            max={23}
            disabled={disabled}
            value={hours}
            onChange={(e) => setHours(Number(e.target.value))}
          />
        </div>
        <div className="space-y-1">
          <Label>分钟</Label>
          <Input
            type="number"
            min={0}
            max={59}
            disabled={disabled}
            value={minutes}
            onChange={(e) => setMinutes(Number(e.target.value))}
          />
        </div>
      </div>
      <p className="text-muted-foreground mt-2 text-xs">共 {normalized.totalMinutes} 分钟</p>
    </div>
  )
}
