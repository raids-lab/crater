// i18n-processed-v1.1.0 (no translatable strings)
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
  resourceType: "model" | "dataset" | "sharefile";
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

export function ModelShareTable(props: DatesetShareTableProps) {
  return <SharedResourceTable {...props} />;
}
