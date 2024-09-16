import {
  apiAdminCancelShareWithQueue,
  apiAdminCancelShareWithUser,
  apiAdminShareDatasetwithQueue,
  apiAdminShareDatasetwithUser,
} from "@/services/api/dataset";
import { DatasetShareTable } from "@/components/custom/DatasetShareTable";

export const Component = () => {
  return (
    <DatasetShareTable
      apiShareDatasetwithQueue={apiAdminShareDatasetwithQueue}
      apiShareDatasetwithUser={apiAdminShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiAdminCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiAdminCancelShareWithUser}
    />
  );
};
