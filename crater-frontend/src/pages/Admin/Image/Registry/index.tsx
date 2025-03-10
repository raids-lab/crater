import KanikoDetail from "@/pages/Portal/Image/Info";
import { KanikoListTable } from "@/pages/Portal/Image/Registry";
import {
  apiAdminDeleteKanikoList,
  apiAdminListKaniko,
} from "@/services/api/admin/imagepack";
import { type FC } from "react";
import { useRoutes } from "react-router-dom";

export const Component: FC = () => {
  const routes = useRoutes([
    {
      index: true,
      element: (
        <KanikoListTable
          apiListKaniko={apiAdminListKaniko}
          apiDeleteKanikoList={apiAdminDeleteKanikoList}
          isAdminMode={true}
        />
      ),
    },
    {
      path: ":id",
      element: <KanikoDetail />,
    },
  ]);

  return <>{routes}</>;
};
