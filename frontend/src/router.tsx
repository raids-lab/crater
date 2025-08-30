import { createRouter } from '@tanstack/react-router'

// Import the generated route tree
import { routeTree } from './routeTree.gen'

export const router = createRouter({
  routeTree,
  context: {
    // will be passed down from App component
    auth: undefined!,
    queryClient: undefined!,
    store: undefined!,
  },
})

declare module '@tanstack/react-router' {
  interface Register {
    router: typeof router
  }
}
