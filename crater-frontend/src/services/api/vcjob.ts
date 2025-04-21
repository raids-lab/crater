import { K8sResources } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { globalJobUrl, store } from "@/utils/store";
import { ImageInfoResponse } from "./imagepack";
import { Event as KubernetesEvent } from "kubernetes-types/core/v1";
import { TerminatedState } from "./tool";

const JOB_URL = store.get(globalJobUrl);

export enum JobType {
  Jupyter = "jupyter",
  Pytorch = "pytorch",
  Tensorflow = "tensorflow",
  Custom = "custom",
  KubeRay = "kuberay",
  DeepSpeed = "deepspeed",
  OpenMPI = "openmpi",
}

export interface IJobInfo {
  name: string;
  jobName: string;
  owner: string;
  userInfo: IUserInfo;
  jobType: JobType;
  queue: string;
  status: JobPhase;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodes: string[];
  resources?: Record<string, string>;
  locked: boolean;
  permanentLocked: boolean;
  lockedTimestamp?: string;
}

export const apiAdminGetJobList = (days?: number) =>
  instance.get<IResponse<IJobInfo[]>>(`${VERSION}/admin/${JOB_URL}`, {
    params: { days },
  });

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
  Cancelled = "Cancelled",
  Init = "",
}

export enum JobStatus {
  NotStarted = "NotStarted",
  Running = "Running",
  Terminated = "Terminated",
  MetadataOnly = "MetadataOnly",
  Unknown = "Unknown",
}

export const getJobStateType = (phase: JobPhase): JobStatus => {
  // 作业还没有开始运行的状态
  const notStartedPhases = new Set([JobPhase.Pending, JobPhase.Init]);

  // 作业正在运行的状态
  const runningPhases = new Set([
    JobPhase.Running,
    JobPhase.Restarting,
    JobPhase.Completing,
    JobPhase.Aborting,
    JobPhase.Terminating,
  ]);

  // 作业终态
  const terminatedPhases = new Set([
    JobPhase.Completed,
    JobPhase.Failed,
    JobPhase.Terminated,
    JobPhase.Aborted,
  ]);

  // 仅保留元数据的状态
  const metadataOnlyPhases = new Set([
    JobPhase.Freed,
    JobPhase.Deleted,
    JobPhase.Cancelled,
  ]);

  if (notStartedPhases.has(phase)) {
    return JobStatus.NotStarted;
  }
  if (runningPhases.has(phase)) {
    return JobStatus.Running;
  }
  if (terminatedPhases.has(phase)) {
    return JobStatus.Terminated;
  }
  if (metadataOnlyPhases.has(phase)) {
    return JobStatus.MetadataOnly;
  }

  return JobStatus.Unknown;
};

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

// {
//   "cpu_usage_avg": 0.12826866,
//   "cpu_usage_max": 29.675838,
//   "cpu_usage_std": 1.0184886,
//   "cpu_mem_avg": 8204.886,
//   "cpu_mem_max": 102399.22,
//   "cpu_mem_std": 23043.736,
//   "gpu_util_avg": 0.0063490635,
//   "gpu_util_max": 0.76,
//   "gpu_util_std": 0.063120745,
//   "sm_active_avg": 0.0052751177,
//   "sm_active_max": 0.592845,
//   "sm_active_std": 0.0486276,
//   "sm_occupancy_avg": 0.0025021092,
//   "sm_occupancy_max": 0.280952,
//   "sm_occupancy_std": 0.023056554,
//   "dram_util_avg": 0.0029674373,
//   "dram_util_max": 0.376592,
//   "dram_util_std": 0.029283673,
//   "mem_copy_util_avg": 0.0039970037,
//   "mem_copy_util_max": 0.5,
//   "mem_copy_util_std": 0.040522728,
//   "pcie_tx_avg": 2.90641,
//   "pcie_tx_max": 355.19397,
//   "pcie_rx_avg": 24.02333,
//   "pcie_rx_max": 2997.0408,
//   "gpu_mem_max": 4899
// }
export interface ProfileData {
  cpu_request?: number; // 展示单位m
  cpu_limit?: number; // 展示单位m
  mem_request?: number; // 展示单位MB
  mem_limit?: number; // 展示单位MB

  cpu_usage_avg?: number; //展示数值
  cpu_usage_max?: number;
  cpu_usage_std?: number;

  cpu_mem_avg?: number; // 展示单位MB，不用进度条
  cpu_mem_max?: number; // 展示单位MB，不用进度条
  cpu_mem_std?: number; // 展示单位MB，不用进度条

  gpu_util_avg?: number; //上限1
  gpu_util_max?: number; //上限1
  gpu_util_std?: number;

  sm_active_avg?: number; //上限1
  sm_active_max?: number; //上限1
  sm_active_std?: number;

  sm_util_avg?: number; //上限1
  sm_util_max?: number;
  sm_util_std?: number;

  sm_occupancy_avg?: number; //上限1
  sm_occupancy_max?: number;
  sm_occupancy_std?: number;

  dram_util_avg?: number; //上限1
  dram_util_max?: number;
  dram_util_std?: number;

  dram_active_avg?: number;
  dram_active_max?: number;
  dram_active_std?: number;

  mem_copy_util_avg?: number; //上限1
  mem_copy_util_max?: number;
  mem_copy_util_std?: number;

  pcie_tx_avg?: number; // 展示数值 MB/s
  pcie_tx_max?: number;

  pcie_rx_avg?: number; // 展示数值 MB/s
  pcie_rx_max?: number;

  gpu_mem_total?: number;
  gpu_mem_max?: number;
  gpu_mem_avg?: number;
  gpu_mem_std?: number;

  tensor_active_avg?: number;
  tensor_active_max?: number;
  tensor_active_std?: number;

  fp64_active_avg?: number;
  fp64_active_max?: number;
  fp64_active_std?: number;

  fp32_active_avg?: number;
  fp32_active_max?: number;
  fp32_active_std?: number;

  fp16_active_avg?: number;
  fp16_active_max?: number;
  fp16_active_std?: number;
}

export interface IUserInfo {
  username: string;
  nickname: string;
}

export interface ScheduleData {
  imagePullTime: string;
  imageSize?: string;
}

export interface IJupyterDetail {
  name: string;
  namespace: string;
  username: string;
  nickname: string;
  userInfo: IUserInfo;
  jobName: string;
  jobType: JobType;
  retry: string;
  queue: string;
  status: JobPhase;
  resources?: Record<string, string>;
  profileData?: ProfileData;
  scheduleData?: ScheduleData;
  events?: KubernetesEvent[];
  terminatedStates?: TerminatedState[];
  createdAt: string;
  startedAt: string;
  completedAt: string;
}

export interface ISSHPortDetail {
  open: boolean;
  data: {
    IP: string;
    nodePort: number;
    username: string;
  };
}

// SSH 信息接口
export interface SSHInfo {
  ip: string;
  port: string;
}

export interface VolumeMount {
  subPath: string;
  mountPath: string;
}

export interface Forward {
  name: string;
  port: number;
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
  selectors?: NodeSelectorRequirement[];
  template: string;
  alertEnabled: boolean;
  forwards: Forward[];
}

export interface ITrainingCreate extends IJupyterCreate {
  shell?: string;
  command?: string;
  workingDir?: string;
  forwards: Forward[];
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
  selectors?: NodeSelectorRequirement[];
  alertEnabled: boolean;
  template?: string;
  forwards: Forward[];
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

export const apiJobToggleKeepForAdmin = async (jobName: string) => {
  const response = await instance.put<IResponse<string>>(
    `${VERSION}/admin/operations/keep/${jobName}`,
  );
  return response.data;
};

export const apiJobUnlock = async (jobName: string) => {
  const response = await instance.put<IResponse<string>>(
    `${VERSION}/admin/operations/clear/locktime`,
    { name: jobName },
  );
  return response.data;
};

export const apiJobLock = async (params: object) => {
  const response = await instance.put<IResponse<string>>(
    `${VERSION}/admin/operations/add/locktime`,
    params,
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
      fullURL: string;
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

// 开启 SSH 端口
export const apiOpenSSH = (jobName: string) =>
  instance.post<IResponse<SSHInfo>>(`${VERSION}/${JOB_URL}/${jobName}/ssh`);

export const apiJobScheduleAdmin = async () => {
  const response = await instance.get<IResponse<string>>(
    `${VERSION}/admin/operations/cronjob`,
  );
  return response.data;
};
export const apiJobScheduleChangeAdmin = async (schedule: object) => {
  const response = await instance.put<IResponse<string>>(
    `${VERSION}/admin/operations/cronjob`,
    schedule,
  );
  return response.data;
};
