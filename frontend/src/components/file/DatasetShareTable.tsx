import { AxiosResponse } from "axios";
import { IResponse } from "@/services/types";
import {
  UserDataset,
  QueueDataset,
  cancelSharedUserResp,
  cancelSharedQueueResp,
} from "@/services/api/dataset";
import { SharedResourceTable } from "./SharedResourceTable";

interface DatesetShareTableProps {
  apiShareDatasetwithUser: (
    ud: UserDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiShareDatasetwithQueue: (
    qd: QueueDataset,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiCancelDatasetSharewithUser: (
    csu: cancelSharedUserResp,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiCancelDatasetSharewithQueue: (
    csq: cancelSharedQueueResp,
  ) => Promise<AxiosResponse<IResponse<string>>>;
  apiDatasetDelete: (
    datasetID: number,
  ) => Promise<AxiosResponse<IResponse<string>>>;
}

export function DatasetShareTable(props: DatesetShareTableProps) {
  return <SharedResourceTable resourceType="dataset" {...props} />;
}
