import React, { FC, PropsWithChildren, Suspense } from "react";
import ReactDOM from "react-dom/client";
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  RouterProvider,
} from "react-router-dom";
import "./index.css";
import { RecoilRoot } from "recoil";
import Login from "./pages/Login";
import { useAuth } from "./hooks/useAuth";

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
          <Outlet />
        </AuthedRouter>
      </Suspense>
    ),
    children: [
      {
        index: true,
        lazy: () => import("./pages/Home"),
      },
      {
        path: "dashboard",
        lazy: () => import("./pages/Home"),
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
      <RouterProvider router={router} />
    </RecoilRoot>
  </React.StrictMode>,
);
