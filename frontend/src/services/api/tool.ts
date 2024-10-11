import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface ContainerInfo {
  name: string;
  image: string;
  state: {
    waiting?: {
      reason: string;
      message: string;
    };
    running?: {
      startedAt: string;
    };
    terminated?: {
      exitCode: number;
      signal: number;
      reason: string;
      message: string;
      startedAt: string;
      finishedAt: string;
      containerID: string;
    };
  };
  resources?: Record<string, string>;
  restartCount: number;
  isInitContainer: boolean;
}

export interface ContainerStatusResponse {
  containers: ContainerInfo[];
}

// http://localhost:8092/v1/pods/jupyter-liyilong-3885b-default0-0/containers
export const apiGetPodContainers = (namespace?: string, podName?: string) =>
  instance.get<IResponse<ContainerStatusResponse>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/containers`,
  );

// http://localhost:8092/v1/pods/jupyter-liyilong-3885b-default0-0/containers/jupyter/log
// PodContainerLogQueryReq struct {
//   // from query
//   TailLines  *int64 `form:"tailLines" binding:"required"`
//   Timestamps bool   `form:"timestamps" binding:"required"`
//   Follow     bool   `form:"follow" binding:"required"`
// }
export interface PodContainerLogQueryReq {
  tailLines: number;
  timestamps: boolean;
  follow: boolean;
}

export const apiGetPodContainerLog = (
  namespace?: string,
  podName?: string,
  containerName?: string,
  query?: PodContainerLogQueryReq,
) => {
  return instance.get<IResponse<string>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/containers/${containerName}/log`,
    {
      params: query,
    },
  );
};
