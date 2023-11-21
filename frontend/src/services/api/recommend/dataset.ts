import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";

export interface IDlDataset {
  name: string;
  namespace: string;
  generation: number;
  creationTimestamp: string;
  spec: {
    pvc: string;
    downloadUrl: string;
    size: number;
  };
  status: {
    phase: string;
  };
}

// v1/dataset/list
export const apiDlDatasetList = () =>
  instance.get<IResponse<IDlDataset[]>>(VERSION + "/dataset/list");

// v1/dataset/info?name=bert-dataset
export const apiDlDatasetInfo = (name: string) =>
  instance.get<IResponse<IDlDataset>>(VERSION + "/dataset/info", {
    params: {
      name,
    },
  });
