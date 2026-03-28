import { createFileRoute } from '@tanstack/react-router'

import { HealthOverviewAdmin } from '@/components/aiops/HealthOverviewAdmin'

export const Route = createFileRoute('/admin/aiops/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <HealthOverviewAdmin />
}
