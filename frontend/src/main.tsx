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
import DashboardLayout from "./pages/Dashboard/admin/Layout";
import { useAuth } from "./hooks/useAuth";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";

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
    path: "/",
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
        lazy: () => import("./pages/Dashboard/admin/Cluster"),
      },
      {
        path: "dashboard",
        lazy: () => import("./pages/Dashboard/admin/Cluster"),
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
]);

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <RecoilRoot>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
      </QueryClientProvider>
    </RecoilRoot>
  </React.StrictMode>,
);
