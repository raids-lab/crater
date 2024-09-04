import { type FC } from "react";
import {
  apiAdminGetDataset,
  apiAdminShareDatasetwithQueue,
  apiAdminShareDatasetwithUser,
} from "@/services/api/dataset";

import { DatasetTable } from "@/components/custom/DatasetTable";

export const Component: FC = () => {
  return (
    <DatasetTable
      apiGetDataset={apiAdminGetDataset}
      apiShareDatasetwithUser={apiAdminShareDatasetwithUser}
      apiShareDatasetwithQueue={apiAdminShareDatasetwithQueue}
    />
  );
};
