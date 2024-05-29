import instance, { VERSION } from "../axios";
import { IResponse } from "@/services/types";

export interface Dataset {
  ID: number;
  Name: string;
  URL: string;
  Describe: string;
  UserID: string;
}

export interface UserDataset {
  datasetID: number;
  userID: number;
}

export interface QueueDataset {
  queueID: number;
  userID: number;
}

export interface DatasetReq {
  describe: string;
  name: string;
  url: string;
}
export const apiGetDataset = () =>
  instance.get<IResponse<Dataset[]>>(VERSION + `/dataset/mydataset`);

export const apiShareDatasetwithUser = (ud: UserDataset) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/share/user", ud);

export const apiShareDatasetwithQueue = (qd: QueueDataset) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/share/queue", qd);

export const apiDatasetCreate = (dataset: DatasetReq) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/create", dataset);

export const apiDatasetDelete = (datasetID: number) =>
  instance.delete<IResponse<string>>(VERSION + `/admin/projects/${datasetID}`);
