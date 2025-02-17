import {
  apiAdminCancelShareWithQueue,
  apiAdminCancelShareWithUser,
  apiAdminShareDatasetwithQueue,
  apiAdminShareDatasetwithUser,
  apiDatasetDelete,
} from "@/services/api/dataset";
import { DatasetShareTable } from "@/components/file/DatasetShareTable";

export const Component = () => {
  return (
    <DatasetShareTable
      apiShareDatasetwithQueue={apiAdminShareDatasetwithQueue}
      apiShareDatasetwithUser={apiAdminShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiAdminCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiAdminCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
    />
  );
};
