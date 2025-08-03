import { QueryClient } from '@tanstack/react-query'
import { Outlet, createRootRouteWithContext } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'

import NotFound from '@/components/placeholder/not-found'

import { RouterAuthState } from '@/hooks/use-auth'

interface RouterContext {
  queryClient: QueryClient
  auth: RouterAuthState
}

export const Route = createRootRouteWithContext<RouterContext>()({
  component: () => (
    <>
      <Outlet />
      {import.meta.env.VITE_TANSTACK_ROUTER_DEVTOOLS === 'true' && (
        <TanStackRouterDevtools position="bottom-left" initialIsOpen={false} />
      )}
    </>
  ),
  notFoundComponent: () => <NotFound />,
})
