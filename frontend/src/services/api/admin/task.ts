import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { IAiTask } from "../aiTask";

export const apiAdminTaskListByType = (
  taskType: string,
  pageSize?: number,
  pageIndex?: number,
) =>
  instance.get<
    IResponse<{
      rows: IAiTask[];
      rowCount: number;
    }>
  >(VERSION + "/admin/tasks", {
    params: {
      taskType,
      pageSize,
      pageIndex,
    },
  });
