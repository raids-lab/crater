'use client'

import {
  getAcceleratorVendorLabel,
  getAcceleratorVendorStyle,
  parseAcceleratorString,
} from '@/utils/accelerator'

import { cn } from '@/lib/utils'

interface DualColorBadgeProps {
  acceleratorString: string
  className?: string
}

export default function AcceleratorBadge({ acceleratorString, className }: DualColorBadgeProps) {
  const { vendor, model } = parseAcceleratorString(acceleratorString)
  const { badgeClassName } = getAcceleratorVendorStyle(vendor)

  return (
    <div
      className={cn(
        'relative inline-flex h-5 min-w-0 items-center overflow-hidden rounded-xs ring-1',
        badgeClassName,
        className
      )}
    >
      {vendor && (
        <span
          className={cn(
            badgeClassName,
            'relative shrink-0 py-1 pr-1.5 pl-2 text-[11px] font-medium text-white uppercase'
          )}
        >
          {getAcceleratorVendorLabel(vendor)}
        </span>
      )}
      {model && (
        <span
          className="bg-card dark:text-foreground min-w-0 flex-1 truncate px-2 py-1 text-[11px] font-medium"
          style={{
            clipPath: vendor ? 'polygon(4px 0, 100% 0, 100% 100%, 0 100%)' : 'none',
          }}
          title={model.toUpperCase()}
        >
          {model.toUpperCase()}
        </span>
      )}
    </div>
  )
}
