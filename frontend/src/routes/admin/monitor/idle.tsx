import { createFileRoute } from '@tanstack/react-router'

import ResourseOverview from '@/components/monitors/idle-monitor'

export const Route = createFileRoute('/admin/monitor/idle')({
  component: RouteComponent,
})

function RouteComponent() {
  return <ResourseOverview />
}
