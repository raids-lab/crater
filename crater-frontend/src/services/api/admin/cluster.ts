import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";

export interface IClusterNodeInfo {
  name: string;
  role: string;
  labels: Record<string, string>;
  isReady: boolean;
  capacity: KubernetesResource;
  allocated: KubernetesResource;
}

export interface IClusterPodInfo {
  name: string;
  CPU: string;
  Mem: string;
  IP: string;
  createTime: string;
  status: string;
}

export interface IClusterNodeDetail {
  name: string;
  role: string;
  isReady: boolean;
  time: string;
  address: string;
  os: string;
  osVersion: string;
  arch: string;
  kubeletVersion: string;
  containerRuntimeVersion: string;
  pods: IClusterPodInfo[];
}

export const apiGetAdminNodes = () =>
  instance.get<
    IResponse<{
      rows: IClusterNodeInfo[];
    }>
  >(VERSION + "/admin/nodes");

export const apiGetAdminNodeDetail = (name: string) =>
  instance.get<IResponse<IClusterNodeDetail>>(
    VERSION + `/admin/nodes/${name}/pod`,
  );
