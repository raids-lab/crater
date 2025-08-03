import { createFileRoute } from '@tanstack/react-router'

import NetworkOverview from '@/components/monitors/network-monitor'

export const Route = createFileRoute('/admin/monitor/network')({
  component: RouteComponent,
})

function RouteComponent() {
  return <NetworkOverview />
}
