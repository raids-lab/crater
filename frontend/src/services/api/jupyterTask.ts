import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";
import { showErrorToast } from "@/utils/toast";
import { JobStatus } from "@/pages/Portal/Job/Ai/statuses";

export interface IJupyterTask {
  ID: number;
  name: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  finishAt: string;
  duration: number;
  jct: number;
  image: string;
  resourceRequest: string;
  //workingDir: string;
  ShareDirs: string[];
  //command: string;
  //args: string;
  //slo: number;
  status: JobStatus;
  jobName: string;
  isDeleted: boolean;
  estimatedTime: number;
  scheduleInfo: string;
}
export interface JupyterTask {
  id: number;
  taskName: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  finishAt: string;
  duration: number;
  jct: number;
  image: string;
  resourceRequest: KubernetesResource;
  //workingDir: string;
  ShareDirs: string[];
  //command: string;
  //args: string;
  //slo: number;
  status: number;
  jobName: string;
  isDeleted: boolean;
  estimatedTime: number;
  scheduleInfo: string;
}

export const convertJTask = (task: IJupyterTask): JupyterTask => {
  try {
    const jTaskINfo: JupyterTask = {
      ...task,
      id: task.ID,
      taskName: task.name,
      taskType: task.taskType,
      resourceRequest: JSON.parse(task.resourceRequest) as KubernetesResource,
    };
    return jTaskINfo;
  } catch (e) {
    showErrorToast(e);
    return task as unknown as JupyterTask;
  }
};
export interface IJupyterTaskCreate {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: KubernetesResource;
  image: string;
  workingDir: string;
  shareDirs: {
    [key: string]: {
      [key: string]: string;
    }[];
  };
  command: string;
  gpuModel: string;
  schedulerName: string;
}

export const apiJTaskCreate = async (task: IJupyterTaskCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/aijobs",
    task,
  );
  return response.data;
};

export const apiJTaskDelete = async (taskID: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/jupyter/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiJupyterJobList = (
  taskType: string,
  pageSize: number,
  pageIndex: number,
) =>
  instance.get<
    IResponse<{
      rows: IJupyterTask[];
    }>
  >(VERSION + "/aijobs", {
    params: {
      taskType,
      pageSize,
      pageIndex,
    },
  });

export const apiJTaskQuota = () =>
  instance.get<
    IResponse<{
      hard: KubernetesResource;
      hardUsed: KubernetesResource;
      softUsed: KubernetesResource;
    }>
  >(VERSION + "/jupyter/getQuota");

// export interface IPortTokenResponse {
//   port: number;
//   token: string;
// }

export const apiJTaskShareDirList = () =>
  instance.get<IResponse<string[]>>(VERSION + "/sharedir/list");

export const apiJTaskImageList = () =>
  instance.get<
    IResponse<{
      images: string[];
    }>
  >(VERSION + "/images/available");

export const apiJTaskGetPortToken = (taskID: number) =>
  instance.get<
    IResponse<{
      port: number;
      token: string;
    }>
  >(VERSION + "/jupyter/getToken", {
    params: {
      taskID,
    },
  });
