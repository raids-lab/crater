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
// import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import Signup from "./pages/Signup";
import { dashboardRoute } from "./pages/Dashboard";
import { Toaster } from "@/components/ui/toaster";
import { ThemeProvider } from "./utils/theme";
import { VITE_UI_THEME_KEY } from "./utils/store";

const router = createBrowserRouter([
  {
    path: "/login",
    element: <Login />,
  },
  {
    path: "/signup",
    element: <Signup />,
  },
  dashboardRoute,
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />,
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
          {/* <ReactQueryDevtools initialIsOpen={false} position="bottom-right" /> */}
        </QueryClientProvider>
      </ThemeProvider>
    </RecoilRoot>
  </React.StrictMode>,
);
