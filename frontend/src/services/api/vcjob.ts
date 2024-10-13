import { KubernetesResourceList } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { globalJobUrl, store } from "@/utils/store";

export enum JobType {
  Custom = "training",
  Jupyter = "jupyter",
  Tensorflow = "tensorflow",
  Pytorch = "pytorch",
  WebIDE = "webide",
  Ray = "ray",
  DeepSpeed = "deepspeed",
  OpenMPI = "openmpi",
}

export interface IJobInfo {
  name: string;
  jobName: string;
  owner: string;
  jobType: JobType;
  queue: string;
  status: string;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodes: string[];
  resources: Record<string, string>;
}

export const apiAdminTaskListByType = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/admin/${JOB_URL}`);

export const apiJobAllList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}/all`);

export const apiJupyterGetAdminDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/admin/${JOB_URL}/${jobName}/detail`,
  );

const JOB_URL = store.get(globalJobUrl);

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
  Unknown = "",
}

export const apiJobBatchList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}`);

export const apiJobInteractiveList = () =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/${JOB_URL}`);

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
  resource: KubernetesResourceList;
  image: string;
  volumeMounts: VolumeMount[];
  envs: Env[];
  useTensorBoard: boolean;
  selectors?: NodeSelectorRequirement[];
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
    resource: KubernetesResourceList;
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

export const apiJobGetDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/${JOB_URL}/${jobName}/detail`,
  );

export const apiJupyterYaml = (jobName: string) =>
  instance.get<IResponse<string>>(`${VERSION}/${JOB_URL}/${jobName}/yaml`);

export const apiJTaskImageList = (imageTaskType: number) =>
  instance.get<
    IResponse<{
      images: string[];
    }>
  >(`${VERSION}/images/available?type=${imageTaskType}`);

export const apiJupyterTokenGet = (jobName: string) =>
  instance.get<
    IResponse<{
      baseURL: string;
      token: string;
      podName: string;
      namespace: string;
    }>
  >(`${VERSION}/${JOB_URL}/${jobName}/token`);
