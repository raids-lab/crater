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
  isVcjob: string;
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
}

export interface IClusterNodePods {
  pods: IClusterPodInfo[];
}

// GPU 信息接口定义
export interface IClusterNodeGPU {
  name: string;
  haveGPU: boolean;
  gpuCount: number;
  gpuUtil: Record<string, number>;
  relateJobs: string[];
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
  instance.get<IResponse<IClusterNodePods>>(VERSION + `/nodes/${name}/pods`);

// 获取节点的 GPU 详情
export const apiGetNodeGPU = (name: string) =>
  instance.get<IResponse<IClusterNodeGPU>>(VERSION + `/nodes/${name}/gpu`);
