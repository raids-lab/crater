/* eslint-disable react-refresh/only-export-components */
import { FC, Suspense, lazy } from "react";
import { Navigate, RouteObject } from "react-router-dom";

/**
 * Lazy loading components.
 * @example const Component = React.lazy(() => import('./Component'));
 */
const HomeLazy = lazy(() => import("@/App"));

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
    path: "/app",
    element: <Home />,
  },
  {
    path: "*",
    element: <Navigate to="/app" replace />,
  },
];

export default routerConfig;
