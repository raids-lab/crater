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

type IAdminUserListResponse = IResponse<{
  Users: User[];
}>;

export const apiAdminUserList = async () =>
  instance.get<IAdminUserListResponse>(VERSION + "/admin/listUser");

type IAdminUserDelete = {
  userName: string;
};

type IAdminUserDeleteResponse = IResponse<string>;

export const apiAdminUserDelete = async (userName: string) =>
  await instance.post<IAdminUserDeleteResponse>(VERSION + "/admin/deleteUser", {
    userName,
  } as IAdminUserDelete);
