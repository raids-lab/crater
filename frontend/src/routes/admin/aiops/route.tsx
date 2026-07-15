import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'

import { Role } from '@/services/api/auth'

export const Route = createFileRoute('/admin/aiops')({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isAuthenticated || context.auth.context?.rolePlatform !== Role.Admin) {
      throw redirect({
        to: '/auth',
        search: {
          // Save current location for redirect after login
          redirect: location.href,
          token: '',
        },
      })
    }
    throw redirect({ to: '/admin' })
  },
  component: RouteComponent,
})

function RouteComponent() {
  return <Outlet />
}
