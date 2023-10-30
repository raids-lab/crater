import { ICreateTask, ICreateTaskResponse, ITaskListResponse } from "../types";
import instance, { VERSION } from "../axios";

export const apiTaskCreate = async (task: ICreateTask) => {
  const response = await instance.post<ICreateTaskResponse>(
    VERSION + "/task/create",
    task,
  );
  return response.data;
};

export const apiTaskList = async () =>
  instance.get<ITaskListResponse>(VERSION + "/task/list");
