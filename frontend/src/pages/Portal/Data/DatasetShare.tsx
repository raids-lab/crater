import {
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
} from "@/services/api/dataset";
import { DatasetShareTable } from "@/components/custom/DatasetShareTable";

export const Component = () => {
  return (
    <DatasetShareTable
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
    />
  );
};
