import {
  ITaskCreate,
  ITaskCreateResponse,
  ITaskDeleteResponse,
  ITaskListResponse,
} from "../types";
import instance, { VERSION } from "../axios";

export const apiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<ITaskCreateResponse>(
    VERSION + "/aitask/create",
    task,
  );
  return response.data;
};

export const apiTaskDelete = async (taskID: number) => {
  const response = await instance.post<ITaskDeleteResponse>(
    VERSION + "/aitask/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiTaskList = async () =>
  instance.get<ITaskListResponse>(VERSION + "/aitask/list");
