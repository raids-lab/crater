import { useRoutes } from "react-router-dom";
import JupyterDetail from "./Detail";
import JupyterOverview from "./Overview";
import JupyterNew from "./New";

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <JupyterOverview />,
    },
    {
      path: "new",
      element: <JupyterNew />,
    },
    {
      path: ":id",
      element: <JupyterDetail />,
    },
  ]);

  return <>{routes}</>;
};
