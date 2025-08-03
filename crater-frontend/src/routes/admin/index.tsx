import { Navigate, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/admin/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <Navigate to="/admin/cluster/nodes" replace />
}
