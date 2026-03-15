import { createFileRoute } from '@tanstack/react-router'

import { HealthOverview } from '@/components/aiops/HealthOverview'

export const Route = createFileRoute('/portal/aiops/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <HealthOverview />
}
