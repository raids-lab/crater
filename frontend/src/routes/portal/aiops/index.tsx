import { createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/portal/aiops/')({
  beforeLoad: () => {
    throw redirect({ to: '/portal' })
  },
  component: RouteComponent,
})

function RouteComponent() {
  return null
}

/*
 * AI 运维独立页面暂时下线：保留主体代码，后续如需恢复可还原。
 *
 * import { t } from 'i18next'
 * import { HealthOverview } from '@/components/aiops/HealthOverview'
 *
 * export const Route = createFileRoute('/portal/aiops/')({
 *   component: RouteComponent,
 *   loader: () => ({ crumb: t('navigation.aiops') }),
 * })
 *
 * function RouteComponent() {
 *   return <HealthOverview />
 * }
 */
