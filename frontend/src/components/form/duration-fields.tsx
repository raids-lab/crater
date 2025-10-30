import { addHours, format } from 'date-fns'
import { zhCN } from 'date-fns/locale'
import { CalendarClock } from 'lucide-react'
import { useEffect, useMemo, useRef, useState } from 'react'

import { Card, CardContent } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'

type DurationValue = {
  days: number
  hours: number
  totalHours: number
}

interface DurationFieldsProps {
  value?: { days?: number; hours?: number }
  onChange?: (val: DurationValue) => void
  // 预览的起点时间（不传则从当前时间开始）
  origin?: Date | string | null
  // 用于前端校验显示（不强制阻止输入），父组件可再次校验
  maxTotalHours?: number
  showPreview?: boolean
  className?: string
}

export function DurationFields({
  value,
  onChange,
  origin,
  maxTotalHours = 0,
  showPreview = true,
  className,
}: DurationFieldsProps) {
  const [days, setDays] = useState<number>(value?.days ?? 0)
  const [hours, setHours] = useState<number>(value?.hours ?? 0)

  // 规范输入：天>=0&&<=4，小时0-23
  const normalized = useMemo(() => {
    let d = Number.isFinite(days) && days > 0 ? Math.floor(days) : 0
    let h = Number.isFinite(hours) && hours > 0 ? Math.floor(hours) : 0
    if (d < 0) d = 0
    if (d > 4) d = 4
    if (h > 23) h = 23
    if (h < 0) h = 0
    return { days: d, hours: h, totalHours: d * 24 + h }
  }, [days, hours])

  // 预览到期时间
  const preview = useMemo(() => {
    if (!showPreview) return null
    if (normalized.totalHours <= 0) return null
    const base =
      typeof origin === 'string' ? new Date(origin) : origin instanceof Date ? origin : new Date()
    return addHours(base, normalized.totalHours)
  }, [normalized.totalHours, origin, showPreview])

  // 同步外部变更
  useEffect(() => {
    if (value) {
      if (typeof value.days === 'number') setDays(value.days)
      if (typeof value.hours === 'number') setHours(value.hours)
    }
  }, [value])

  // 保持 onChange 稳定引用，避免因父组件 inline 函数导致无限循环
  const onChangeRef = useRef<typeof onChange>(undefined)
  useEffect(() => {
    onChangeRef.current = onChange
  }, [onChange])

  // 派发变更：仅依赖 normalized
  useEffect(() => {
    onChangeRef.current?.(normalized)
  }, [normalized])

  return (
    <div className={className}>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <Label htmlFor="duration-days" className="mb-2 block">
            天数
          </Label>
          <Input
            id="duration-days"
            type="number"
            min="0"
            max="4"
            placeholder="0"
            value={days}
            onChange={(e) => setDays(Number(e.target.value))}
            onWheel={(e) => {
              e.preventDefault()
              const change = e.deltaY < 0 ? 1 : -1
              setDays((prev) => {
                const newValue = prev + change
                if (newValue >= 0 && newValue <= 4) {
                  return newValue
                }
                return prev
              })
            }}
          />
        </div>
        <div>
          <Label htmlFor="duration-hours" className="mb-2 block">
            小时数
          </Label>
          <Input
            id="duration-hours"
            type="number"
            min="0"
            max="23"
            placeholder="0"
            value={hours}
            onChange={(e) => setHours(Number(e.target.value))}
            onWheel={(e) => {
              e.preventDefault()
              const change = e.deltaY < 0 ? 1 : -1
              setHours((prev) => {
                const newValue = prev + change
                if (newValue >= 0 && newValue <= 23) {
                  return newValue
                }
                return prev
              })
            }}
          />
        </div>
      </div>

      <div className="text-muted-foreground mt-4 text-xs">
        {maxTotalHours ? `（上限建议：${maxTotalHours} 小时）` : null}
      </div>

      {preview && (
        <Card className="mt-5 border-dashed">
          <CardContent className="pt-4">
            <div className="text-muted-foreground flex items-center gap-2">
              <CalendarClock className="h-4 w-4" />
              <span>到期时间预览</span>
            </div>
            <div className="mt-2">
              <p className="text-lg font-medium">
                {format(preview, 'yyyy年MM月dd日 HH:mm', { locale: zhCN })}
              </p>
              <p className="text-muted-foreground mt-1 text-sm">
                当前时间：{format(new Date(), 'yyyy年MM月dd日 HH:mm', { locale: zhCN })}
              </p>
            </div>
          </CardContent>
        </Card>
      )}
    </div>
  )
}
