import { Navigate, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/portal/jobs/new/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <Navigate to="/portal/jobs/custom" />
}
