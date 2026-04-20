import { V1ResourceList, convertKResourceToResource } from './resource'

export interface BillingPriceSource {
  label: string
  unitPrice: number
}

export interface BillingPriceEntryInput {
  label?: string
  multiplier?: number
  resourceList?: V1ResourceList
}

export interface BillingResourceCostItem {
  name: string
  label: string
  amount: number
  unitPrice: number
  costPerHour: number
}

export interface BillingPriceEntrySummary {
  label?: string
  multiplier: number
  subtotalPerHour: number
  items: BillingResourceCostItem[]
}

export const formatBillingPoints = (value?: number) =>
  new Intl.NumberFormat('zh-CN', {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value ?? 0)

export const calcBillingAmount = (resourceName: string, rawValue?: string) => {
  if (!rawValue) {
    return 0
  }
  const normalized = convertKResourceToResource(resourceName, rawValue)
  if (normalized == null || Number.isNaN(normalized)) {
    return 0
  }
  return normalized
}

export const summarizeBillingPriceEntries = (
  entries: BillingPriceEntryInput[],
  priceSources: Record<string, BillingPriceSource>
): BillingPriceEntrySummary[] =>
  entries.map((entry) => {
    const multiplier = Math.max(1, entry.multiplier ?? 1)
    const resourceList = entry.resourceList ?? {}
    const items = Object.entries(resourceList)
      .map(([resourceName, rawValue]) => {
        const source = priceSources[resourceName]
        const unitPrice = source?.unitPrice ?? 0
        const amount = calcBillingAmount(resourceName, rawValue)
        const costPerHour = amount * unitPrice * multiplier
        return {
          name: resourceName,
          label: source?.label ?? resourceName,
          amount,
          unitPrice,
          costPerHour,
        }
      })
      .filter((item) => item.unitPrice > 0 && item.amount > 0)

    return {
      label: entry.label,
      multiplier,
      subtotalPerHour: items.reduce((sum, item) => sum + item.costPerHour, 0),
      items,
    }
  })
