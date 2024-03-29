import instance, { VERSION } from "@/services/axios";
import { IResponse } from "@/services/types";
import { ImagePackCreate, ImagePackInfoResponse } from "../imagepack";

export const apiAdminImagepackCreate = async (imagepack: ImagePackCreate) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/create",
    imagepack,
  );
  return response.data;
};

export const apiAdminPublicImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(
    VERSION + "/admin/images/public",
  );

export const apiAdminPersonalImagePackList = () =>
  instance.get<IResponse<ImagePackInfoResponse[]>>(
    VERSION + "/admin/images/personal",
  );

export const apiAdminImagePackDelete = async (id: number) => {
  const response = await instance.post<IResponse<string>>(
    VERSION + "/admin/images/delete",
    {
      id,
    },
  );
  return response.data;
};
