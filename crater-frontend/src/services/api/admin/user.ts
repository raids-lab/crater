import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";
import { Role } from "../auth";
import { ProjectStatus } from "../project";

export interface Quota {
  cpu: number;
  cpuReq: number;
  gpu: number;
  gpuMem: number;
  gpuMemReq: number;
  gpuReq: number;
  job: number;
  jobReq: number;
  mem: number;
  memReq: number;
  node: number;
  nodeReq: number;
  storage: number;
  extra: string;
}

export interface User {
  id: number;
  name: string;
  role: Role;
  status: ProjectStatus;
  quota: Quota;
}

export const apiAdminUserList = () =>
  instance.get<IResponse<User[]>>(`${VERSION}/admin/users`);

export const apiAdminUserDelete = (userName: string) =>
  instance.delete<IResponse<string>>(`${VERSION}/admin/users/${userName}`);

export interface IAdminUserUpdate {
  userName: string;
  hardQuota: KubernetesResource;
}

export const apiAdminUserUpdateQuota = (data: IAdminUserUpdate) =>
  instance.put<IResponse<string>>(
    `${VERSION}/admin/quotas/${data.userName}`,
    data,
  );

export const apiAdminUserUpdateRole = (userName: string, role: Role) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${userName}/role`, {
    role,
  });
