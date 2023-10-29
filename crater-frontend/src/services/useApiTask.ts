import { ITaskAttr, ITaskAttrResponse } from "./types";
import useAxios, { VERSION } from "./useAxios";

const useApiTask = () => {
  const { instance } = useAxios();

  const apiCreateTask = async (task: ITaskAttr) => {
    const response = await instance.post<ITaskAttrResponse>(
      VERSION + "/task/create",
      task,
    );
    return response.data;
  };

  const apiGetTaskList = async () => {};

  return { apiCreateTask, apiGetTaskList };
};

export default useApiTask;
