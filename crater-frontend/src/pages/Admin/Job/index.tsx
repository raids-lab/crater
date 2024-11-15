import { useRoutes } from "react-router-dom";
import AdminJobOverview from "./Overview";
import { Component } from "../../Portal/Job/Detail/Base";

const AdminJob = () => {
  const routes = useRoutes([
    {
      index: true,
      element: <AdminJobOverview />,
    },
    {
      path: ":name",
      element: <Component />,
    },
  ]);

  return <>{routes}</>;
};

export default AdminJob;
