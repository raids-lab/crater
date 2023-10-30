import { ICreateTask, ICreateTaskResponse, ITaskListResponse } from "./types";
import useAxios, { VERSION } from "./useAxios";

const useApiTask = () => {
  const { instance } = useAxios();

  const apiCreateTask = async (task: ICreateTask) => {
    const response = await instance.post<ICreateTaskResponse>(
      VERSION + "/task/create",
      task,
    );
    return response.data;
  };

  const apiGetTaskList = async () =>
    instance.get<ITaskListResponse>(VERSION + "/task/list");

  return { apiCreateTask, apiGetTaskList };
};

export default useApiTask;
