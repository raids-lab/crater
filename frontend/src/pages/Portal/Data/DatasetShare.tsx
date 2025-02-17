import {
  apiCancelShareWithQueue,
  apiCancelShareWithUser,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
  apiDatasetDelete,
} from "@/services/api/dataset";
import { DatasetShareTable } from "@/components/file/DatasetShareTable";

export const Component = () => {
  return (
    <DatasetShareTable
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
    />
  );
};
