import { createFileRoute } from '@tanstack/react-router'

import AdminJobOverview from '../../../components/job/overview/admin-jobs'

export const Route = createFileRoute('/admin/jobs/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <AdminJobOverview />
}
