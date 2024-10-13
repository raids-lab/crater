import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";
import { Role } from "../auth";
import { ProjectStatus } from "../account";

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
