import { createFileRoute } from '@tanstack/react-router'
import { t } from 'i18next'

import { HealthOverview } from '@/components/aiops/HealthOverview'

export const Route = createFileRoute('/portal/aiops/')({
  component: RouteComponent,
  loader: () => ({ crumb: t('navigation.aiops') }),
})

function RouteComponent() {
  return <HealthOverview />
}
