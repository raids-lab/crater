import { KubernetesResourceList } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { IVolcanoJobInfo } from "./admin/task";

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

export const apiJobAllList = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/vcjobs", {});

export const apiJobBatchList = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/vcjobs", {});

export const apiJobInteractiveList = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/vcjobs", {});

export interface PodDetail {
  name: string;
  nodename: string;
  ip: string;
  port: string;
  resource: string;
  status: string;
}
export interface IJupyterDetail {
  name: string;
  namespace: string;
  username: string;
  jobName: string;
  retry: string;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
  runtime: string;
  podDetails: PodDetail[];
  useTensorBoard: boolean;
}

export interface VolumeMount {
  subPath: string;
  mountPath: string;
}

export interface Env {
  name: string;
  value: string;
}

export interface IJupyterCreate {
  name: string;
  resource: KubernetesResourceList;
  image: string;
  volumeMounts: VolumeMount[];
  envs: Env[];
  useTensorBoard: boolean;
}

export interface ITrainingCreate extends IJupyterCreate {
  command: string;
  workingDir: string;
}

export interface ITensorflowCreate {
  name: string;
  tasks: {
    name: string;
    replicas: number;
    resource: KubernetesResourceList;
    image: string;
    command: string;
    workingDir: string;
    ports: {
      name: string;
      port: number;
    }[];
  }[];
  volumeMounts: VolumeMount[];
  envs: Env[];
  useTensorBoard: boolean;
}

export const apiJupyterCreate = async (task: IJupyterCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/vcjobs/jupyter",
    task,
  );
  return response.data;
};

export const apiTrainingCreate = async (task: ITrainingCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/vcjobs/training",
    task,
  );
  return response.data;
};

export const apiTensorflowCreate = async (task: ITensorflowCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/vcjobs/tensorflow",
    task,
  );
  return response.data;
};

export const apiJobDelete = async (jobName: string) => {
  const response = await instance.delete<IResponse<string>>(
    `${VERSION}/vcjobs/${jobName}`,
  );
  return response.data;
};

export const apiJupyterGetDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/vcjobs/${jobName}/detail`,
  );
export interface Logs {
  [key: string]: string;
}
interface GetJobLogResp {
  logs: Logs;
}

export const apiJupyterLog = (jobName: string) =>
  instance.get<IResponse<GetJobLogResp>>(VERSION + `/vcjobs/${jobName}/log`);

export const apiJupyterYaml = (jobName: string) =>
  instance.get<IResponse<string>>(VERSION + `/vcjobs/${jobName}/yaml`);

export const apiJTaskShareDirList = () =>
  instance.get<IResponse<string[]>>(VERSION + "/sharedir/list");

export const apiJTaskImageList = () =>
  instance.get<
    IResponse<{
      images: string[];
    }>
  >(VERSION + "/images/available?type=1");

export const apiJupyterTokenGet = (jobName: string) =>
  instance.get<
    IResponse<{
      baseURL: string;
      token: string;
    }>
  >(VERSION + `/vcjobs/${jobName}/token`);
