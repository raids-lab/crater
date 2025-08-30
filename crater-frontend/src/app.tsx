import { QueryClient } from '@tanstack/react-query'
import { RouterProvider } from '@tanstack/react-router'
import { getDefaultStore } from 'jotai'

import { AuthProvider, useAuth } from '@/hooks/use-auth'

import { router } from './router'

const store = getDefaultStore()

function InnerApp({ queryClient }: { queryClient: QueryClient }) {
  const auth = useAuth()
  return <RouterProvider router={router} context={{ auth, queryClient, store }} />
}

function App({ queryClient }: { queryClient: QueryClient }) {
  return (
    <AuthProvider>
      <InnerApp queryClient={queryClient} />
    </AuthProvider>
  )
}

export default App
