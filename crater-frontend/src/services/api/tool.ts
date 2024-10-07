import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface ContainerStatus {
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
  resources: Record<string, string>;
  restartCount: number;
  isInitContainer: boolean;
}

export interface ContainerStatusResponse {
  containers: ContainerStatus[];
}

// http://localhost:8092/v1/pods/jupyter-liyilong-3885b-default0-0/containers
export const apiGetPodContainers = (podName: string) =>
  instance.get<IResponse<ContainerStatusResponse>>(
    `${VERSION}/pods/${podName}/containers`,
  );
