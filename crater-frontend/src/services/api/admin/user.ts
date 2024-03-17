import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { KubernetesResource } from "@/utils/resource";

export interface User {
  userID: number;
  userName: string;
  role: string;
  quotaHard: KubernetesResource;
  createdAt: string;
  updatedAt: string;
}

export const apiAdminUserList = () =>
  instance.get<
    IResponse<{
      Users: User[];
    }>
  >(`${VERSION}/admin/users`);

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

export const apiAdminUserUpdateRole = (userName: string, role: string) =>
  instance.put<IResponse<string>>(`${VERSION}/admin/users/${userName}`, {
    userName,
    role,
  });
