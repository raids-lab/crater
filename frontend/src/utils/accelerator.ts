export interface AcceleratorInfo {
  vendor: string
  model: string
}

export interface AcceleratorVendorStyle {
  badgeClassName: string
  fillColor: string
}

const vendorStyles: Record<string, AcceleratorVendorStyle> = {
  'nvidia.com': {
    badgeClassName: 'bg-[#75a031] ring-[#75a031] text-[#75a031]',
    fillColor: '#75a031',
  },
  'huawei.com': {
    badgeClassName: 'bg-[#be2a34] ring-[#be2a34] text-[#be2a34]',
    fillColor: '#be2a34',
  },
  'cambricon.com': {
    badgeClassName: 'bg-[#0052D9] ring-[#0052D9] text-[#0052D9]',
    fillColor: '#0052D9',
  },
  'iluvatar.com': {
    badgeClassName: 'bg-[#1e40af] ring-[#1e40af] text-[#1e40af]',
    fillColor: '#1e40af',
  },
  'amd.com': {
    badgeClassName: 'bg-red-500 ring-red-500 text-white',
    fillColor: '#ef4444',
  },
  'intel.com': {
    badgeClassName: 'bg-blue-600 ring-blue-600 text-white',
    fillColor: '#2563eb',
  },
  'qualcomm.com': {
    badgeClassName: 'bg-blue-500 ring-blue-500 text-white',
    fillColor: '#3b82f6',
  },
  'broadcom.com': {
    badgeClassName: 'bg-orange-600 ring-orange-600 text-white',
    fillColor: '#ea580c',
  },
  'xilinx.com': {
    badgeClassName: 'bg-purple-600 ring-purple-600 text-purple-600',
    fillColor: '#9333ea',
  },
  default: {
    badgeClassName: 'bg-gray-600 ring-gray-600 text-gray-600',
    fillColor: '#4b5563',
  },
}

export function parseAcceleratorString(input: string): AcceleratorInfo {
  if (!input) {
    return { vendor: '', model: '' }
  }

  const parts = input.split('/')
  if (parts.length === 2) {
    return { vendor: parts[0], model: parts[1] }
  }

  return { vendor: '', model: input }
}

export function getAcceleratorVendorStyle(vendor: string): AcceleratorVendorStyle {
  return vendorStyles[vendor] || vendorStyles.default
}

export function getAcceleratorVendorLabel(vendor: string): string {
  return vendor.replace('.com', '')
}
