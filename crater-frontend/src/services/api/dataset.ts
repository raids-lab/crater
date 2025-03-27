import instance, { VERSION } from "../axios";
import { IResponse } from "@/services/types";
import { IUserAttributes } from "./admin/user";

export interface Extra {
  tag: string[];
  weburl: string;
}

export interface Dataset {
  id: number;
  name: string;
  url: string;
  describe: string;
  username: string;
  createdAt: string;
  type: string;
  extra: Extra;
  attribute: IUserAttributes;
}

export interface UserDataset {
  datasetID: number;
  userIDs: number[];
}

export interface QueueDataset {
  queueIDs: number[];
  datasetID: number;
}

export interface DatasetReq {
  describe: string;
  name: string;
  url: string;
  type: string;
  tags: string[];
  weburl: string;
  ispublic: boolean;
}

export interface DatasetReanmeReq {
  datasetID: number;
  name: string;
}

export interface UserDatasetResp {
  id: number;
  name: string;
  isowner: boolean;
}
export interface QueueDatasetGetResp {
  id: number;
  name: string;
}

export interface cancelSharedUserResp {
  datasetID: number;
  userID: number;
}
export interface cancelSharedQueueResp {
  datasetID: number;
  queueID: number;
}
export const apiGetDataset = () =>
  instance.get<IResponse<Dataset[]>>(VERSION + `/dataset/mydataset`);

//因为table表单的query必须要返回数组，实际上数组里只有一个数据集的数据
export const apiGetDatasetByID = (datasetID: number) =>
  instance.get<IResponse<Dataset[]>>(VERSION + `/dataset/detail/${datasetID}`);

export const apiAdminGetDataset = () =>
  instance.get<IResponse<Dataset[]>>(VERSION + `/admin/dataset/alldataset`);

export const apiShareDatasetwithUser = (ud: UserDataset) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/share/user", ud);

export const apiShareDatasetwithQueue = (qd: QueueDataset) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/share/queue", qd);

export const apiDatasetCreate = (dataset: DatasetReq) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/create", dataset);

export const apiDatasetDelete = (datasetID: number) =>
  instance.delete<IResponse<string>>(VERSION + `/dataset/delete/${datasetID}`);

export const apiDatasetRename = (drr: DatasetReanmeReq) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/rename", drr);

export const apiAdminShareDatasetwithUser = (ud: UserDataset) =>
  instance.post<IResponse<string>>(VERSION + "/admin/dataset/share/user", ud);

export const apiAdminShareDatasetwithQueue = (qd: QueueDataset) =>
  instance.post<IResponse<string>>(VERSION + "/admin/dataset/share/queue", qd);

export const apiListUsersNotInDataset = (datasetID: number) =>
  instance.get<IResponse<IUserAttributes[]>>(
    VERSION + `/dataset/${datasetID}/usersNotIn`,
  );

export const apiListQueuesNotInDataset = (datasetID: number) =>
  instance.get<IResponse<QueueDatasetGetResp[]>>(
    VERSION + `/dataset/${datasetID}/queuesNotIn`,
  );
export const apiListUsersInDataset = (datasetID: number) =>
  instance.get<IResponse<UserDatasetResp[]>>(
    VERSION + `/dataset/${datasetID}/usersIn`,
  );

export const apiListQueuesInDataset = (datasetID: number) =>
  instance.get<IResponse<QueueDatasetGetResp[]>>(
    VERSION + `/dataset/${datasetID}/queuesIn`,
  );

export const apiCancelShareWithUser = (CSU: cancelSharedUserResp) =>
  instance.post<IResponse<string>>(VERSION + `/dataset/cancelshare/user`, CSU);

export const apiCancelShareWithQueue = (CSQ: cancelSharedQueueResp) =>
  instance.post<IResponse<string>>(VERSION + `/dataset/cancelshare/queue`, CSQ);

export const apiAdminCancelShareWithUser = (CSU: cancelSharedUserResp) =>
  instance.post<IResponse<string>>(
    VERSION + `/admin/dataset/cancelshare/user`,
    CSU,
  );

export const apiAdminCancelShareWithQueue = (CSQ: cancelSharedQueueResp) =>
  instance.post<IResponse<string>>(
    VERSION + `/admin/dataset/cancelshare/queue`,
    CSQ,
  );
export const apiDatasetUpdate = (dataset: DatasetReq & { datasetId: number }) =>
  instance.post<IResponse<string>>(VERSION + "/dataset/update", dataset);
