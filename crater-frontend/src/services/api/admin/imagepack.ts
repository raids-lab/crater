import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";

// export const apiAdminImagepackCreate = async (imagepack: KanikoCreate) => {
//   const response = await instance.post<IResponse<string>>(
//     VERSION + "/admin/images/create",
//     imagepack,
//   );
//   return response.data;
// };

// export const apiAdminImagePackList = (type: number) =>
//   instance.get<IResponse<ImagePackListResponse>>(
//     `${VERSION}/admin/images/list?type=${type}`,
//   );

// export const apiAdminImagePackDelete = async (id: number) => {
//   const response = await instance.post<IResponse<string>>(
//     VERSION + "/admin/images/delete",
//     id,
//   );
//   return response.data;
// };

// export interface UpdateImagePublicStatusRequest {
//   id: number;
//   imagetype: number;
// }

export const apiAdminImagePublicStatusChange = async (id: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/change",
    id,
  );
  return response.data;
};
