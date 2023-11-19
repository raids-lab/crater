import React from "react";
import ReactDOM from "react-dom/client";
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
} from "react-router-dom";
import "./index.css";
import { RecoilRoot } from "recoil";
import Login from "./pages/Login";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import Signup from "./pages/Signup";
import { portalRoute } from "./pages/Portal";
import { Toaster } from "@/components/ui/toaster";
import { ThemeProvider } from "./utils/theme";
import { VITE_UI_THEME_KEY } from "./utils/store";
import { adminRoute } from "./pages/Admin";
import { recommendRoute } from "./pages/PortalR";

const router = createBrowserRouter([
  {
    path: "/login",
    element: <Login />,
  },
  {
    path: "/signup",
    element: <Signup />,
  },
  portalRoute,
  recommendRoute,
  adminRoute,
  {
    path: "*",
    element: <Navigate to="/portal" replace />,
  },
]);

const queryClient = new QueryClient();

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RecoilRoot>
      <ThemeProvider defaultTheme="light" storageKey={VITE_UI_THEME_KEY}>
        <QueryClientProvider client={queryClient}>
          <RouterProvider router={router} />
          <Toaster />
          <ReactQueryDevtools initialIsOpen={false} />
        </QueryClientProvider>
      </ThemeProvider>
    </RecoilRoot>
  </React.StrictMode>,
);
