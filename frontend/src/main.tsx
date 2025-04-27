import React from "react";
import ReactDOM from "react-dom/client";
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
} from "react-router-dom";
import "./index.css";
import Login from "./pages/Auth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import { portalRoute } from "./pages/Portal";
import { Toaster } from "@/components/ui-custom/sonner";
import { ThemeProvider } from "./utils/theme";
import { store, VITE_UI_THEME_KEY } from "./utils/store";
import { adminRoute } from "./pages/Admin";
import Jupyter from "./pages/Embed/IframeJupyter";
import { logger } from "./utils/loglevel";
import Website from "./pages/Website";
import { getDefaultStore, Provider as JotaiProvider } from "jotai";
import NotFound from "./components/layout/NotFound";
import { asyncDocsAsHomeAtom } from "./utils/store/config";

const defaultStore = getDefaultStore();

const docsAsHome = await defaultStore.get(asyncDocsAsHomeAtom);

const router = createBrowserRouter([
  {
    path: "/login",
    element: <Login />,
  },
  portalRoute,
  adminRoute,
  {
    path: "/job/jupyter/:id",
    element: <Jupyter />,
  },
  {
    path: "/",
    element: <Navigate to={docsAsHome ? "/website" : "/portal"} replace />,
  },
  {
    path: "/website",
    element: <Website />,
  },
  {
    path: "*",
    element: <NotFound />,
  },
]);

// TypeError: Failed to fetch dynamically imported module
// https://github.com/vitejs/vite/issues/11804
// https://vitejs.dev/guide/build#load-error-handling
window.addEventListener("vite:preloadError", () => {
  logger.info("vite:preloadError");
  window.location.reload(); // for example, refresh the page
});

const queryClient = new QueryClient();

async function enableMocking() {
  // Enable mocking in development when VITE_USE_MSW is true
  if (
    process.env.NODE_ENV !== "development" ||
    import.meta.env.VITE_USE_MSW !== "true"
  ) {
    return;
  }

  const { worker } = await import("./mocks/browser");

  // `worker.start()` returns a Promise that resolves
  // once the Service Worker is up and ready to intercept requests.
  return worker.start();
}

enableMocking()
  .then(() => {
    ReactDOM.createRoot(document.getElementById("root")!).render(
      <React.StrictMode>
        <JotaiProvider store={store}>
          <ThemeProvider storageKey={VITE_UI_THEME_KEY}>
            <QueryClientProvider client={queryClient}>
              <RouterProvider router={router} />
              <Toaster richColors closeButton />
              <ReactQueryDevtools
                initialIsOpen={false}
                buttonPosition="bottom-left"
              />
            </QueryClientProvider>
          </ThemeProvider>
        </JotaiProvider>
      </React.StrictMode>,
    );
  })
  .catch((err) => {
    logger.error(err);
  });
