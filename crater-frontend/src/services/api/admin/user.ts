import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";
import { Role } from "../auth";
import { ProjectStatus } from "../project";

export interface IProject {
  ID: number;
  name: string;
  description: string;
  namespace: string;
  status: ProjectStatus;
  isPersonal: boolean;
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

export interface IUser {
  id: number;
  name: string;
  role: Role;
  status: ProjectStatus;
}

export const apiAdminUserList = () =>
  instance.get<IResponse<IUser[]>>(`${VERSION}/admin/users`);

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

interface IGetAdminProjectList {
  pageIndex: number;
  pageSize: number;
  isPersonal: boolean;
  nameLike?: string;
  orderCol?: string;
  order?: "desc" | "asc";
  status?: ProjectStatus;
}

export const apiAdminProjectList = ({
  pageIndex,
  pageSize,
  isPersonal,
  nameLike,
  orderCol,
  order,
  status,
}: IGetAdminProjectList) =>
  instance.get<IResponse<{ rows: IProject[]; count: number }>>(
    `${VERSION}/admin/projects`,
    {
      params: {
        isPersonal,
        nameLike,
        orderCol,
        order,
        pageIndex,
        pageSize,
        status,
      },
    },
  );
