import {
  ITaskCreate,
  ITaskCreateResponse,
  ITaskDeleteResponse,
  ITaskListResponse,
} from "../types";
import instance, { VERSION } from "../axios";

export const apiTaskCreate = async (task: ITaskCreate) => {
  const response = await instance.post<ITaskCreateResponse>(
    VERSION + "/task/create",
    task,
  );
  return response.data;
};

export const apiTaskDelete = async (taskID: number) => {
  const response = await instance.post<ITaskDeleteResponse>(
    VERSION + "/task/delete",
    {
      taskID,
    },
  );
  return response.data;
};

export const apiTaskList = async () =>
  instance.get<ITaskListResponse>(VERSION + "/task/list");
