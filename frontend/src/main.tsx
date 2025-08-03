/**
 * Copyright 2025 RAIDS Lab
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { Provider as JotaiProvider } from 'jotai'
import React from 'react'
import ReactDOM from 'react-dom/client'

import { Toaster } from '@/components/ui-custom/sonner'

import App from './app'
import './i18n'
import './index.css'
import { logger } from './utils/loglevel'
import { VITE_UI_THEME_KEY, store } from './utils/store'
import { ThemeProvider } from './utils/theme'

// TypeError: Failed to fetch dynamically imported module
// https://github.com/vitejs/vite/issues/11804
// https://vitejs.dev/guide/build#load-error-handling
window.addEventListener('vite:preloadError', () => {
  logger.info('vite:preloadError')
  window.location.reload() // for example, refresh the page
})

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 1000,
    },
  },
})

async function enableMocking() {
  // Enable mocking in development when VITE_USE_MSW is true
  if (process.env.NODE_ENV !== 'development' || import.meta.env.VITE_USE_MSW !== 'true') {
    return
  }

  const { worker } = await import('./mocks/browser')

  // `worker.start()` returns a Promise that resolves
  // once the Service Worker is up and ready to intercept requests.
  return worker.start()
}

enableMocking()
  .then(() => {
    ReactDOM.createRoot(document.getElementById('root')!).render(
      <React.StrictMode>
        <JotaiProvider store={store}>
          <ThemeProvider storageKey={VITE_UI_THEME_KEY}>
            <QueryClientProvider client={queryClient}>
              <App queryClient={queryClient} />
              <Toaster richColors closeButton />
              {import.meta.env.VITE_TANSTACK_QUERY_DEVTOOLS === 'true' && (
                <ReactQueryDevtools initialIsOpen={false} buttonPosition="top-right" />
              )}
            </QueryClientProvider>
          </ThemeProvider>
        </JotaiProvider>
      </React.StrictMode>
    )
  })
  .catch((err) => {
    logger.error(err)
  })
