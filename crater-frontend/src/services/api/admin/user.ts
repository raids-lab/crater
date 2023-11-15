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

export const apiAdminUserList = () =>
  instance.get<IAdminUserListResponse>(VERSION + "/admin/listUser");

type IAdminUserDelete = {
  userName: string;
};

type IAdminUserDeleteResponse = IResponse<string>;

export const apiAdminUserDelete = (userName: string) =>
  instance.post<IAdminUserDeleteResponse>(VERSION + "/admin/deleteUser", {
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

type IAdminUserUpdateResponse = IResponse<string>;

export const apiAdminUserUpdate = (data: IAdminUserUpdate) =>
  instance.post<IAdminUserUpdateResponse>(VERSION + "/admin/updateQuota", data);
