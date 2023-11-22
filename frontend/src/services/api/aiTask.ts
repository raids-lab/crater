import { KubernetesResource } from "@/utils/resource";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface IAiTask {
  id: number;
  taskName: string;
  userName: string;
  namespace: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  image: string;
  resourceRequest: string;
  workingDir: string;
  ShareDirs: string[];
  command: string;
  args: string;
  slo: number;
  status: string;
  isDeleted: false;
  profileStatus: number;
  profileStat: string;
  estimatedTime: number;
  scheduleInfo: string;
}

export interface AiTask {
  id: number;
  taskName: string;
  userName: string;
  namespace: string;
  taskType: string;
  createdAt: string;
  updatedAt: string;
  admittedAt: string;
  startedAt: string;
  image: string;
  resourceRequest: KubernetesResource;
  workingDir: string;
  ShareDirs: string[];
  command: string;
  args: string;
  slo: number;
  status: string;
  isDeleted: false;
  profileStatus: number;
  profileStat: unknown;
  estimatedTime: number;
  scheduleInfo: string;
}

export const convertAiTask = (task: IAiTask): AiTask => {
  return {
    ...task,
    resourceRequest: JSON.parse(task.resourceRequest) as KubernetesResource,
    profileStat: JSON.parse(task.profileStat),
  };
};

export interface ITaskCreate {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: KubernetesResource;
  image: string;
  workingDir: string;
  shareDirs: {
    [key: string]: string;
  };
  command: string;
  args: {
    [key: string]: string;
  };
}

export const apiAiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/aitask/create",
    task,
  );
  return response.data;
};

export const apiAiTaskDelete = async (taskID: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/aitask/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiAiTaskList = () =>
  instance.get<
    IResponse<{
      Tasks: IAiTask[];
    }>
  >(VERSION + "/aitask/list");

export const apiAiTaskQuota = () =>
  instance.get<
    IResponse<{
      hard: KubernetesResource;
      hardUsed: KubernetesResource;
      softUsed: KubernetesResource;
    }>
  >(VERSION + "/aitask/getQuota");

export const apiAiTaskGet = (taskID: number) =>
  instance.get<IResponse<IAiTask>>(VERSION + "/aitask/get", {
    params: {
      taskID,
    },
  });
