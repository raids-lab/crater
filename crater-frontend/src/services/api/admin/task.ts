import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";

export interface IVolcanoJobInfo {
  name: string;
  jobName: string;
  userName: string;
  jobType: string;
  queue: string;
  status: string;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodeName: string;
  resource: string;
}
export const apiAdminTaskListByType = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/admin/vcjobs");

export const apiTaskListByType = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/vcjobs/all");
