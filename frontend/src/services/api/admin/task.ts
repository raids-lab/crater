import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { IAiTask } from "../aiTask";

export const apiAdminTaskListByType = (taskType: string) =>
  instance.get<
    IResponse<{
      Tasks: IAiTask[];
    }>
  >(VERSION + "/admin/listByTaskType", {
    params: {
      taskType,
    },
  });
