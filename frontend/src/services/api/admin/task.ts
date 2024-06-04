import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { IJupyterDetail } from "../vcjob";

export interface IVolcanoJobInfo {
  name: string;
  jobName: string;
  owner: string;
  jobType: string;
  queue: string;
  status: string;
  createdAt: string;
  startedAt: string;
  completedAt: string;
  nodes: string[];
  resources: Record<string, string>;
}

export const apiAdminTaskListByType = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/admin/vcjobs");

export const apiTaskListByType = () =>
  instance.get<IResponse<IVolcanoJobInfo[]>>(VERSION + "/vcjobs/all");

export const apiJupyterGetAdminDetail = (jobName: string) =>
  instance.get<IResponse<IJupyterDetail>>(
    `${VERSION}/admin/vcjobs/${jobName}/detail`,
  );
