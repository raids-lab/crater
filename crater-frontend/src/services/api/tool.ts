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

export interface PodIngress {
  name: string;
  port: number;
  prefix: string;
}

export interface PodIngressMgr {
  name: string;
  port: number;
}

export interface PodIngressesList {
  ingresses: PodIngress[];
}

export const apiGetPodIngresses = (namespace: string, podName: string) =>
  instance.get<IResponse<PodIngressesList>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/ingresses`,
  );

export const apiCreatePodIngress = (
  namespace: string,
  podName: string,
  ingressMgr: PodIngressMgr,
) =>
  instance.post<IResponse<PodIngress>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/ingresses`,
    ingressMgr,
  );

export const apiDeletePodIngress = (
  namespace: string,
  podName: string,
  ingressMgr: PodIngressMgr,
) =>
  instance.delete<IResponse<string>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/ingresses`,
    {
      data: ingressMgr,
    },
  );

export interface PodNodeport {
  name: string;
  containerPort: number;
  address: string;
  nodePort: number;
}

export interface PodNodeportMgr {
  name: string;
  containerPort: number;
}

export interface PodNodeportsList {
  nodeports: PodNodeport[];
}

export const apiGetPodNodeports = (namespace: string, podName: string) =>
  instance.get<IResponse<PodNodeportsList>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/nodeports`,
  );

export const apiCreatePodNodeport = (
  namespace: string,
  podName: string,
  nodeportMgr: PodNodeportMgr,
) =>
  instance.post<IResponse<PodIngress>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/nodeports`,
    nodeportMgr,
  );

export const apiDeletePodNodeport = (
  namespace: string,
  podName: string,
  nodeportMgr: PodNodeportMgr,
) =>
  instance.delete<IResponse<string>>(
    `${VERSION}/namespaces/${namespace}/pods/${podName}/nodeports`,
    {
      data: nodeportMgr,
    },
  );
