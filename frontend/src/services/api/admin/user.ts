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
  >(VERSION + "/admin/listUser");

type IAdminUserDelete = {
  userName: string;
};

export const apiAdminUserDelete = (userName: string) =>
  instance.post<IResponse<string>>(VERSION + "/admin/deleteUser", {
    userName,
  } as IAdminUserDelete);

// "/v1/admin/updateQuota"
// {
//   "userName": "zkr",
//   "hardQuota": {
//       "cpu": 40,
//       "memory": "40Gi",
//       "nvidia.com/gpu": 4
//   }
// }
export interface IAdminUserUpdate {
  userName: string;
  hardQuota: KubernetesResource;
}

export const apiAdminUserUpdateQuota = (data: IAdminUserUpdate) =>
  instance.post<IResponse<string>>(VERSION + "/admin/updateQuota", data);

export const apiAdminUserUpdateRole = (userName: string, role: string) =>
  instance.post<IResponse<string>>(VERSION + "/admin/updateRole", {
    userName,
    role,
  });
