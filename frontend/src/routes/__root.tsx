import { QueryClient, queryOptions } from '@tanstack/react-query'
import { Outlet, createRootRouteWithContext } from '@tanstack/react-router'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { Store } from 'jotai/vanilla/store'

import NotFound from '@/components/placeholder/not-found'

import { RouterAuthState } from '@/hooks/use-auth'

import { logger } from '@/utils/loglevel'
import { apiGetConfig, configAtom } from '@/utils/store/config'

interface RouterContext {
  queryClient: QueryClient
  auth: RouterAuthState
  store: Store
}

const queryConfig = queryOptions({
  queryKey: ['appConfig'],
  queryFn: apiGetConfig,
})

export const Route = createRootRouteWithContext<RouterContext>()({
  loader: async ({ context }) => {
    const config = await context.queryClient.ensureQueryData(queryConfig)
    const prevConfig = context.store.get(configAtom)
    context.store.set(configAtom, config)
    const isEqual = JSON.stringify(prevConfig) === JSON.stringify(config)
    if (!isEqual) {
      logger.info('App config updated, reloading...')
      window.location.reload()
    }
  },
  component: () => (
    <>
      <Outlet />
      {import.meta.env.MODE === 'development' &&
        import.meta.env.VITE_TANSTACK_ROUTER_DEVTOOLS === 'true' && (
          <TanStackRouterDevtools position="bottom-left" initialIsOpen={false} />
        )}
    </>
  ),
  notFoundComponent: () => <NotFound />,
})
