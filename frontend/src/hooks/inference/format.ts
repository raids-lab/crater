import type { KthenaService } from '@/services/api/inference'

// Kubernetes quantities use decimal SI for CPU and binary or decimal SI for memory.
export function quantityValue(value: string): number {
  const match = /^([+-]?(?:\d+(?:\.\d*)?|\.\d+))([eE][+-]?\d+|[numkKMGTPE]i?)?$/.exec(value.trim())
  if (!match) throw new Error(`Invalid resource quantity: ${value}`)
  const suffix = match[2] || ''
  const powers: Record<string, number> = {
    n: 1e-9,
    u: 1e-6,
    m: 1e-3,
    k: 1e3,
    K: 1e3,
    M: 1e6,
    G: 1e9,
    T: 1e12,
    P: 1e15,
    E: 1e18,
    Ki: 1024,
    Mi: 1024 ** 2,
    Gi: 1024 ** 3,
    Ti: 1024 ** 4,
    Pi: 1024 ** 5,
    Ei: 1024 ** 6,
  }
  return (
    Number(match[1]) *
    (suffix ? (/^[eE][+-]?\d+$/.test(suffix) ? 10 ** Number(suffix.slice(1)) : powers[suffix]) : 1)
  )
}

export const shellQuote = (value: string) => "'" + value.replace(/'/g, "'\\''") + "'"

export function getServingResources(service: KthenaService) {
  return service.totalResources
}
