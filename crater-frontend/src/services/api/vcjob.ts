import { K8sResources } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { globalJobUrl, store } from "@/utils/store";
import { ImageInfoResponse } from "./imagepack";
import { Event as KubernetesEvent } from "kubernetes-types/core/v1";

const JOB_URL = store.get(globalJobUrl);

export enum JobType {
  Jupyter = "jupyter",
  WebIDE = "webide",
  Pytorch = "pytorch",
  Tensorflow = "tensorflow",
  KubeRay = "kuberay",
  DeepSpeed = "deepspeed",
  OpenMPI = "openmpi",
  Custom = "custom",
}

export interface IJobInfo {
  name: string;
  jobName: string;
  owner: string;
  jobType: JobType;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodes: string[];
  resources?: Record<string, string>;
  keepWhenLowUsage: boolean;
}

export const apiAdminGetJobList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/admin/${JOB_URL}`);

export const apiAdminGetJobDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/admin/${JOB_URL}/${jobName}/detail`,
  );

export const apiJobAllList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}/all`);

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
  Deleted = "Deleted",
  Freed = "Freed",
  Init = "",
}

export const apiJobBatchList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}`);

export const apiJobInteractiveList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}`);

export interface PodDetail {
  name: string;
  namespace: string;
  nodename: string;
  ip: string;
  port: string;
  resource?: Record<string, string>;
  phase: string;
}

export interface IJupyterDetail {
  name: string;
  namespace: string;
  username: string;
  jobName: string;
  jobType: JobType;
  retry: string;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
  completedAt: string;
}

export interface VolumeMount {
  subPath: string;
  mountPath: string;
}

export interface Env {
  name: string;
  value: string;
}

export type NodeSelectorOperator =
  | "In"
  | "NotIn"
  | "Exists"
  | "DoesNotExist"
  | "Gt"
  | "Lt";

export interface NodeSelectorRequirement {
  key: string;
  operator: NodeSelectorOperator;
  values?: string[];
}

export interface IJupyterCreate {
  name: string;
  resource: K8sResources;
  image: string;
  volumeMounts: VolumeMount[];
  envs: Env[];
  useTensorBoard: boolean;
  selectors?: NodeSelectorRequirement[];
  template: string;
}

export interface ITrainingCreate extends IJupyterCreate {
  command?: string;
  workingDir?: string;
}

export interface ISparseCreate extends ITrainingCreate {
  runningType?: string;
  params?: number;
  macs?: number;
  batchSize?: number;
  vocabularySize?: number[];
  embeddingDim?: number[];
  replicas?: number;
}

export interface ITensorflowCreate {
  name: string;
  tasks: {
    name: string;
    replicas: number;
    resource: K8sResources;
    image: string;
    command?: string;
    workingDir?: string;
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
    `${VERSION}/${JOB_URL}/jupyter`,
    task,
  );
  return response.data;
};

export const apiTrainingCreate = async (task: ITrainingCreate) => {
  const response = await instance.post<IResponse<string>>(
    `${VERSION}/${JOB_URL}/training`,
    task,
  );
  return response.data;
};

export const apiSparseCreate = async (task: ISparseCreate) => {
  const response = await instance.post<IResponse<string>>(
    `${VERSION}/${JOB_URL}/training`,
    task,
  );
  return response.data;
};

export const apiTensorflowCreate = async (task: ITensorflowCreate) => {
  const response = await instance.post<IResponse<string>>(
    `${VERSION}/${JOB_URL}/tensorflow`,
    task,
  );
  return response.data;
};

export const apiPytorchCreate = async (task: ITensorflowCreate) => {
  const response = await instance.post<IResponse<string>>(
    `${VERSION}/${JOB_URL}/pytorch`,
    task,
  );
  return response.data;
};

export const apiJobDelete = async (jobName: string) => {
  const response = await instance.delete<IResponse<string>>(
    `${VERSION}/${JOB_URL}/${jobName}`,
  );
  return response.data;
};

export const apiJobDeleteForAdmin = async (jobName: string) => {
  const response = await instance.delete<IResponse<string>>(
    `${VERSION}/admin/${JOB_URL}/${jobName}`,
  );
  return response.data;
};

export const apiJobKeepForAdmin = async (jobName: string) => {
  const response = await instance.put<IResponse<string>>(
    `${VERSION}/admin/operations/keep/${jobName}`,
  );
  return response.data;
};

export const apiJobGetDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/${JOB_URL}/${jobName}/detail`,
  );

export const apiJobGetPods = (jobName: string) =>
  instance.get<IResponse<PodDetail[]>>(`${VERSION}/${JOB_URL}/${jobName}/pods`);

export const apiJobGetYaml = (jobName: string) =>
  instance.get<IResponse<string>>(`${VERSION}/${JOB_URL}/${jobName}/yaml`);

export const apiJobGetEvent = (jobName: string) =>
  instance.get<IResponse<KubernetesEvent[]>>(
    `${VERSION}/${JOB_URL}/${jobName}/event`,
  );

export const apiJobTemplate = (jobName: string) =>
  instance.get<IResponse<string>>(`${VERSION}/${JOB_URL}/${jobName}/template`);

export const apiJTaskImageList = (imageTaskType: string) =>
  instance.get<IResponse<{ images: ImageInfoResponse[] }>>(
    `${VERSION}/images/available?type=${imageTaskType}`,
  );

export const apiJupyterTokenGet = (jobName: string) =>
  instance.get<
    IResponse<{
      baseURL: string;
      token: string;
      podName: string;
      namespace: string;
    }>
  >(`${VERSION}/${JOB_URL}/${jobName}/token`);

// @Router /v1/vcjobs/jupyter/{name}/snapshot [post]
export const apiJupyterSnapshot = async (jobName: string) => {
  const response = await instance.post<IResponse<string>>(
    `${VERSION}/${JOB_URL}/jupyter/${jobName}/snapshot`,
  );
  return response.data;
};
