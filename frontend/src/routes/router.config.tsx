import Login from "@/pages/Login";
import { FC, Suspense, lazy } from "react";
import { Navigate, RouteObject } from "react-router-dom";

/**
 * Lazy loading components.
 * @example const Component = React.lazy(() => import('./Component'));
 */
const HomeLazy = lazy(() => import("@/pages/Home"));

/**
 * Router elements
 */
const Home: FC = () => {
  return (
    <Suspense fallback={<h2>Loading...</h2>}>
      <HomeLazy />
    </Suspense>
  );
};

/**
 * Router config
 * @see {@link https://reactrouter.com/docs/en/v6/hooks/use-routes}
 */
export const routerConfig: RouteObject[] = [
  {
    path: "/",
    element: <Login />,
  },
  {
    path: "/dashboard",
    element: <Home />,
  },
  {
    path: "/admin",
    loader: () => {
      throw new Response("Not Found", { status: 404 });

      return { status: "success" };
    },
    errorElement: <div>401 Unauth</div>,
    children: [
      {
        path: "dashboard",
        element: <Home />,
      },
      {
        path: "settings",
        element: <div>Settings</div>,
      },
    ],
  },
  {
    path: "*",
    element: <Navigate to="/" replace />,
  },
];

export default routerConfig;
