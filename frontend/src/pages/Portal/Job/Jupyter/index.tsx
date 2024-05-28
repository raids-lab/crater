import { useRoutes } from "react-router-dom";
import JupyterDetail from "./Detail";
import JupyterOverview from "./Overview";
import JupyterNew from "../New/Jupyter";

export const Component = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <JupyterOverview />,
    },
    {
      path: "new-jupyter",
      element: <JupyterNew />,
    },
    {
      path: ":id",
      element: <JupyterDetail />,
    },
  ]);

  return <>{routes}</>;
};
