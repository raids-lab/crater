import {
  apiCancelShareWithQueue,
  apiCancelShareWithUser,
  apiShareDatasetwithQueue,
  apiShareDatasetwithUser,
} from "@/services/api/dataset";
import { ModelShareTable } from "@/components/file/ModelShareTable";

export const Component = () => {
  return (
    <ModelShareTable
      apiShareDatasetwithQueue={apiShareDatasetwithQueue}
      apiShareDatasetwithUser={apiShareDatasetwithUser}
      apiCancelDatasetSharewithQueue={apiCancelShareWithQueue}
      apiCancelDatasetSharewithUser={apiCancelShareWithUser}
    />
  );
};
