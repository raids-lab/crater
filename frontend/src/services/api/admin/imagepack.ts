import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import {
  ImageDeleteRequest,
  ImagePackCreate,
  ImagePackListResponse,
} from "../imagepack";

export const apiAdminImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/create",
    imagepack,
  );
  return response.data;
};

export const apiAdminImagePackList = (type: number) =>
  instance.get<IResponse<ImagePackListResponse>>(
    `${VERSION}/admin/images/list?type=${type}`,
  );

export const apiAdminImagePackDelete = async (req: ImageDeleteRequest) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/delete",
    req,
  );
  return response.data;
};

export interface UpdateImagePublicStatusRequest {
  id: number;
  imagetype: number;
}

export const apiAdminImagePublicStatusChange = async (
  req: UpdateImagePublicStatusRequest,
) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/change",
    req,
  );
  return response.data;
};
