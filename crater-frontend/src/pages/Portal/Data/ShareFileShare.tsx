import { ModelShareTable } from "@/components/file/ModelShareTable";
import {
  apiCancelShareWithQueue,
  apiCancelShareWithUser,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
  apiDatasetDelete,
} from "@/services/api/dataset";

export function Component() {
  return (
    <ModelShareTable
      resourceType="sharefile"
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiCancelShareWithUser}
      apiDatasetDelete={apiDatasetDelete}
    />
  );
}
