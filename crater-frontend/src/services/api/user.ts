import { Role } from "./auth";
import { ProjectStatus } from "./account";
import instance, { VERSION } from "../axios";
import { IResponse } from "../types";

// type UserDetailResp struct {
// 	ID        uint         `json:"id"`        // 用户ID
// 	Name      string       `json:"name"`      // 用户名称
// 	Nickname  string       `json:"nickname"`  // 用户昵称
// 	Role      model.Role   `json:"role"`      // 用户角色
// 	Status    model.Status `json:"status"`    // 用户状态
// 	CreatedAt time.Time    `json:"createdAt"` // 创建时间
// 	Teacher   *string      `json:"teacher"`   // 导师
// 	Group     *string      `json:"group"`     // 课题组
// 	Avatar    *string      `json:"avatar"`    // 头像
// }
export interface User {
  id: number; // 用户ID
  name: string; // 用户名称
  nickname: string; // 用户昵称
  role: Role; // 用户角色
  status: ProjectStatus; // 用户状态
  createdAt: string; // 创建时间
  teacher?: string; // 导师
  group?: string; // 课题组
  avatar?: string; // 头像
}

export const apiGetUser = (userName: string) =>
  instance.get<IResponse<User>>(`${VERSION}/users/${userName}`);

export const apiUserEmailVerified = async () => {
  const res = await instance.get<IResponse<boolean>>(
    `${VERSION}/users/email/verified`,
  );
  return res.data;
};
