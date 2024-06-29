import type { FC } from "react";
import {
  apiGetDataset,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
} from "@/services/api/dataset";

import { DatasetTable } from "@/components/custom/DatasetTable";

export const Component: FC = () => {
  return (
    <DatasetTable
      apiGetDataset={apiGetDataset}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
    />
  );
};
