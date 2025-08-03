import { Navigate, createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/')({
  beforeLoad: () => {
    throw redirect({ to: '/portal' })
  },
  component: Index,
})

function Index() {
  return <Navigate to="/portal" replace />
}
