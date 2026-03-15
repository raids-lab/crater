import { Outlet, createFileRoute, redirect } from '@tanstack/react-router'

import { AIChatAssistantProvider } from '@/components/aiops/AIChatAssistantProvider'
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
  },
  component: RouteComponent,
})

function RouteComponent() {
  return (
    <AIChatAssistantProvider>
      <Outlet />
    </AIChatAssistantProvider>
  )
}
