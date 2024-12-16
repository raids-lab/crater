import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { K8sResources } from "@/utils/resource";

export enum NodeType {
  Hygon = "hygon",
  Shenwei = "shenwei",
  Yitian = "yitian",
}

export interface IClusterNodeInfo {
  type: NodeType;
  name: string;
  role: string;
  labels: Record<string, string>;
  isReady: string;
  taint: string;
  capacity: K8sResources;
  allocated: K8sResources;
}

export interface IOwnerReference {
  apiVersion: string;
  kind: string;
  name: string;
  uid: string;
}

export interface IClusterPodInfo {
  // from backend
  name: string;
  namespace: string;
  ownerReference: IOwnerReference[];
  ip: string;
  createTime: string;
  status: string;
  resources: Record<string, string>;
  // added by frontend
  type?: string;
}

export interface IClusterNodeDetail {
  name: string;
  role: string;
  isReady: string;
  time: string;
  address: string;
  os: string;
  osVersion: string;
  arch: string;
  kubeletVersion: string;
  containerRuntimeVersion: string;
}

// GPU 信息接口定义
export interface IClusterNodeGPU {
  name: string;
  haveGPU: boolean;
  gpuCount: number;
  gpuUtil: Record<string, number>;
  relateJobs: string[];
  gpuMemory: string;
  gpuArch: string;
  gpuDriver: string;
  cudaVersion: string;
  gpuProduct: string;
}

export const apiGetNodes = () =>
  instance.get<
    IResponse<{
      rows: IClusterNodeInfo[];
    }>
  >(VERSION + "/nodes");

export const apiGetNodeDetail = (name: string) =>
  instance.get<IResponse<IClusterNodeDetail>>(VERSION + `/nodes/${name}`);

export const apiGetNodePods = (name: string) =>
  instance.get<IResponse<IClusterPodInfo[]>>(VERSION + `/nodes/${name}/pods`);

// 获取节点的 GPU 详情
export const apiGetNodeGPU = (name: string) =>
  instance.get<IResponse<IClusterNodeGPU>>(VERSION + `/nodes/${name}/gpu`);
