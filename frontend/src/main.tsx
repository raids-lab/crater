import React, { FC, PropsWithChildren, Suspense } from "react";
import ReactDOM from "react-dom/client";
import {
  createBrowserRouter,
  Navigate,
  RouterProvider,
} from "react-router-dom";
import "./index.css";
import { RecoilRoot } from "recoil";
import Login from "./pages/Login";
import DashboardLayout from "./pages/Dashboard/Admin/Layout";
import { useAuth } from "./hooks/useAuth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";
import Signup from "./pages/Signup";

const queryClient = new QueryClient();

const AuthedRouter: FC<PropsWithChildren> = ({ children }) => {
  const isAuthenticated = useAuth("admin");
  return isAuthenticated ? children : <Navigate to="/login" replace />;
};

const router = createBrowserRouter([
  {
    path: "/login",
    element: <Login />,
  },
  {
    path: "/signup",
    element: <Signup />,
  },
  {
    path: "/dashboard",
    element: (
      <Suspense fallback={<h2>Loading...</h2>}>
        <AuthedRouter>
          <DashboardLayout />
        </AuthedRouter>
      </Suspense>
    ),
    children: [
      {
        index: true,
        lazy: () => import("./pages/Dashboard/Admin/Cluster"),
      },
      {
        path: "cluster",
        lazy: () => import("./pages/Dashboard/Admin/Cluster"),
      },
      {
        path: "task",
        lazy: () => import("./pages/Dashboard/Admin/Training"),
      },
      {
        path: "ai_task",
        lazy: () => import("./pages/Dashboard/Admin/Training"),
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/dashboard" replace />,
  },
]);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RecoilRoot>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
        <ReactQueryDevtools initialIsOpen={false} />
      </QueryClientProvider>
    </RecoilRoot>
  </React.StrictMode>,
);
