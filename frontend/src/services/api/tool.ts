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

export interface PodContainerLogQueryReq {
  timestamps: boolean;
  follow: boolean;
  previous: boolean;
  tailLines?: number;
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
