import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'

export const Route = createFileRoute('/portal/aiops')({
  beforeLoad: ({ context, location }) => {
    if (!context.auth.isAuthenticated) {
      throw redirect({
        to: '/auth',
        search: {
          // Save current location for redirect after login
          redirect: location.href,
          token: '',
        },
      })
    }
  },
  component: RouteComponent,
})

function RouteComponent() {
  return <Outlet />
}
