import { Navigate, createFileRoute } from '@tanstack/react-router'

export const Route = createFileRoute('/portal/')({
  component: RouteComponent,
})

function RouteComponent() {
  return <Navigate to="/portal/overview" replace resetScroll />
}
