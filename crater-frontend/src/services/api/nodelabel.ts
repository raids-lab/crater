import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export enum LabelType {
  NVIDIA = 1,
}

export interface LabelInfo {
  id: number;
  label: string;
  name: string;
  type: LabelType;
  count: number;
  priority: number;
}

export const apiNodeLabelsList = () => {
  return instance.get<IResponse<LabelInfo[]>>(`${VERSION}/labels`);
};

export const apiNodeLabelsUpdate = (
  id: number,
  name: string,
  priority: number,
) => {
  return instance.put<IResponse<LabelInfo>>(`${VERSION}/admin/labels/${id}`, {
    name,
    priority,
  });
};

export const apiNodeLabelsNvidiaSync = () => {
  return instance.post<IResponse<never>>(`${VERSION}/admin/labels/sync/nvidia`);
};
