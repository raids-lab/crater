import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

export interface ITask {
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
  // resourceRequest: {
  //   [key: string]: string;
  // };
  resourceRequest: string;
  workingDir: string;
  ShareDirs: string[];
  command: string;
  args: string;
  slo: number;
  status: string;
  isDeleted: false;
  profiled: false;
  utilStat: string;
  estimatedTime: number;
  scheduleInfo: string;
}

export interface ITaskCreate {
  taskName: string;
  slo: number;
  taskType: string;
  resourceRequest: {
    [key: string]: string;
  };
  image: string;
  workingDir: string;
  shareDirs: string[];
  command: string;
  args: {
    [key: string]: string;
  };
}

export type ITaskCreateResponse = IResponse<string>;

export const apiAiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<ITaskCreateResponse>(
    VERSION + "/aitask/create",
    task,
  );
  return response.data;
};

export interface ITaskDelete {
  taskID: number;
}

export type ITaskDeleteResponse = IResponse<string>;

export const apiAiTaskDelete = async (taskID: number) => {
  const response = await instance.post<ITaskDeleteResponse>(
    VERSION + "/aitask/delete",
    {
      taskID,
    } as ITaskDelete,
  );
  return response.data;
};

export type ITaskListResponse = IResponse<{
  Tasks: ITask[];
}>;

export const apiAiTaskList = async () =>
  instance.get<ITaskListResponse>(VERSION + "/aitask/list");

export type ITaskQuotaResponse = IResponse<{
  hard: {
    cpu: number;
    memory: string;
    "nvidia.com/gpu": number;
  };
  hardUsed: {
    cpu: number;
    memory: string;
    "nvidia.com/gpu": number;
  };
  softUsed: {
    cpu: number;
    memory: string;
    "nvidia.com/gpu": number;
  };
}>;

export const apiAiTaskQuota = async () =>
  instance.get<ITaskQuotaResponse>(VERSION + "/aitask/getQuota");
