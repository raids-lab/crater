import { IBillingStatus } from '@/services/api/system-config'

type BillingStatusLike = Pick<IBillingStatus, 'featureEnabled' | 'active'> | null | undefined

export function isBillingVisibleForAdmin(status: BillingStatusLike) {
  return Boolean(status?.featureEnabled)
}

export function isBillingVisibleForUser(status: BillingStatusLike) {
  return Boolean(status?.featureEnabled && status?.active)
}

export function isBillingVisible(status: BillingStatusLike, audience: 'admin' | 'user') {
  return audience === 'admin' ? isBillingVisibleForAdmin(status) : isBillingVisibleForUser(status)
}
