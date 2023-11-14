import instance, { VERSION } from "../axios";

export interface ITaskDelete {
  taskID: number;
}

export interface ITaskDeleteResponse {
  data: string;
  error: string;
  status: boolean;
}

export interface ITaskListResponse {
  data: {
    Tasks: ITask[];
  };
}

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

export interface ITaskCreateResponse {
  data: string;
  error: string;
  status: boolean;
}

export const apiAiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<ITaskCreateResponse>(
    VERSION + "/aitask/create",
    task,
  );
  return response.data;
};

export const apiAiTaskDelete = async (taskID: number) => {
  const response = await instance.post<ITaskDeleteResponse>(
    VERSION + "/aitask/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiAiTaskList = async () =>
  instance.get<ITaskListResponse>(VERSION + "/aitask/list");

export const apiAiTaskQuota = async () =>
  instance.get(VERSION + "/aitask/getQuota");
