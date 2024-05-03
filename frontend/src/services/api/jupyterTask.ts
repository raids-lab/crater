import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export enum JobPhase {
  Pending = "Pending",
  Aborting = "Aborting",
  Aborted = "Aborted",
  Running = "Running",
  Restarting = "Restarting",
  Completing = "Completing",
  Completed = "Completed",
  Terminating = "Terminating",
  Terminated = "Terminated",
  Failed = "Failed",
}

export const apiJupyterList = () =>
  instance.get<IResponse<IJupyterResp[]>>(VERSION + "/vcjobs", {});

export interface IJupyterResp {
  name: string;
  jobName: string;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
}

export interface VolumeMount {
  subPath: string;
  mountPath: string;
}

export interface IJupyterCreate {
  name: string;
  resource: KubernetesResource;
  image: string;
  volumeMounts: VolumeMount[];
  nodeSelector: Record<string, string>;
}

export const apiJupyterCreate = async (task: IJupyterCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/vcjobs/jupyter",
    task,
  );
  return response.data;
};

export const apiJupyterDelete = async (jobName: string) => {
  const response = await instance.delete<IResponse<string>>(
    `${VERSION}/vcjobs/${jobName}`,
  );
  return response.data;
};

export const apiJTaskShareDirList = () =>
  instance.get<IResponse<string[]>>(VERSION + "/sharedir/list");

export const apiJTaskImageList = () =>
  instance.get<
    IResponse<{
      images: string[];
    }>
  >(VERSION + "/images/available");

export const apiJupyterTokenGet = (jobName: string) =>
  instance.get<
    IResponse<{
      baseURL: string;
      token: string;
    }>
  >(VERSION + `/vcjobs/${jobName}/token`);
