'use client'

import { cn } from '@/lib/utils'

interface AcceleratorInfo {
  vendor: string
  model: string
}

// 厂商颜色映射表
const vendorColors: Record<string, string> = {
  'nvidia.com': 'bg-[#75a031] ring-[#75a031] text-[#75a031]', // NVIDIA 绿色
  'huawei.com': 'bg-[#be2a34] ring-[#be2a34] text-[#be2a34]', // 华为红色
  'amd.com': 'bg-red-500 ring-red-500 text-white', // AMD 红色
  'intel.com': 'bg-blue-600 ring-blue-600 text-white', // Intel 蓝色
  'qualcomm.com': 'bg-blue-500 ring-blue-500 text-white', // 高通蓝色
  'broadcom.com': 'bg-orange-600 ring-orange-600 text-white', // 博通橙色
  'xilinx.com': 'bg-purple-600 text-white', // Xilinx 紫色
  default: 'bg-gray-600 text-white', // 默认颜色
}

function parseAcceleratorString(input: string): AcceleratorInfo {
  if (!input) return { vendor: '', model: '' }

  const parts = input.split('/')
  if (parts.length === 2) {
    return { vendor: parts[0], model: parts[1] }
  }
  return { vendor: '', model: input }
}

function getVendorColor(vendor: string): string {
  return vendorColors[vendor] || vendorColors.default
}

interface DualColorBadgeProps {
  acceleratorString: string
  className?: string
}

export default function AcceleratorBadge({ acceleratorString }: DualColorBadgeProps) {
  const { vendor, model } = parseAcceleratorString(acceleratorString)

  return (
    <div
      className={cn(
        'relative inline-flex h-5 items-center overflow-hidden rounded-xs ring-1',
        getVendorColor(vendor)
      )}
    >
      {vendor && (
        <span
          className={cn(
            getVendorColor(vendor),
            'relative py-1 pr-1.5 pl-2 text-xs font-medium text-white uppercase'
          )}
        >
          {vendor.replace('.com', '')}
        </span>
      )}
      {model && (
        <span
          className="bg-card dark:text-foreground px-2 py-1 text-xs font-medium"
          style={{
            clipPath: vendor ? 'polygon(4px 0, 100% 0, 100% 100%, 0 100%)' : 'none',
          }}
        >
          {model.toUpperCase()}
        </span>
      )}
    </div>
  )
}
